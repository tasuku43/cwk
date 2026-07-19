package cli

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tasuku43/cwk/internal/domain/operation"
)

// configTUIKey is one presentation-level selector action. Terminal-specific
// raw-mode setup and reads stay outside this pure model.
type configTUIKey uint8

const (
	configTUIKeyIgnored configTUIKey = iota
	configTUIKeyUp
	configTUIKeyDown
	configTUIKeyToggle
	configTUIKeySave
	configTUIKeyCancel
	configTUIKeyInterrupt
)

// configTUIDecision tells the I/O owner whether the selector remains active or
// whether it should cross the save boundary or leave without saving.
type configTUIDecision uint8

const (
	configTUIContinue configTUIDecision = iota
	configTUISave
	configTUICancel
	configTUIInterrupted
)

type configTUIItem struct {
	Path    string
	Summary string
	Effect  operation.Effect
	Enabled bool
}

// configTUIModel owns cursor, draft-selection state, and the one bit proving
// that the current identity was present in the last complete frame. Items are
// copied in the curated catalog order supplied by Catalog.ConfigurableCommands;
// the catalog remains the single source for deterministic screen and save order.
type configTUIModel struct {
	items                     []configTUIItem
	alwaysPaths               []string
	cursor                    int
	notice                    string
	selectionVisibleLastFrame bool
}

func newConfigTUIModel(items []configTUIItem, alwaysPaths []string) configTUIModel {
	cloned := append([]configTUIItem(nil), items...)
	return configTUIModel{items: cloned, alwaysPaths: append([]string(nil), alwaysPaths...)}
}

func (m *configTUIModel) apply(key configTUIKey) configTUIDecision {
	if m == nil {
		return configTUIContinue
	}
	switch key {
	case configTUIKeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case configTUIKeyDown:
		if m.cursor+1 < len(m.items) {
			m.cursor++
		}
	case configTUIKeyToggle:
		if len(m.items) != 0 {
			m.items[m.cursor].Enabled = !m.items[m.cursor].Enabled
		}
	case configTUIKeySave:
		return configTUISave
	case configTUIKeyCancel:
		return configTUICancel
	case configTUIKeyInterrupt:
		return configTUIInterrupted
	}
	return configTUIContinue
}

func (m configTUIModel) enabledPaths() []string {
	paths := make([]string, 0, len(m.items))
	for _, item := range m.items {
		if item.Enabled {
			paths = append(paths, item.Path)
		}
	}
	return paths
}

type configTUIEscapeState uint8

const (
	configTUIEscapeNone configTUIEscapeState = iota
	configTUIEscapeStarted
	configTUIEscapeCSI
	configTUIEscapeSS3
)

// configTUIKeyParser incrementally decodes the small key vocabulary used by
// the selector. A caller that receives no continuation byte for a lone Escape
// invokes flushEscape after its own bounded timeout.
type configTUIKeyParser struct {
	escapeState configTUIEscapeState
	swallowLF   bool
}

func (p *configTUIKeyParser) feed(chunk []byte) []configTUIKey {
	if p == nil {
		return nil
	}
	keys := make([]configTUIKey, 0, len(chunk))
	for _, value := range chunk {
		p.feedByte(value, &keys)
	}
	return keys
}

func (p *configTUIKeyParser) feedByte(value byte, keys *[]configTUIKey) {
	for {
		switch p.escapeState {
		case configTUIEscapeStarted:
			switch value {
			case '[':
				p.escapeState = configTUIEscapeCSI
				return
			case 'O':
				p.escapeState = configTUIEscapeSS3
				return
			default:
				p.escapeState = configTUIEscapeNone
				appendConfigTUIKey(keys, configTUIKeyCancel)
				// The byte after a non-sequence Escape is still a key of its
				// own. Reprocess it in the ground state.
				continue
			}
		case configTUIEscapeCSI, configTUIEscapeSS3:
			if value == 0x1b {
				p.escapeState = configTUIEscapeStarted
				return
			}
			// CSI and SS3 final bytes occupy 0x40..0x7e. Intermediate
			// parameter bytes are retained across arbitrarily fragmented
			// reads until a final byte arrives.
			if value < 0x40 || value > 0x7e {
				return
			}
			p.escapeState = configTUIEscapeNone
			switch value {
			case 'A':
				appendConfigTUIKey(keys, configTUIKeyUp)
			case 'B':
				appendConfigTUIKey(keys, configTUIKeyDown)
			}
			return
		}

		if p.swallowLF {
			p.swallowLF = false
			if value == '\n' {
				return
			}
		}
		switch value {
		case 0x1b:
			p.escapeState = configTUIEscapeStarted
		case 0x03:
			appendConfigTUIKey(keys, configTUIKeyInterrupt)
		case 'q':
			appendConfigTUIKey(keys, configTUIKeyCancel)
		case ' ':
			appendConfigTUIKey(keys, configTUIKeyToggle)
		case '\r':
			p.swallowLF = true
			appendConfigTUIKey(keys, configTUIKeySave)
		case '\n':
			appendConfigTUIKey(keys, configTUIKeySave)
		}
		return
	}
}

// flushEscape resolves a lone or incomplete escape sequence as cancel. It is
// deliberately time-free so the terminal owner controls the timeout policy.
func (p *configTUIKeyParser) flushEscape() []configTUIKey {
	if p == nil || p.escapeState == configTUIEscapeNone {
		return nil
	}
	p.escapeState = configTUIEscapeNone
	return []configTUIKey{configTUIKeyCancel}
}

func appendConfigTUIKey(keys *[]configTUIKey, key configTUIKey) {
	*keys = append(*keys, key)
}

const (
	configTUITitle        = "Command selection"
	configTUIFooter       = "Up/Down move  Space toggle  Enter save  q quit"
	configTUIResizeNotice = "Resize terminal to review the exact command"
	configTUIQuitFooter   = "q quit"

	configTUIColorCyan    = "\x1b[36m"
	configTUIColorYellow  = "\x1b[33m"
	configTUIColorMagenta = "\x1b[35m"
	configTUIColorReset   = "\x1b[0m"
)

// renderConfigTUIScreen returns one complete, repaintable frame. Its only ANSI
// spans are the fixed effect-badge colors authored below; the terminal owner
// decides how to clear or replace a previous frame. Every physical line is
// truncated to the supplied display width and the complete frame never
// exceeds height lines.
func renderConfigTUIScreen(model configTUIModel, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if !configTUISelectionVisible(model, width, height) {
		return renderConfigTUIResizeScreen(width, height)
	}
	lines := make([]string, 0, height)
	lines = append(lines, truncateConfigTUILine(configTUITitle, width))
	lines = append(lines, truncateConfigTUILine(renderConfigTUIAlways(model.alwaysPaths), width))

	noticeLines, _ := configTUINoticeLines(model, width, height)
	lines = append(lines, noticeLines...)
	visibleItems := configTUIVisibleItemCount(model, width, height)
	start := configTUIViewportStart(len(model.items), model.cursor, visibleItems)
	for index := start; index < start+visibleItems; index++ {
		lines = append(lines, renderConfigTUIItem(model.items[index], index == model.cursor, width))
	}
	lines = append(lines, truncateConfigTUILine(configTUIFooter, width))
	return strings.Join(lines, "\n") + "\n"
}

func renderConfigTUIResizeScreen(width, height int) string {
	lines := make([]string, 0, min(height, 3))
	if height == 1 {
		lines = append(lines, truncateConfigTUILine(configTUIResizeNotice, width))
		return strings.Join(lines, "\n") + "\n"
	}
	lines = append(lines, truncateConfigTUILine(configTUITitle, width))
	lines = append(lines, truncateConfigTUILine(configTUIResizeNotice, width))
	if height >= 3 {
		lines = append(lines, truncateConfigTUILine(configTUIQuitFooter, width))
	}
	return strings.Join(lines, "\n") + "\n"
}

func configTUISelectionVisible(model configTUIModel, width, height int) bool {
	if width <= 0 || height <= 0 || len(model.items) == 0 || model.cursor < 0 || model.cursor >= len(model.items) {
		return false
	}
	if configTUIVisibleItemCount(model, width, height) == 0 {
		return false
	}
	prefix, badge, _ := configTUIItemPrefix(model.items[model.cursor], true)
	identity := prefix + badge + " " + safeExternalText(model.items[model.cursor].Path)
	return configTUIDisplayWidth(identity) <= width
}

func configTUIVisibleItemCount(model configTUIModel, width, height int) int {
	noticeLines, noticeFits := configTUINoticeLines(model, width, height)
	if !noticeFits {
		return 0
	}
	visible := height - 3 - len(noticeLines)
	if visible < 0 {
		return 0
	}
	if visible > len(model.items) {
		return len(model.items)
	}
	return visible
}

func configTUINoticeLines(model configTUIModel, width, height int) ([]string, bool) {
	if model.notice == "" {
		return nil, true
	}
	maximum := height - 4 // Reserve title, always-on facts, one item, and footer.
	return wrapConfigTUILines("Notice: "+safeExternalText(model.notice), width, maximum)
}

func renderConfigTUIAlways(paths []string) string {
	if len(paths) == 0 {
		return "Always on: (none)"
	}
	escaped := make([]string, len(paths))
	for index, path := range paths {
		escaped[index] = safeExternalText(path)
	}
	return "Always on: " + strings.Join(escaped, ", ")
}

// wrapConfigTUILines returns only a complete, structure-safe wrapping. A
// partial diagnostic is not actionable, so callers fall back to resize
// guidance rather than adding an ellipsis or dropping whitespace.
func wrapConfigTUILines(value string, width, maximum int) ([]string, bool) {
	if width <= 0 || maximum <= 0 || value == "" {
		return nil, value == ""
	}
	lines := make([]string, 0, maximum)
	remaining := value
	for len(remaining) != 0 && len(lines) < maximum {
		cut := configTUIByteIndexAtWidth(remaining, width)
		if cut >= len(remaining) {
			lines = append(lines, remaining)
			return lines, true
		}
		if cut == 0 {
			return nil, false
		}
		lines = append(lines, remaining[:cut])
		remaining = remaining[cut:]
	}
	return nil, false
}

func configTUIByteIndexAtWidth(value string, width int) int {
	used := 0
	for index, r := range value {
		runeWidth := configTUIRuneWidth(r)
		if used+runeWidth > width {
			return index
		}
		used += runeWidth
	}
	return len(value)
}

func configTUIViewportStart(itemCount, cursor, visible int) int {
	if itemCount <= 0 || visible <= 0 {
		return 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= itemCount {
		cursor = itemCount - 1
	}
	start := cursor - visible/2
	if start < 0 {
		start = 0
	}
	if maximum := itemCount - visible; start > maximum {
		start = maximum
	}
	return start
}

func renderConfigTUIItem(item configTUIItem, current bool, width int) string {
	prefix, badge, color := configTUIItemPrefix(item, current)
	plainPrefix := prefix + badge + " "
	line := plainPrefix + safeExternalText(item.Path)
	if item.Summary != "" {
		line += " - " + safeExternalText(item.Summary)
	}
	// Truncate the terminal-safe plain structure before adding color. The only
	// ANSI bytes in the result are therefore CLI-authored complete sequences,
	// and they never consume display width or become attacker-controlled.
	line = truncateConfigTUILine(line, width)
	if color == "" || !strings.HasPrefix(line, plainPrefix) {
		return line
	}
	return prefix + color + badge + configTUIColorReset + line[len(prefix)+len(badge):]
}

func configTUIItemPrefix(item configTUIItem, current bool) (prefix, badge, color string) {
	prefix = "  [ ] "
	if current {
		prefix = "> [ ] "
	}
	if item.Enabled {
		prefix = strings.Replace(prefix, "[ ]", "[x]", 1)
	}
	badge, color = configTUIEffectBadge(item.Effect)
	return prefix, badge, color
}

func configTUIEffectBadge(effect operation.Effect) (badge, color string) {
	switch effect {
	case operation.EffectRead:
		return "[read]", configTUIColorCyan
	case operation.EffectCreate:
		return "[create]", configTUIColorYellow
	case operation.EffectWrite:
		return "[write]", configTUIColorMagenta
	default:
		return "[unknown]", ""
	}
}

func truncateConfigTUILine(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if configTUIDisplayWidth(value) <= width {
		return value
	}
	const ellipsis = '…'
	limit := width - configTUIRuneWidth(ellipsis)
	if limit < 0 {
		return ""
	}
	var output strings.Builder
	used := 0
	for _, r := range value {
		runeWidth := configTUIRuneWidth(r)
		if used+runeWidth > limit {
			break
		}
		output.WriteRune(r)
		used += runeWidth
	}
	output.WriteRune(ellipsis)
	return output.String()
}

func configTUIDisplayWidth(value string) int {
	width := 0
	for index := 0; index < len(value); {
		if value[index] == 0x1b && index+1 < len(value) && value[index+1] == '[' {
			index += 2
			for index < len(value) {
				final := value[index]
				index++
				if final >= 0x40 && final <= 0x7e {
					break
				}
			}
			continue
		}
		r, size := utf8.DecodeRuneInString(value[index:])
		width += configTUIRuneWidth(r)
		index += size
	}
	return width
}

// configTUIRuneWidth covers the common wide ranges relevant to command names
// and summaries without making terminal presentation depend on a Unicode
// width package. Structural/control runes have already been escaped.
func configTUIRuneWidth(r rune) int {
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
		return 0
	}
	if r >= 0x1100 && (r <= 0x115f || r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f300 && r <= 0x1faff) ||
		(r >= 0x20000 && r <= 0x3fffd)) {
		return 2
	}
	return 1
}
