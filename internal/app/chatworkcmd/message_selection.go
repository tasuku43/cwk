package chatworkcmd

import (
	"fmt"
	"sort"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// assembleMessageWindow resolves one complete bounded source window before
// selecting newest primary messages. Sender matching forms one OR predicate,
// period membership forms an AND predicate, StartIndex and Count select the
// resulting candidates by typed send time without reordering output, and reply
// context is exactly one hop from the selected anchors.
func assembleMessageWindow(messages []chatwork.Message, filter chatwork.MessageFilter) ([]chatwork.Message, *chatwork.MessageSelection, error) {
	resolvedSource, err := ResolveMessageRelations(messages)
	if err != nil {
		return nil, nil, err
	}
	if len(filter.Senders) == 0 && filter.Period == (chatwork.MessagePeriod{}) && filter.StartIndex == 0 && filter.Count == 0 {
		return resolvedSource, nil, nil
	}

	senders := make(map[string]struct{}, len(filter.Senders))
	for _, sender := range filter.Senders {
		senders[sender.Value] = struct{}{}
	}
	candidates := make([]bool, len(resolvedSource))
	included := make([]bool, len(resolvedSource))
	messageIndex := make(map[string]int, len(resolvedSource))
	candidateIndexes := make([]int, 0, len(resolvedSource))
	for index, message := range resolvedSource {
		messageIndex[message.Ref.Value] = index
		matches := len(senders) == 0
		if !matches {
			_, matches = senders[message.Sender.Ref.Value]
		}
		matches = matches && filter.Period.Contains(message.SendTime)
		if matches {
			candidates[index] = true
			candidateIndexes = append(candidateIndexes, index)
		}
	}

	anchors := append([]bool(nil), candidates...)
	if filter.StartIndex > 0 || filter.Count > 0 {
		sort.Slice(candidateIndexes, func(left, right int) bool {
			leftIndex, rightIndex := candidateIndexes[left], candidateIndexes[right]
			if resolvedSource[leftIndex].SendTime == resolvedSource[rightIndex].SendTime {
				// A later provider position wins an otherwise indistinguishable
				// timestamp tie, while physical output remains in provider order.
				return leftIndex > rightIndex
			}
			return resolvedSource[leftIndex].SendTime > resolvedSource[rightIndex].SendTime
		})
		anchors = make([]bool, len(resolvedSource))
		first := filter.StartIndex - 1
		if first > len(candidateIndexes) {
			first = len(candidateIndexes)
		}
		last := len(candidateIndexes)
		if filter.Count > 0 && first+filter.Count < last {
			last = first + filter.Count
		}
		for _, index := range candidateIndexes[first:last] {
			anchors[index] = true
		}
	}
	copy(included, anchors)

	switch filter.Context {
	case chatwork.MessageContextNone:
		// Sender matches are the complete selected result.
	case chatwork.MessageContextReplies:
		for index, message := range resolvedSource {
			if message.Reply == nil || !message.Reply.Resolved {
				continue
			}
			parentIndex, found := messageIndex[message.Reply.Target.Value]
			if !found {
				return nil, nil, fmt.Errorf("resolved Chatwork reply target is absent from its source window")
			}
			if anchors[index] {
				included[parentIndex] = true
			}
			if anchors[parentIndex] {
				included[index] = true
			}
		}
	default:
		return nil, nil, fmt.Errorf("Chatwork message context is invalid")
	}

	selected := make([]chatwork.Message, 0, len(resolvedSource))
	sourceSequences := make([]int, 0, len(resolvedSource))
	anchorSequences := make([]int, 0, len(filter.Senders))
	for index, message := range resolvedSource {
		if !included[index] {
			continue
		}
		selected = append(selected, message)
		sourceSequences = append(sourceSequences, index+1)
		if anchors[index] {
			anchorSequences = append(anchorSequences, index+1)
		}
	}

	// Resolution is relative to the displayed result. Reset the source-window
	// proof before resolving the selected subset so an omitted parent is shown
	// as unresolved instead of retaining a false in-result edge.
	selected = cloneMessages(selected)
	for index := range selected {
		if selected[index].Reply != nil {
			selected[index].Reply.Resolved = false
		}
	}
	selected, err = ResolveMessageRelations(selected)
	if err != nil {
		return nil, nil, err
	}

	filterCopy := filter
	filterCopy.Senders = append([]chatwork.Reference(nil), filter.Senders...)
	selection := &chatwork.MessageSelection{
		Filter:          filterCopy,
		SourceCount:     len(resolvedSource),
		CandidateCount:  len(candidateIndexes),
		SourceSequences: sourceSequences,
		AnchorSequences: anchorSequences,
	}
	if filter.StartIndex > 0 || filter.Count > 0 {
		selection.ItemsPerPage = len(anchorSequences)
	}
	if filter.Count > 0 && filter.StartIndex-1+len(anchorSequences) < len(candidateIndexes) {
		selection.NextStartIndex = filter.StartIndex + len(anchorSequences)
	}
	return selected, selection, nil
}
