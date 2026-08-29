package console

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"tide/internal/runner"
)

// OutputType defines what kind of message is displayed in console.
type OutputType int

const (
	TypeGeneral OutputType = iota
	TypeCommand
	TypeBuild
	TypeAI
)

// Entry represents an item in the console history.
type Entry struct {
	Type      OutputType
	Header    string
	Content   string
	Timestamp time.Time
	ExitCode  int
	Duration  time.Duration
	IsError   bool
}

// Model represents the Console / Output pane.
type Model struct {
	Viewport     viewport.Model
	Entries      []Entry
	Width        int
	Height       int
	Focused      bool
	IsRunning    bool
	RunningTitle string
	LastOutput   string // Store raw output for Gemini context
	LastCommand  string
	glamourRend  *glamour.TermRenderer
}

// New creates a new console model.
func New(width, height int) Model {
	vp := viewport.New(width, max(1, height))
	vp.SetContent("Console ready. Press ^R to build, ^X for shell command, ^G for Gemini AI.")

	rend, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(max(20, width-4)),
	)

	return Model{
		Viewport:    vp,
		Width:       width,
		Height:      height,
		glamourRend: rend,
	}
}

// SetSize updates console dimensions.
func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	vpHeight := max(1, height)
	m.Viewport.Width = width
	m.Viewport.Height = vpHeight

	rend, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(max(20, width-4)),
	)
	m.glamourRend = rend
	m.rebuildContent()
}

// AddEntry adds a new entry to the console.
func (m *Model) AddEntry(entry Entry) {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	m.Entries = append(m.Entries, entry)
	m.LastOutput = entry.Content
	m.rebuildContent()
	m.Viewport.GotoBottom()
}

// AddCommandResult adds process execution results.
func (m *Model) AddCommandResult(msg runner.ProcessFinishedMsg, isBuild bool) {
	entryType := TypeCommand
	if isBuild {
		entryType = TypeBuild
	}

	header := fmt.Sprintf("$ %s", msg.Command)
	content := msg.Output

	if len(msg.Diagnostics) > 0 {
		errorCount := 0
		warnCount := 0
		for _, d := range msg.Diagnostics {
			if d.Severity == "error" {
				errorCount++
			} else {
				warnCount++
			}
		}
		summary := fmt.Sprintf("\n[Found %d error(s), %d warning(s)]", errorCount, warnCount)
		content += summary
	} else if msg.ExitCode == 0 {
		if content == "" {
			content = "✓ Command succeeded with no output."
		}
	}

	m.LastCommand = msg.Command
	m.AddEntry(Entry{
		Type:      entryType,
		Header:    header,
		Content:   content,
		Timestamp: time.Now(),
		ExitCode:  msg.ExitCode,
		Duration:  msg.Duration,
		IsError:   msg.ExitCode != 0,
	})
}

// AddAIChunk appends or updates the active AI response.
func (m *Model) AddAIChunk(chunk string, isNew bool) {
	if isNew || len(m.Entries) == 0 || m.Entries[len(m.Entries)-1].Type != TypeAI {
		m.Entries = append(m.Entries, Entry{
			Type:      TypeAI,
			Header:    "✦ Gemini AI Assistant",
			Content:   chunk,
			Timestamp: time.Now(),
		})
	} else {
		m.Entries[len(m.Entries)-1].Content += chunk
	}
	m.LastOutput = m.Entries[len(m.Entries)-1].Content
	m.rebuildContent()
	m.Viewport.GotoBottom()
}

// Clear removes all console entries.
func (m *Model) Clear() {
	m.Entries = nil
	m.LastOutput = ""
	m.Viewport.SetContent("Console cleared.")
}

func (m *Model) rebuildContent() {
	var renderedBlocks []string

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD"))
	successHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#50FA7B"))
	errorHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5555"))
	aiHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#BD93F9"))

	for _, entry := range m.Entries {
		var headerStr string
		switch entry.Type {
		case TypeAI:
			headerStr = aiHeader.Render(entry.Header)
		case TypeBuild, TypeCommand:
			statusTag := ""
			if entry.Duration > 0 {
				durStr := entry.Duration.Round(time.Millisecond).String()
				if entry.IsError {
					statusTag = errorHeader.Render(fmt.Sprintf(" [Failed (exit %d) in %s]", entry.ExitCode, durStr))
				} else {
					statusTag = successHeader.Render(fmt.Sprintf(" [OK in %s]", durStr))
				}
			}
			headerStr = headerStyle.Render(entry.Header) + statusTag
		default:
			headerStr = headerStyle.Render(entry.Header)
		}

		body := entry.Content
		if entry.Type == TypeAI && m.glamourRend != nil {
			md, err := m.glamourRend.Render(entry.Content)
			if err == nil {
				body = strings.TrimSpace(md)
			}
		}

		renderedBlocks = append(renderedBlocks, headerStr+"\n"+body)
	}

	fullText := strings.Join(renderedBlocks, "\n\n"+strings.Repeat("─", max(10, m.Width-4))+"\n\n")
	if fullText == "" {
		fullText = "Console ready. Press ^R to build, ^X for shell command, ^G for Gemini AI."
	}
	m.Viewport.SetContent(fullText)
}

// Update handles tea.Msg for viewport scrolling.
func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return *m, cmd
}

// View renders the console pane content.
func (m *Model) View() string {
	content := m.Viewport.View()
	lines := strings.Split(content, "\n")
	for len(lines) < m.Height {
		lines = append(lines, "")
	}
	if len(lines) > m.Height {
		lines = lines[:m.Height]
	}
	for i, line := range lines {
		if m.Width > 0 && ansi.StringWidth(line) > m.Width {
			lines[i] = ansi.Truncate(line, m.Width, "")
		}
	}
	return strings.Join(lines, "\n")
}
