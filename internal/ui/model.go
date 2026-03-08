package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gfonseca/numer/internal/eval"
)

const (
	resultColWidth = 24
	hPad           = 3 // horizontal padding on each side
	vPad           = 2 // vertical padding (top and bottom)
)

const oddBgColor = lipgloss.Color("235")

const commentColor = lipgloss.Color("135")

var (
	styleResult      = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styleResultOdd   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Background(oddBgColor)
	styleError       = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleErrorOdd    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Background(oddBgColor)
	styleCursor      = lipgloss.NewStyle().Reverse(true)
	styleOddChar     = lipgloss.NewStyle().Background(oddBgColor)
	styleComment     = lipgloss.NewStyle().Foreground(commentColor)
	styleCommentOdd  = lipgloss.NewStyle().Foreground(commentColor).Background(oddBgColor)
	styleStatus      = lipgloss.NewStyle().
				Background(lipgloss.Color("4")).
				Foreground(lipgloss.Color("15")).
				Bold(true)
)

type snapshot struct {
	lines []string
	row   int
	col   int
}

type mode int

const (
	modeNormal mode = iota
	modePrompt      // save-as prompt active in status bar
)

type Model struct {
	lines     []string
	row       int
	col       int
	width     int
	height    int
	scroll    int
	evaluator *eval.Evaluator
	results   []string
	errors    []string
	undoStack []snapshot
	redoStack []snapshot
	filename  string
	dirty     bool
	// prompt state
	inputMode  mode
	promptText string // what the user is typing
	promptMsg  string // success / error feedback
}

func New(filename string) (Model, error) {
	m := Model{
		lines:     []string{""},
		evaluator: eval.New(),
		filename:  filename,
	}
	if filename != "" {
		data, err := os.ReadFile(filename)
		if err != nil && !os.IsNotExist(err) {
			return m, fmt.Errorf("opening %s: %w", filename, err)
		}
		if err == nil {
			lines := strings.Split(string(data), "\n")
			// trim trailing empty line added by most editors
			if len(lines) > 1 && lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}
			m.lines = lines
		}
	}
	m.reeval()
	return m, nil
}

func (m *Model) save() error {
	content := strings.Join(m.lines, "\n") + "\n"
	if err := os.WriteFile(m.filename, []byte(content), 0644); err != nil {
		return err
	}
	m.dirty = false
	m.promptMsg = ""
	return nil
}

func (m *Model) reeval() {
	m.evaluator.Reset()
	m.results = make([]string, len(m.lines))
	m.errors = make([]string, len(m.lines))
	for i, line := range m.lines {
		res, err := m.evaluator.EvalLine(line)
		m.results[i] = res
		m.errors[i] = err
	}
}

const maxUndoStack = 100

func (m *Model) pushUndo() {
	snap := snapshot{
		lines: make([]string, len(m.lines)),
		row:   m.row,
		col:   m.col,
	}
	copy(snap.lines, m.lines)
	if len(m.undoStack) >= maxUndoStack {
		m.undoStack = m.undoStack[1:]
	}
	m.undoStack = append(m.undoStack, snap)
	m.redoStack = nil
	m.dirty = true
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if m.inputMode == modePrompt {
			return m.handlePromptKey(msg)
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Defensive copy so we don't alias the previous model's slice
	cp := make([]string, len(m.lines))
	copy(cp, m.lines)
	m.lines = cp

	line := []rune(m.lines[m.row])

	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyCtrlQ:
		return m, tea.Quit

	case tea.KeyCtrlZ:
		if len(m.undoStack) > 0 {
			redo := snapshot{lines: make([]string, len(m.lines)), row: m.row, col: m.col}
			copy(redo.lines, m.lines)
			m.redoStack = append(m.redoStack, redo)
			snap := m.undoStack[len(m.undoStack)-1]
			m.undoStack = m.undoStack[:len(m.undoStack)-1]
			m.lines = snap.lines
			m.row = snap.row
			m.col = snap.col
			m.adjustScroll()
			m.reeval()
		}

	case tea.KeyCtrlY:
		if len(m.redoStack) > 0 {
			undo := snapshot{lines: make([]string, len(m.lines)), row: m.row, col: m.col}
			copy(undo.lines, m.lines)
			m.undoStack = append(m.undoStack, undo)
			snap := m.redoStack[len(m.redoStack)-1]
			m.redoStack = m.redoStack[:len(m.redoStack)-1]
			m.lines = snap.lines
			m.row = snap.row
			m.col = snap.col
			m.adjustScroll()
			m.reeval()
		}

	case tea.KeyCtrlS:
		if m.filename == "" {
			m.inputMode = modePrompt
			m.promptText = ""
			m.promptMsg = ""
		} else {
			if err := (&m).save(); err != nil {
				m.promptMsg = "error: " + err.Error()
			}
		}

	case tea.KeyUp:
		if m.row > 0 {
			m.row--
			if runes := []rune(m.lines[m.row]); m.col > len(runes) {
				m.col = len(runes)
			}
			m.adjustScroll()
		}

	case tea.KeyDown:
		if m.row < len(m.lines)-1 {
			m.row++
			if runes := []rune(m.lines[m.row]); m.col > len(runes) {
				m.col = len(runes)
			}
			m.adjustScroll()
		}

	case tea.KeyLeft:
		if m.col > 0 {
			m.col--
		} else if m.row > 0 {
			m.row--
			m.col = len([]rune(m.lines[m.row]))
			m.adjustScroll()
		}

	case tea.KeyRight:
		if m.col < len(line) {
			m.col++
		} else if m.row < len(m.lines)-1 {
			m.row++
			m.col = 0
			m.adjustScroll()
		}

	case tea.KeyHome, tea.KeyCtrlA:
		m.col = 0

	case tea.KeyEnd, tea.KeyCtrlE:
		m.col = len(line)

	case tea.KeyPgUp:
		vis := m.visibleLines()
		m.row -= vis
		if m.row < 0 {
			m.row = 0
		}
		if runes := []rune(m.lines[m.row]); m.col > len(runes) {
			m.col = len(runes)
		}
		m.adjustScroll()

	case tea.KeyPgDown:
		vis := m.visibleLines()
		m.row += vis
		if m.row >= len(m.lines) {
			m.row = len(m.lines) - 1
		}
		if runes := []rune(m.lines[m.row]); m.col > len(runes) {
			m.col = len(runes)
		}
		m.adjustScroll()

	case tea.KeyEnter:
		(&m).pushUndo()
		before := string(line[:m.col])
		after := string(line[m.col:])
		m.lines[m.row] = before
		newLines := make([]string, len(m.lines)+1)
		copy(newLines, m.lines[:m.row+1])
		newLines[m.row+1] = after
		copy(newLines[m.row+2:], m.lines[m.row+1:])
		m.lines = newLines
		m.row++
		m.col = 0
		m.adjustScroll()
		m.reeval()

	case tea.KeyBackspace:
		(&m).pushUndo()
		if m.col > 0 {
			newLine := make([]rune, len(line)-1)
			copy(newLine, line[:m.col-1])
			copy(newLine[m.col-1:], line[m.col:])
			m.lines[m.row] = string(newLine)
			m.col--
			m.reeval()
		} else if m.row > 0 {
			prevRunes := []rune(m.lines[m.row-1])
			merged := string(prevRunes) + string(line)
			newLines := make([]string, len(m.lines)-1)
			copy(newLines, m.lines[:m.row])
			copy(newLines[m.row:], m.lines[m.row+1:])
			m.lines = newLines
			m.row--
			m.lines[m.row] = merged
			m.col = len(prevRunes)
			m.adjustScroll()
			m.reeval()
		}

	case tea.KeyDelete:
		(&m).pushUndo()
		if m.col < len(line) {
			newLine := make([]rune, len(line)-1)
			copy(newLine, line[:m.col])
			copy(newLine[m.col:], line[m.col+1:])
			m.lines[m.row] = string(newLine)
			m.reeval()
		} else if m.row < len(m.lines)-1 {
			merged := string(line) + m.lines[m.row+1]
			m.lines[m.row] = merged
			newLines := make([]string, len(m.lines)-1)
			copy(newLines, m.lines[:m.row+1])
			copy(newLines[m.row+1:], m.lines[m.row+2:])
			m.lines = newLines
			m.reeval()
		}

	case tea.KeyCtrlK:
		(&m).pushUndo()
		m.lines[m.row] = string(line[:m.col])
		m.reeval()

	case tea.KeyRunes, tea.KeySpace:
		(&m).pushUndo()
		ins := msg.Runes
		if msg.Type == tea.KeySpace {
			ins = []rune{' '}
		}
		newLine := make([]rune, len(line)+len(ins))
		copy(newLine, line[:m.col])
		copy(newLine[m.col:], ins)
		copy(newLine[m.col+len(ins):], line[m.col:])
		m.lines[m.row] = string(newLine)
		m.col += len(ins)
		m.reeval()
	}

	return m, nil
}

func (m *Model) visibleLines() int {
	n := m.height - 2 - 2*vPad // subtract status bar + top/bottom padding
	if n < 1 {
		return 1
	}
	return n
}

func (m *Model) adjustScroll() {
	vis := m.visibleLines()
	if m.row < m.scroll {
		m.scroll = m.row
	}
	if m.row >= m.scroll+vis {
		m.scroll = m.row - vis + 1
	}
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	vis := m.visibleLines()
	rows := make([]string, 0, m.height)

	emptyRow := strings.Repeat(" ", m.width)
	leftPad := strings.Repeat(" ", hPad)

	for range vPad {
		rows = append(rows, emptyRow)
	}
	for i := m.scroll; i < m.scroll+vis; i++ {
		if i < len(m.lines) {
			rows = append(rows, leftPad+m.renderLine(i))
		} else {
			rows = append(rows, emptyRow)
		}
	}
	for range vPad {
		rows = append(rows, emptyRow)
	}

	rows = append(rows, m.renderStatus())
	return strings.Join(rows, "\n")
}

func (m Model) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Dismiss feedback message on any key
	if m.promptMsg != "" {
		m.promptMsg = ""
		if msg.Type == tea.KeyEscape || msg.Type == tea.KeyCtrlC {
			m.inputMode = modeNormal
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyEscape, tea.KeyCtrlC:
		m.inputMode = modeNormal
		m.promptText = ""

	case tea.KeyEnter:
		path, err := sanitizePath(m.promptText)
		if err != nil {
			m.promptMsg = "error: " + err.Error()
			return m, nil
		}
		m.filename = path
		if err := (&m).save(); err != nil {
			m.filename = ""
			m.promptMsg = "error: " + err.Error()
		} else {
			m.inputMode = modeNormal
			m.promptText = ""
			m.promptMsg = "saved " + filepath.Base(path)
		}

	case tea.KeyBackspace:
		runes := []rune(m.promptText)
		if len(runes) > 0 {
			m.promptText = string(runes[:len(runes)-1])
		}

	case tea.KeyRunes:
		m.promptText += string(msg.Runes)

	case tea.KeySpace:
		m.promptText += " "
	}
	return m, nil
}

// sanitizePath cleans and validates a user-supplied path.
func sanitizePath(raw string) (string, error) {
	p := strings.TrimSpace(raw)
	if p == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	// Expand leading ~
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot resolve home directory")
		}
		p = filepath.Join(home, p[2:])
	}
	p = filepath.Clean(p)
	// Reject suspicious patterns
	if strings.Contains(p, "\x00") {
		return "", fmt.Errorf("invalid path")
	}
	// Ensure parent directory exists
	dir := filepath.Dir(p)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", fmt.Errorf("directory does not exist: %s", dir)
	}
	return p, nil
}

func isComment(line string) bool {
	s := strings.TrimSpace(line)
	return strings.HasPrefix(s, "#") || strings.HasPrefix(s, "//")
}

func (m Model) renderLine(i int) string {
	isOdd := i%2 != 0
	isCmt := isComment(m.lines[i])
	contentW := m.width - 2*hPad
	exprW := contentW - resultColWidth - 1
	if exprW < 5 {
		exprW = contentW // too narrow: skip result column
	}

	runes := []rune(m.lines[i])
	hlStyles := highlightStyles(m.lines[i], isCmt)

	var exprPart string
	if i == m.row {
		exprPart = renderWithCursor(runes, m.col, exprW, isOdd, hlStyles)
	} else {
		exprPart = renderStatic(runes, exprW, isOdd, hlStyles)
	}

	if exprW == contentW {
		return exprPart
	}

	resStyle := styleResult
	errStyle := styleError
	sepStyle := lipgloss.NewStyle()
	if isOdd {
		resStyle = styleResultOdd
		errStyle = styleErrorOdd
		sepStyle = styleOddChar
	}

	var resPart string
	switch {
	case m.errors[i] != "":
		resPart = errStyle.Width(resultColWidth).Align(lipgloss.Right).Render("❌")
	case m.results[i] != "":
		resPart = resStyle.Width(resultColWidth).Align(lipgloss.Right).Render(m.results[i])
	default:
		resPart = sepStyle.Width(resultColWidth).Render("")
	}

	return exprPart + sepStyle.Render(" ") + resPart
}

func renderWithCursor(runes []rune, col, width int, isOdd bool, styles []lipgloss.Style) string {
	viewStart := 0
	if col >= width {
		viewStart = col - width + 1
	}

	var sb strings.Builder
	count := 0
	for i := viewStart; i < len(runes) && count < width; i++ {
		if i == col {
			sb.WriteString(styleCursor.Render(string(runes[i])))
		} else {
			sb.WriteString(applyChar(runes[i], styles[i], isOdd))
		}
		count++
	}
	// Cursor at end of line
	if col >= len(runes) && col-viewStart < width {
		sb.WriteString(styleCursor.Render(" "))
		count++
	}
	for count < width {
		sb.WriteString(applyChar(' ', hlNone, isOdd))
		count++
	}
	return sb.String()
}

func renderStatic(runes []rune, width int, isOdd bool, styles []lipgloss.Style) string {
	var sb strings.Builder
	count := 0
	for i, r := range runes {
		if count >= width {
			break
		}
		sb.WriteString(applyChar(r, styles[i], isOdd))
		count++
	}
	for count < width {
		sb.WriteString(applyChar(' ', hlNone, isOdd))
		count++
	}
	return sb.String()
}

// applyChar renders a single rune with its highlight style and optional odd-line background.
func applyChar(r rune, hl lipgloss.Style, isOdd bool) string {
	if isOdd {
		hl = withBg(hl)
	}
	// Skip render call if no styling at all
	if !isOdd && hl.GetForeground() == (lipgloss.NoColor{}) {
		return string(r)
	}
	return hl.Render(string(r))
}

var stylePromptErr = lipgloss.NewStyle().
	Background(lipgloss.Color("1")).
	Foreground(lipgloss.Color("15")).
	Bold(true)

var stylePromptOk = lipgloss.NewStyle().
	Background(lipgloss.Color("2")).
	Foreground(lipgloss.Color("15")).
	Bold(true)

func (m Model) renderStatus() string {
	if m.inputMode == modePrompt {
		return m.renderPromptBar()
	}

	name := "untitled"
	if m.filename != "" {
		name = filepath.Base(m.filename)
	}
	if m.dirty {
		name += " *"
	}

	var right string
	if m.promptMsg != "" {
		right = fmt.Sprintf(" %s  ^Q quit ", m.promptMsg)
	} else if m.filename != "" {
		right = fmt.Sprintf(" Ln %d  Col %d  ^S save  ^Q quit ", m.row+1, m.col+1)
	} else {
		right = fmt.Sprintf(" Ln %d  Col %d  ^S save  ^Q quit ", m.row+1, m.col+1)
	}

	left := " " + name + " "
	padLen := m.width - len(left) - len(right)
	if padLen < 0 {
		padLen = 0
	}
	return styleStatus.Width(m.width).Render(left + strings.Repeat(" ", padLen) + right)
}

func (m Model) renderPromptBar() string {
	prefix := " Save as: "
	cursor := styleCursor.Render(" ")
	hint := "  Esc cancel "

	input := m.promptText
	// Reserve space: prefix + input + cursor + hint
	maxInput := m.width - len(prefix) - len(hint) - 1
	if maxInput < 1 {
		maxInput = 1
	}
	// Scroll input view to keep end visible
	runes := []rune(input)
	if len(runes) > maxInput {
		runes = runes[len(runes)-maxInput:]
	}

	bar := prefix + string(runes) + cursor + strings.Repeat(" ", max(0, maxInput-len(runes))) + hint
	if m.promptMsg != "" {
		if strings.HasPrefix(m.promptMsg, "error") {
			return stylePromptErr.Width(m.width).Render(" " + m.promptMsg + " ")
		}
		return stylePromptOk.Width(m.width).Render(" " + m.promptMsg + " ")
	}
	return styleStatus.Width(m.width).Render(bar)
}
