package modal

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"tide/internal/ai"
)

// ModalType specifies the active dialog.
type ModalType int

const (
	None ModalType = iota
	NewFile
	ShellCommand
	GeminiPrompt
	APIKey
	BuildCommand
	RunCommand
	GitSync
)

// Model represents a popover dialog modal.
type Model struct {
	Type         ModalType
	Title        string
	Description  string
	Input        textinput.Model
	Width        int
	Height       int
	Active       bool
	AIMode       ai.AIMode
	PushToRemote bool
}

// New creates an unactivated modal dialog.
func New() Model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	return Model{
		Type:         None,
		Input:        ti,
		PushToRemote: true,
	}
}

// OpenNewFile activates the new file modal dialog.
func (m *Model) OpenNewFile() {
	m.Type = NewFile
	m.Title = "Create New File"
	m.Description = "Enter path/filename (e.g. main.go, src/utils.c):"
	m.Input.SetValue("")
	m.Input.Placeholder = "filename.ext"
	m.Input.EchoMode = textinput.EchoNormal
	m.Input.Focus()
	m.Active = true
}

// OpenShellCommand activates the arbitrary shell runner modal dialog.
func (m *Model) OpenShellCommand(defaultCmd string) {
	m.Type = ShellCommand
	m.Title = "Run Shell Command"
	m.Description = "Enter arbitrary CLI command to execute:"
	m.Input.SetValue(defaultCmd)
	m.Input.Placeholder = "e.g. go test ./..., git status, make"
	m.Input.EchoMode = textinput.EchoNormal
	m.Input.Focus()
	m.Active = true
}

// OpenBuildCommand activates the modal dialog to configure and run a custom build command.
func (m *Model) OpenBuildCommand(defaultCmd string) {
	m.Type = BuildCommand
	m.Title = "Configure & Run Build Command"
	m.Description = "Enter build command for this session (saved for future ^B):"
	m.Input.SetValue(defaultCmd)
	m.Input.Placeholder = "e.g. go build -v ./..., make all, cargo build"
	m.Input.EchoMode = textinput.EchoNormal
	m.Input.Focus()
	m.Active = true
}

// OpenRunCommand activates the modal dialog to configure and run a custom run command.
func (m *Model) OpenRunCommand(defaultCmd string) {
	m.Type = RunCommand
	m.Title = "Configure & Run Command"
	m.Description = "Enter run command for this session (saved for future ^R):"
	m.Input.SetValue(defaultCmd)
	m.Input.Placeholder = "e.g. ./main --dev, python3 script.py, make run"
	m.Input.EchoMode = textinput.EchoNormal
	m.Input.Focus()
	m.Active = true
}

// OpenGitSync activates the git add, commit, and push modal dialog.
func (m *Model) OpenGitSync(branch string, changesCount int) {
	m.Type = GitSync
	if branch == "" {
		branch = "main"
	}
	m.Title = fmt.Sprintf("Git: Add & Commit (%s)", branch)
	if changesCount > 0 {
		m.Description = fmt.Sprintf("Enter commit message for %d changed file(s):", changesCount)
	} else {
		m.Description = "Enter commit message (will stage all changes):"
	}
	m.PushToRemote = true
	m.Input.SetValue("")
	m.Input.Placeholder = "e.g. feat: add new feature, fix: bug description"
	m.Input.EchoMode = textinput.EchoNormal
	m.Input.Focus()
	m.Active = true
}

// TogglePush toggles the PushToRemote flag for GitSync modal.
func (m *Model) TogglePush() {
	m.PushToRemote = !m.PushToRemote
}

// OpenGeminiPrompt activates the Gemini assistant prompt dialog according to active pane mode.
func (m *Model) OpenGeminiPrompt(mode ai.AIMode, activeFileName string) {
	m.Type = GeminiPrompt
	m.AIMode = mode
	m.Input.SetValue("")
	m.Input.EchoMode = textinput.EchoNormal
	m.Input.Focus()
	m.Active = true

	switch mode {
	case ai.ModeUpdateFile:
		if activeFileName == "" {
			activeFileName = "current file"
		}
		m.Title = fmt.Sprintf("✦ Gemini: Update %s", activeFileName)
		m.Description = fmt.Sprintf("Describe changes or fixes to apply to %s (applied directly to editor):", activeFileName)
		m.Input.Placeholder = "e.g. Add error handling, refactor function X, fix bug"

	case ai.ModeGenerateFile:
		m.Title = "✦ Gemini: Generate New File"
		m.Description = "Describe new file to create (e.g. \"server.go: REST API with health check\"):"
		m.Input.Placeholder = "filename.ext: description of file to generate"

	default: // ai.ModeConsoleQA
		m.Title = "✦ Gemini: Ask Assistant"
		m.Description = "Ask a question or explain errors (answers stream to console):"
		m.Input.Placeholder = "e.g. Why is this error happening? How does this work?"
	}
}

// OpenAPIKey activates the Gemini API key input dialog.
func (m *Model) OpenAPIKey() {
	m.Type = APIKey
	m.Title = "Gemini API Key Required"
	m.Description = "Enter your Google Gemini API Key (saved to ~/.config/tide):"
	m.Input.SetValue("")
	m.Input.Placeholder = "AIzaSy..."
	m.Input.EchoMode = textinput.EchoPassword
	m.Input.Focus()
	m.Active = true
}

// Clear clears the current input text without closing the dialog.
func (m *Model) Clear() {
	m.Input.SetValue("")
}

// Close deactivates the modal.
func (m *Model) Close() {
	m.Active = false
	m.Type = None
	m.Input.Blur()
	m.Input.SetValue("")
}

// Value returns the current text in the input.
func (m *Model) Value() string {
	return m.Input.Value()
}

// Update handles tea.Msg for input text.
func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.Active {
		return *m, nil
	}
	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return *m, cmd
}

// SetSize updates screen size for modal positioning.
func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	inputWidth := min(60, max(30, width-12))
	m.Input.Width = inputWidth
}

// View renders the modal centered or floating.
func (m *Model) View() string {
	if !m.Active {
		return ""
	}

	dialogWidth := min(68, max(40, m.Width-6))

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#BD93F9")).
		MarginBottom(1)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F8F8F2")).
		MarginBottom(1)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6272A4")).
		MarginTop(1)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#BD93F9")).
		Background(lipgloss.Color("#21222C")).
		Padding(1, 2).
		Width(dialogWidth)

	escHint := "[Esc] Close"
	if m.Input.Value() != "" {
		escHint = "[Esc] Clear"
	}

	confirmHint := "[Enter] Confirm"
	if m.Type == GitSync {
		if m.PushToRemote {
			confirmHint = "[Enter] Commit & Push"
		} else {
			confirmHint = "[Enter] Commit (Local)  •  [Ctrl+Enter] Push"
		}
	}

	var elements []string
	elements = append(elements,
		titleStyle.Render(m.Title),
		descStyle.Render(m.Description),
		m.Input.View(),
	)

	if m.Type == GitSync {
		checkboxBox := "[ ]"
		checkColor := lipgloss.Color("#A6ADC8")
		if m.PushToRemote {
			checkboxBox = "[x]"
			checkColor = lipgloss.Color("#A6E3A1")
		}
		renderedCheckbox := lipgloss.NewStyle().Foreground(checkColor).Bold(true).Render(checkboxBox) +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4")).Render(" Push to remote") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render(" (Tab to toggle)")

		elements = append(elements, lipgloss.NewStyle().MarginTop(1).Render(renderedCheckbox))
	}

	elements = append(elements, hintStyle.Render(fmt.Sprintf("%s  •  %s", confirmHint, escHint)))

	content := lipgloss.JoinVertical(lipgloss.Left, elements...)

	return boxStyle.Render(content)
}
