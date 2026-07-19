package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/tasuku43/cwk/internal/app/configcmd"
	"github.com/tasuku43/cwk/internal/app/execution"
	"github.com/tasuku43/cwk/internal/domain/commandselection"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

const maxConfigSelectionLineBytes = 4096

type commandSelectionState struct {
	configured bool
	source     string
	enabled    map[string]bool
	stale      []string
	viewErr    error
}

func runConfigShow(ctx context.Context, c *CLI, command CommandSpec, _ operation.Intent, args []string) int {
	if len(args) != 0 {
		return c.failUsage(ctx, "invalid_arguments", "usage: "+command.Usage(), "help config show", "Run the read-only inspection without arguments.")
	}
	state, err := c.loadCommandSelectionState(ctx)
	if err != nil {
		return c.fail(ctx, err)
	}
	if state.viewErr != nil {
		return c.fail(ctx, fault.Wrap(
			fault.KindInvalidInput,
			"command_selection_invalid",
			"The saved command selection is invalid.",
			false,
			state.viewErr,
		))
	}
	return c.emit(ctx, c.renderCommandSelection(state))
}

func runConfigEdit(ctx context.Context, c *CLI, command CommandSpec, base operation.Intent, args []string) int {
	if len(args) != 0 {
		return c.failUsage(ctx, "invalid_arguments", "usage: "+command.Usage(), "help config edit", "Run the selector without command arguments.")
	}

	state, repairWarning, err := c.loadCommandSelectionForEdit(ctx)
	if err != nil {
		return c.fail(ctx, err)
	}
	choices := c.baseCatalog.ConfigurableCommands()
	if len(c.baseCatalog.commands) == 0 {
		choices = c.catalog.ConfigurableCommands()
	}
	selected := cloneSelectionMap(state.enabled)
	original := cloneSelectionMap(state.enabled)

	var initial strings.Builder
	initial.WriteString("Command selection (attention only; not an authorization or security boundary)\n")
	if repairWarning {
		initial.WriteString("warning: saved selection is invalid; nothing changes until save\n")
	}
	if state.viewErr != nil {
		fmt.Fprintf(&initial, "warning: current selection is incomplete: %s\n", escapeTSVCell(state.viewErr.Error()))
	}
	initial.WriteString("Always enabled:\n")
	baseCatalog := c.baseCatalog
	if len(baseCatalog.commands) == 0 {
		baseCatalog = c.catalog
	}
	for _, spec := range baseCatalog.AlwaysCommands() {
		fmt.Fprintf(&initial, "  %s\n", spec.Path)
	}
	initial.WriteString("Selectable commands:\n")
	for index, spec := range choices {
		mark := " "
		if selected[spec.Path] {
			mark = "x"
		}
		fmt.Fprintf(&initial, "  %d [%s] %s - %s\n", index+1, mark, spec.Path, spec.Summary)
	}
	initial.WriteString("Enter numbers to toggle, or one of: all, none, save, cancel\nconfig> ")
	if err := c.writeConfigTranscript(ctx, []byte(initial.String())); err != nil {
		return c.fail(ctx, err)
	}

	lines := scanConfigLines(ctx, c.In)
	for {
		var result configLineResult
		select {
		case <-ctx.Done():
			return c.fail(ctx, ctx.Err())
		case received, ok := <-lines:
			if !ok {
				return c.fail(ctx, fault.New(fault.KindCanceled, "configuration_canceled", "Command selection editing was canceled.", false))
			}
			result = received
		}

		if result.err != nil {
			if errors.Is(result.err, io.EOF) {
				return c.fail(ctx, fault.New(fault.KindCanceled, "configuration_canceled", "Command selection editing was canceled.", false))
			}
			return c.fail(ctx, fault.Wrap(
				fault.KindInternal,
				"configuration_input_failed",
				"Command selection input could not be read.",
				false,
				result.err,
			))
		}

		action, indexes, parseErr := parseConfigSelectionLine(result.line, len(choices))
		if parseErr != nil {
			if err := c.writeConfigTranscript(ctx, []byte("invalid selection; use displayed numbers, all, none, save, or cancel\nconfig> ")); err != nil {
				return c.fail(ctx, err)
			}
			continue
		}

		switch action {
		case configSelectionCancel:
			return c.fail(ctx, fault.New(fault.KindCanceled, "configuration_canceled", "Command selection editing was canceled.", false))
		case configSelectionAll:
			for _, choice := range choices {
				selected[choice.Path] = true
			}
		case configSelectionNone:
			for _, choice := range choices {
				selected[choice.Path] = false
			}
		case configSelectionToggle:
			for _, index := range indexes {
				path := choices[index-1].Path
				selected[path] = !selected[path]
			}
		case configSelectionSave:
			enabled := selectedPathsInCatalogOrder(choices, selected)
			if _, _, err := baseCatalog.ActiveView(enabled); err != nil {
				message := fmt.Sprintf("cannot save: %s\nconfig> ", escapeTSVCell(err.Error()))
				if writeErr := c.writeConfigTranscript(ctx, []byte(message)); writeErr != nil {
					return c.fail(ctx, writeErr)
				}
				continue
			}
			profile, err := commandselection.New(enabled)
			if err != nil {
				return c.fail(ctx, fault.Wrap(fault.KindInvalidInput, "command_selection_invalid", "The selected command paths are invalid.", false, err))
			}
			request, err := buildConfigExecutionRequest(command, base)
			if err != nil {
				return c.fail(ctx, fault.Wrap(fault.KindContract, "invalid_mutation_contract", "The command-selection mutation contract is invalid.", false, err))
			}
			err = execution.New(configcmd.ExplicitSavePolicy{Confirmed: true}).Invoke(
				ctx,
				request,
				func(actionContext context.Context, _ operation.Intent) error {
					if c == nil || c.commandSelection == nil {
						return fault.New(fault.KindUnavailable, "command_selection_unavailable", "Command selection configuration is unavailable.", true)
					}
					return c.commandSelection.Save(actionContext, profile)
				},
			)
			if err != nil {
				return c.fail(ctx, err)
			}
			output := fmt.Sprintf(
				"config saved enabled=%d disabled=%d changed=%d stale-removed=%d\n",
				len(enabled), len(choices)-len(enabled), selectionChangeCount(choices, original, selected), len(state.stale),
			)
			return c.emit(ctx, []byte(output))
		}

		enabledCount := len(selectedPathsInCatalogOrder(choices, selected))
		status := fmt.Sprintf("selection enabled=%d disabled=%d\nconfig> ", enabledCount, len(choices)-enabledCount)
		if err := c.writeConfigTranscript(ctx, []byte(status)); err != nil {
			return c.fail(ctx, err)
		}
	}
}

func buildConfigExecutionRequest(command CommandSpec, base operation.Intent) (execution.Request, error) {
	if command.Agent.FixedTarget == nil || command.Agent.Mutation == nil {
		return execution.Request{}, fmt.Errorf("fixed command-selection target is missing")
	}
	intent := base
	intent.Target = operation.TargetRef{
		Kind: command.Agent.FixedTarget.Kind,
		ID:   command.Agent.FixedTarget.StableID,
	}
	intent.Impact = command.Agent.Mutation.Impact
	request := execution.Request{
		Intent:          intent,
		ExpectedCommand: command.Path,
		ExpectedEffect:  command.Effect,
		ExpectedTarget:  intent.Target,
		ExpectedImpact:  intent.Impact,
	}
	if intent != configcmd.EditIntent() {
		return execution.Request{}, fmt.Errorf("catalog and application command-selection targets differ")
	}
	return request, nil
}

func (c *CLI) loadCommandSelectionState(ctx context.Context) (commandSelectionState, error) {
	base := c.baseCatalog
	if len(base.commands) == 0 {
		base = c.catalog
	}
	if c == nil || c.commandSelection == nil {
		return commandSelectionState{}, fault.New(fault.KindUnavailable, "command_selection_unavailable", "Command selection configuration is unavailable.", true)
	}
	profile, configured, err := c.commandSelection.Load(ctx)
	if err != nil {
		return commandSelectionState{}, err
	}
	return deriveCommandSelectionState(base, profile, configured), nil
}

func (c *CLI) loadCommandSelectionForEdit(ctx context.Context) (commandSelectionState, bool, error) {
	state, err := c.loadCommandSelectionState(ctx)
	if err == nil {
		return state, false, nil
	}
	public, ok := fault.PublicCopy(err)
	if !ok || public.Code != "command_selection_invalid" {
		return commandSelectionState{}, false, err
	}
	base := c.baseCatalog
	if len(base.commands) == 0 {
		base = c.catalog
	}
	return defaultCommandSelectionState(base, "repair"), true, nil
}

func deriveCommandSelectionState(base Catalog, profile commandselection.Profile, configured bool) commandSelectionState {
	if !configured {
		return defaultCommandSelectionState(base, "default")
	}
	state := commandSelectionState{
		configured: true,
		source:     "saved",
		enabled:    make(map[string]bool),
	}
	for _, path := range profile.EnabledCommands() {
		command, known := base.Lookup(path)
		switch {
		case !known:
			state.stale = append(state.stale, path)
		case command.Configurable:
			state.enabled[path] = true
		}
	}
	_, _, state.viewErr = base.ActiveView(profile.EnabledCommands())
	return state
}

func defaultCommandSelectionState(base Catalog, source string) commandSelectionState {
	state := commandSelectionState{source: source, enabled: make(map[string]bool)}
	paths := make([]string, 0)
	for _, command := range base.ConfigurableCommands() {
		state.enabled[command.Path] = true
		paths = append(paths, command.Path)
	}
	_, _, state.viewErr = base.ActiveView(paths)
	return state
}

func (c *CLI) renderCommandSelection(state commandSelectionState) []byte {
	base := c.baseCatalog
	if len(base.commands) == 0 {
		base = c.catalog
	}
	var output strings.Builder
	fmt.Fprintf(&output, "config purpose=attention-only security-boundary=false source=%s\n", state.source)
	writeCommandSection(&output, "always-on", base.AlwaysCommands(), func(CommandSpec) bool { return true })
	writeCommandSection(&output, "enabled", base.ConfigurableCommands(), func(command CommandSpec) bool { return state.enabled[command.Path] })
	writeCommandSection(&output, "disabled", base.ConfigurableCommands(), func(command CommandSpec) bool { return !state.enabled[command.Path] })
	fmt.Fprintf(&output, "stale count=%d\n", len(state.stale))
	for _, path := range state.stale {
		fmt.Fprintf(&output, "  %s\n", path)
	}
	return []byte(output.String())
}

func writeCommandSection(output *strings.Builder, label string, commands []CommandSpec, include func(CommandSpec) bool) {
	count := 0
	for _, command := range commands {
		if include(command) {
			count++
		}
	}
	fmt.Fprintf(output, "%s count=%d\n", label, count)
	for _, command := range commands {
		if include(command) {
			fmt.Fprintf(output, "  %s\n", command.Path)
		}
	}
}

func (c *CLI) writeConfigTranscript(ctx context.Context, contents []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := writeOnce(c.Out, contents); err != nil {
		return fault.Wrap(fault.KindInternal, "output_write_failed", "The complete command-selection output could not be written.", true, err)
	}
	return nil
}

type configLineResult struct {
	line string
	err  error
}

func scanConfigLines(ctx context.Context, reader io.Reader) <-chan configLineResult {
	results := make(chan configLineResult)
	go func() {
		defer close(results)
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 1024), maxConfigSelectionLineBytes)
		for scanner.Scan() {
			result := configLineResult{line: scanner.Text()}
			select {
			case results <- result:
			case <-ctx.Done():
				return
			}
		}
		err := scanner.Err()
		if err == nil {
			err = io.EOF
		}
		select {
		case results <- configLineResult{err: err}:
		case <-ctx.Done():
		}
	}()
	return results
}

type configSelectionAction uint8

const (
	configSelectionToggle configSelectionAction = iota
	configSelectionAll
	configSelectionNone
	configSelectionSave
	configSelectionCancel
)

func parseConfigSelectionLine(line string, choiceCount int) (configSelectionAction, []int, error) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, ",") || strings.HasSuffix(trimmed, ",") || strings.Contains(trimmed, ",,") {
		return configSelectionToggle, nil, fmt.Errorf("selection separators are invalid")
	}
	words := strings.Fields(strings.ReplaceAll(trimmed, ",", " "))
	if len(words) == 0 {
		return configSelectionToggle, nil, fmt.Errorf("selection is empty")
	}
	if len(words) == 1 {
		switch words[0] {
		case "all":
			return configSelectionAll, nil, nil
		case "none":
			return configSelectionNone, nil, nil
		case "save":
			return configSelectionSave, nil, nil
		case "cancel":
			return configSelectionCancel, nil, nil
		}
	}
	indexes := make([]int, 0, len(words))
	seen := make(map[int]struct{}, len(words))
	for _, word := range words {
		index, err := strconv.Atoi(word)
		if err != nil || index < 1 || index > choiceCount {
			return configSelectionToggle, nil, fmt.Errorf("selection index is invalid")
		}
		if _, duplicate := seen[index]; duplicate {
			return configSelectionToggle, nil, fmt.Errorf("selection index is duplicated")
		}
		seen[index] = struct{}{}
		indexes = append(indexes, index)
	}
	return configSelectionToggle, indexes, nil
}

func cloneSelectionMap(values map[string]bool) map[string]bool {
	cloned := make(map[string]bool, len(values))
	for path, enabled := range values {
		cloned[path] = enabled
	}
	return cloned
}

func selectedPathsInCatalogOrder(choices []CommandSpec, selected map[string]bool) []string {
	paths := make([]string, 0, len(choices))
	for _, choice := range choices {
		if selected[choice.Path] {
			paths = append(paths, choice.Path)
		}
	}
	return paths
}

func selectionChangeCount(choices []CommandSpec, before, after map[string]bool) int {
	count := 0
	for _, choice := range choices {
		if before[choice.Path] != after[choice.Path] {
			count++
		}
	}
	return count
}
