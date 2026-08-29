package editor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"tide/internal/runner"
)

// Mode represents the current state of the editor pane.
type Mode int

const (
	ModeView Mode = iota
	ModeEdit
)

// Model represents the editor / Chroma viewer component.
type Model struct {
	Buffer      Buffer
	Mode        Mode
	Textarea    textarea.Model
	ScrollLine  int
	ScrollCol   int
	WordWrap    bool
	Width       int
	Height      int
	Focused     bool
	Theme       string
	Diagnostics map[int][]runner.Diagnostic
}

// New creates a new editor model.
func New(theme string, width, height int) Model {
	ta := textarea.New()
	ta.ShowLineNumbers = true
	ta.Prompt = ""
	ta.CharLimit = 0 // unlimited

	// Custom styling for textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("#282A36"))
	ta.FocusedStyle.LineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	ta.BlurredStyle.LineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color("#44475A"))

	m := Model{
		Buffer:      NewBuffer(),
		Mode:        ModeView,
		Textarea:    ta,
		ScrollLine:  0,
		ScrollCol:   0,
		WordWrap:    false,
		Width:       width,
		Height:      height,
		Theme:       theme,
		Diagnostics: make(map[int][]runner.Diagnostic),
	}
	m.updateSizes()
	return m
}

// OpenFile loads a file into the editor with sanity checks.
func (m *Model) OpenFile(filePath string) error {
	buf, err := LoadFile(filePath)
	m.Buffer = buf
	m.ScrollLine = 0
	m.ScrollCol = 0
	if err != nil {
		m.Mode = ModeView
		m.Textarea.SetValue("")
		return err
	}

	m.Textarea.SetValue(buf.CurrentText)
	m.Textarea.CursorStart()
	return nil
}

// Reload re-reads the active file from disk if unchanged in memory.
// Returns true if content was modified on disk and reloaded.
func (m *Model) Reload() bool {
	changed, _ := m.Buffer.Reload()
	if changed && m.Buffer.IsLoaded {
		if m.Mode == ModeEdit {
			m.Textarea.SetValue(m.Buffer.CurrentText)
		}
	}
	return changed
}

// ToggleMode switches between View and Edit modes (only if a valid text file is loaded).
func (m *Model) ToggleMode() {
	if !m.Buffer.IsLoaded {
		return
	}

	if m.Mode == ModeView {
		m.Mode = ModeEdit
		m.Textarea.SetValue(m.Buffer.CurrentText)
		m.Textarea.Focus()
	} else {
		m.Mode = ModeView
		m.Buffer.SetText(m.Textarea.Value())
		m.Textarea.Blur()
	}
}

// ToggleWordWrap toggles word wrapping on and off in View mode.
func (m *Model) ToggleWordWrap() {
	m.WordWrap = !m.WordWrap
	if m.WordWrap {
		m.ScrollCol = 0
	}
}

// SetDiagnostics updates compiler diagnostics for error highlights.
func (m *Model) SetDiagnostics(allDiags []runner.Diagnostic) {
	if m.Buffer.FilePath == "" {
		m.Diagnostics = make(map[int][]runner.Diagnostic)
		return
	}
	m.Diagnostics = runner.DiagnosticsForFile(allDiags, m.Buffer.FilePath)
}

// SaveFile saves current buffer to disk.
func (m *Model) SaveFile() error {
	if !m.Buffer.IsLoaded {
		return nil
	}
	if m.Mode == ModeEdit {
		m.Buffer.SetText(m.Textarea.Value())
	}
	return m.Buffer.Save()
}

// SetSize updates editor dimensions.
func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	m.updateSizes()
}

func (m *Model) updateSizes() {
	contentHeight := max(1, m.Height-1) // 1 row for mode status bar
	contentWidth := max(10, m.Width)
	m.Textarea.SetWidth(contentWidth)
	m.Textarea.SetHeight(contentHeight)
}

// Update handles tea.Msg for editor.
func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.Mode == ModeEdit {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.Type == tea.KeyTab || keyMsg.String() == "tab" {
				m.Textarea.InsertString("\t")
				m.Buffer.IsModified = (m.Textarea.Value() != m.Buffer.InitialText)
				return *m, nil
			}
		}
		var cmd tea.Cmd
		m.Textarea, cmd = m.Textarea.Update(msg)
		// Check modification
		m.Buffer.IsModified = (m.Textarea.Value() != m.Buffer.InitialText)
		return *m, cmd
	}

	// In View Mode, handle scrolling and word-wrap toggle
	if keyMsg, ok := msg.(tea.KeyMsg); ok && m.Focused {
		switch keyMsg.String() {
		case "w", "W":
			m.ToggleWordWrap()
		case "up", "k":
			m.ScrollUp(1)
		case "down", "j":
			m.ScrollDown(1)
		case "left", "h":
			m.ScrollLeft(4)
		case "right", "l":
			m.ScrollRight(4)
		case "0":
			m.ScrollCol = 0
		case "pgup":
			m.ScrollUp(max(1, m.Height/2))
		case "pgdown":
			m.ScrollDown(max(1, m.Height/2))
		case "home", "g":
			m.ScrollLine = 0
			m.ScrollCol = 0
		case "end", "G":
			m.ScrollLine = max(0, m.Buffer.LineCount-(m.Height-1))
		}
	}

	return *m, nil
}

// ScrollUp scrolls up by n lines.
func (m *Model) ScrollUp(n int) {
	m.ScrollLine = max(0, m.ScrollLine-n)
}

// ScrollDown scrolls down by n lines.
func (m *Model) ScrollDown(n int) {
	contentHeight := max(1, m.Height-1)
	maxScroll := max(0, m.Buffer.LineCount-contentHeight)
	m.ScrollLine = min(maxScroll, m.ScrollLine+n)
}

// ScrollLeft scrolls left by n columns.
func (m *Model) ScrollLeft(n int) {
	m.ScrollCol = max(0, m.ScrollCol-n)
}

// ScrollRight scrolls right by n columns.
func (m *Model) ScrollRight(n int) {
	m.ScrollCol = max(0, m.ScrollCol+n)
}

// View renders the editor component.
func (m *Model) View() string {
	contentHeight := max(1, m.Height-1)

	if !m.Buffer.IsLoaded {
		var blankLines []string
		if m.Buffer.ErrorMessage != "" {
			warnBox := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF5555")).
				Padding(1, 2)

			msg := fmt.Sprintf("⚠️  Cannot preview file (%s):\n\n%s\n\n- Press ^F to select a text file\n- Press ^N to create a new file", m.Buffer.FileName(), m.Buffer.ErrorMessage)
			rendered := warnBox.Render(msg)
			blankLines = append(blankLines, strings.Split(rendered, "\n")...)
		} else {
			welcomeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Padding(1, 1)
			welcomeText := "Welcome to TIDE\n\n- Select a file from sidebar (^F)\n- Press ^N to create a new file\n- Press ^B to build / ^R to run\n- Press ^G to ask Gemini"
			rendered := welcomeStyle.Render(welcomeText)
			welcomeLines := strings.Split(rendered, "\n")
			blankLines = append(blankLines, welcomeLines...)
		}
		for len(blankLines) < m.Height {
			blankLines = append(blankLines, "")
		}
		if len(blankLines) > m.Height {
			blankLines = blankLines[:m.Height]
		}
		for i, line := range blankLines {
			if m.Width > 0 && ansi.StringWidth(line) > m.Width {
				blankLines[i] = ansi.Truncate(line, m.Width, "")
			}
		}
		return strings.Join(blankLines, "\n")
	}

	var content string
	if m.Mode == ModeEdit {
		content = m.Textarea.View()
	} else {
		content = HighlightCode(
			m.Buffer.FilePath,
			m.Buffer.CurrentText,
			m.Theme,
			m.Diagnostics,
			m.ScrollLine,
			contentHeight,
			m.Width,
			m.WordWrap,
			m.ScrollCol,
		)
	}

	// Ensure exact height and width for code lines
	renderedLines := strings.Split(content, "\n")
	for len(renderedLines) < contentHeight {
		renderedLines = append(renderedLines, "")
	}
	if len(renderedLines) > contentHeight {
		renderedLines = renderedLines[:contentHeight]
	}

	// Truncate every code line to m.Width
	for idx, line := range renderedLines {
		if m.Width > 0 && ansi.StringWidth(line) > m.Width {
			renderedLines[idx] = ansi.Truncate(line, m.Width, "")
		}
	}

	// Mode / Status bar at bottom of editor pane
	var modeBadge string
	if m.Mode == ModeEdit {
		modeBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("#FFB86C")).
			Foreground(lipgloss.Color("#282A36")).
			Bold(true).
			Render(" EDIT ") + " " +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#F1FA8C")).Render("^S Save  ^E View")
	} else {
		wrapStatus := "w Wrap [OFF]"
		if m.WordWrap {
			wrapStatus = "w Wrap [ON]"
		}
		modeBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("#BD93F9")).
			Foreground(lipgloss.Color("#282A36")).
			Bold(true).
			Render(" VIEW ") + " " +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render(fmt.Sprintf("^E Edit  %s  ↑/↓/←/→ Scroll", wrapStatus))
	}

	infoText := fmt.Sprintf("[%s | %d lines]", m.Buffer.Language, m.Buffer.LineCount)
	if m.WordWrap {
		infoText = fmt.Sprintf("[%s | %d lines | Wrap]", m.Buffer.Language, m.Buffer.LineCount)
	} else if m.ScrollCol > 0 {
		infoText = fmt.Sprintf("[%s | %d lines | Col %d]", m.Buffer.Language, m.Buffer.LineCount, m.ScrollCol)
	}

	langBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#50FA7B")).
		Render(infoText)

	statusBar := lipgloss.JoinHorizontal(lipgloss.Top, modeBadge, "  ", langBadge)
	if m.Width > 0 && ansi.StringWidth(statusBar) > m.Width {
		statusBar = ansi.Truncate(statusBar, m.Width, "")
	}

	renderedLines = append(renderedLines, statusBar)
	if len(renderedLines) > m.Height {
		renderedLines = renderedLines[:m.Height]
	}

	return strings.Join(renderedLines, "\n")
}
