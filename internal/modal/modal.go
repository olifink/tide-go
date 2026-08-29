package modal

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ModalType specifies the active dialog.
type ModalType int

const (
	None ModalType = iota
	NewFile
	ShellCommand
	GeminiPrompt
	APIKey
)

// Model represents a popover dialog modal.
type Model struct {
	Type        ModalType
	Title       string
	Description string
	Input       textinput.Model
	Width       int
	Height      int
	Active      bool
}

// New creates an unactivated modal dialog.
func New() Model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	return Model{
		Type:  None,
		Input: ti,
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

// OpenGeminiPrompt activates the Gemini assistant prompt dialog.
func (m *Model) OpenGeminiPrompt(hasContext bool) {
	m.Type = GeminiPrompt
	m.Title = "Ask Gemini AI Assistant"
	if hasContext {
		m.Description = "Ask a question (Active file & compiler output attached):"
	} else {
		m.Description = "Ask a question:"
	}
	m.Input.SetValue("")
	m.Input.Placeholder = "e.g. Why is this error happening? How to fix it?"
	m.Input.EchoMode = textinput.EchoNormal
	m.Input.Focus()
	m.Active = true
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

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(m.Title),
		descStyle.Render(m.Description),
		m.Input.View(),
		hintStyle.Render("[Enter] Confirm  •  [Esc] Cancel"),
	)

	return boxStyle.Render(content)
}
