package cli

import (
	"fmt"
	"strings"
	"unicode"
)

type successFormat string

const (
	successFormatTSV  successFormat = "tsv"
	successFormatJSON successFormat = "json"
)

func parseSuccessFormat(value string) (successFormat, error) {
	switch successFormat(value) {
	case successFormatTSV:
		return successFormatTSV, nil
	case successFormatJSON:
		return successFormatJSON, nil
	default:
		return successFormatTSV, fmt.Errorf("--format must be tsv or json")
	}
}

func parseFormatOnlyArgs(args []string) (successFormat, error) {
	format := successFormatTSV
	seen := false
	for index := 0; index < len(args); index++ {
		argument := args[index]
		var value string
		switch {
		case argument == "--format":
			if seen {
				return format, fmt.Errorf("--format may be specified only once")
			}
			if index+1 >= len(args) || strings.HasPrefix(args[index+1], "-") {
				return format, fmt.Errorf("--format requires tsv or json")
			}
			index++
			value = args[index]
		case strings.HasPrefix(argument, "--format="):
			if seen {
				return format, fmt.Errorf("--format may be specified only once")
			}
			value = strings.TrimPrefix(argument, "--format=")
		case strings.HasPrefix(argument, "-"):
			return format, fmt.Errorf("フラグ %q は不明です", argument)
		default:
			return format, fmt.Errorf("予期しない引数 %q です", argument)
		}
		parsed, err := parseSuccessFormat(value)
		if err != nil {
			return format, err
		}
		format = parsed
		seen = true
	}
	return format, nil
}

// safeExternalText makes structural runes visible without interpreting the
// remaining text. Backslashes are escaped first so a literal sequence such as
// \n stays distinguishable from a projected newline. Opaque IDs bypass this
// projection and must instead pass their domain validator byte-for-byte.
func safeExternalText(value string) string {
	var output strings.Builder
	for _, r := range value {
		writeExternalRune(&output, r, true)
	}
	return output.String()
}

func escapeTSVCell(value string) string {
	var output strings.Builder
	for _, r := range value {
		writeExternalRune(&output, r, true)
	}
	return output.String()
}

func writeExternalRune(output *strings.Builder, r rune, escapeBackslash bool) {
	if escapeBackslash && r == '\\' {
		output.WriteString("\\\\")
		return
	}
	if r == '\u2028' || r == '\u2029' {
		fmt.Fprintf(output, "\\u%04X", r)
		return
	}
	if unicode.Is(unicode.C, r) {
		switch r {
		case '\t':
			output.WriteString("\\t")
		case '\r':
			output.WriteString("\\r")
		case '\n':
			output.WriteString("\\n")
		default:
			if r <= 0xffff {
				fmt.Fprintf(output, "\\u%04X", r)
			} else {
				fmt.Fprintf(output, "\\U%08X", r)
			}
		}
		return
	}
	output.WriteRune(r)
}
