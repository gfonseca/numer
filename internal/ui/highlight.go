package ui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

var (
	hlNum   = lipgloss.NewStyle().Foreground(lipgloss.Color("221")) // gold
	hlOp    = lipgloss.NewStyle().Foreground(lipgloss.Color("203")) // coral
	hlFn    = lipgloss.NewStyle().Foreground(lipgloss.Color("114")) // soft green
	hlConst = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // amber
	hlVar   = lipgloss.NewStyle().Foreground(lipgloss.Color("183")) // soft lavender
	hlParen = lipgloss.NewStyle().Foreground(lipgloss.Color("215")) // orange
	hlNone  = lipgloss.NewStyle()
)

var knownFunctions = map[string]bool{
	"sqrt": true, "abs": true, "floor": true, "ceil": true, "round": true,
	"sin": true, "cos": true, "tan": true, "asin": true, "acos": true,
	"atan": true, "atan2": true, "log": true, "ln": true, "log2": true,
	"log10": true, "exp": true, "pow": true, "max": true, "min": true, "sum": true,
}

var knownConstants = map[string]bool{
	"pi": true, "e": true, "tau": true, "phi": true, "last": true,
}

// highlightStyles returns one lipgloss.Style per rune in line.
// For comment lines, the entire slice is filled with the comment style.
func highlightStyles(line string, isCmt bool) []lipgloss.Style {
	runes := []rune(line)
	styles := make([]lipgloss.Style, len(runes))

	if isCmt {
		for i := range styles {
			styles[i] = styleComment
		}
		return styles
	}

	for i := range styles {
		styles[i] = hlNone
	}

	i := 0
	for i < len(runes) {
		ch := runes[i]

		switch {
		case unicode.IsDigit(ch) || (ch == '.' && i+1 < len(runes) && unicode.IsDigit(runes[i+1])):
			start := i
			for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.') {
				i++
			}
			// scientific notation
			if i < len(runes) && (runes[i] == 'e' || runes[i] == 'E') {
				i++
				if i < len(runes) && (runes[i] == '+' || runes[i] == '-') {
					i++
				}
				for i < len(runes) && unicode.IsDigit(runes[i]) {
					i++
				}
			}
			for j := start; j < i; j++ {
				styles[j] = hlNum
			}

		case unicode.IsLetter(ch) || ch == '_':
			start := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
				i++
			}
			name := strings.ToLower(string(runes[start:i]))
			// peek past whitespace for '(' to detect function call
			j := i
			for j < len(runes) && unicode.IsSpace(runes[j]) {
				j++
			}
			var s lipgloss.Style
			switch {
			case j < len(runes) && runes[j] == '(':
				s = hlFn
			case knownConstants[name]:
				s = hlConst
			default:
				s = hlVar
			}
			for k := start; k < i; k++ {
				styles[k] = s
			}

		case ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '%' || ch == '^' || ch == '=':
			styles[i] = hlOp
			i++

		case ch == '(' || ch == ')' || ch == ',':
			styles[i] = hlParen
			i++

		default:
			i++
		}
	}

	return styles
}

// withBg returns the style with the odd-line background added.
func withBg(s lipgloss.Style) lipgloss.Style {
	return s.Background(oddBgColor)
}
