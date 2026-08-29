package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"tide/internal/ai"
	"tide/internal/config"
	"tide/internal/console"
	"tide/internal/editor"
	"tide/internal/filetree"
	"tide/internal/modal"
	"tide/internal/runner"
)

// Pane represents which major UI section currently has keyboard focus.
type Pane int

const (
	PaneFiles Pane = iota
	PaneEditor
	PaneConsole
)

// Model is the top-level Bubble Tea model for TIDE.
type Model struct {
	Config        config.Config
	WorkingDir    string
	ActivePane    Pane
	FileTree      filetree.Model
	Editor        editor.Model
	Console       console.Model
	Modal         modal.Model
	Keys          KeyMap
	Width         int
	Height        int
	Diagnostics   []runner.Diagnostic
	StatusMessage string
	AIChannel     chan ai.AIChunkMsg
	pendingAIQ    string
}

// InitialModel initializes the TIDE application model.
func InitialModel(startPath string) Model {
	cfg := config.Load()

	absDir, err := filepath.Abs(startPath)
	if err != nil {
		absDir = "."
	}

	initialFile := ""
	// If starting path is a file, set working dir to its directory
	stat, err := os.Stat(absDir)
	if err == nil && !stat.IsDir() {
		initialFile = absDir
		absDir = filepath.Dir(absDir)
	}

	ft := filetree.New(absDir, 26, 20)
	ed := editor.New(cfg.ChromaTheme, 60, 20)
	con := console.New(80, 8)
	mod := modal.New()

	m := Model{
		Config:     cfg,
		WorkingDir: absDir,
		ActivePane: PaneFiles,
		FileTree:   ft,
		Editor:     ed,
		Console:    con,
		Modal:      mod,
		Keys:       DefaultKeyMap(),
	}

	if initialFile != "" {
		_ = m.Editor.OpenFile(initialFile)
		m.FileTree.SelectFile(initialFile)
		m.ActivePane = PaneEditor
	} else if len(ft.VisibleItems) > 0 {
		// Auto-open first file if available
		for _, item := range ft.VisibleItems {
			if !item.IsDir {
				_ = m.Editor.OpenFile(item.Path)
				m.FileTree.SelectFile(item.Path)
				break
			}
		}
	}

	m.updateFocus()
	return m
}

// Init sets up initial Bubble Tea commands.
func (m Model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m *Model) updateFocus() {
	m.FileTree.Focused = (m.ActivePane == PaneFiles && !m.Modal.Active)
	m.Editor.Focused = (m.ActivePane == PaneEditor && !m.Modal.Active)
	m.Console.Focused = (m.ActivePane == PaneConsole && !m.Modal.Active)
}

// Update handles all events and state transitions.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.recalculateLayout()
		return m, nil

	case runner.ProcessFinishedMsg:
		m.Console.IsRunning = false
		isBuild := strings.HasPrefix(msg.Command, "go build") || strings.HasPrefix(msg.Command, "make") || strings.HasPrefix(msg.Command, "gcc") || strings.HasPrefix(msg.Command, "g++")
		m.Console.AddCommandResult(msg, isBuild)
		m.Diagnostics = msg.Diagnostics
		m.Editor.SetDiagnostics(m.Diagnostics)
		if len(msg.Diagnostics) > 0 {
			m.StatusMessage = fmt.Sprintf("Build finished with %d diagnostic(s)", len(msg.Diagnostics))
		} else if msg.ExitCode == 0 {
			m.StatusMessage = "Build/Run succeeded!"
		} else {
			m.StatusMessage = fmt.Sprintf("Process exited with code %d", msg.ExitCode)
		}
		return m, nil

	case ai.AIChunkMsg:
		if msg.Err != nil {
			m.Console.IsRunning = false
			m.Console.AddEntry(console.Entry{
				Type:     console.TypeAI,
				Header:   "✦ Gemini Assistant [Error]",
				Content:  msg.Err.Error(),
				IsError:  true,
				ExitCode: 1,
			})
			m.StatusMessage = "Gemini request failed"
			return m, nil
		}
		if msg.Done {
			m.Console.IsRunning = false
			m.StatusMessage = "Gemini finished response"
			return m, nil
		}
		m.Console.AddAIChunk(msg.Chunk, false)
		return m, ai.ListenForAIChunk(m.AIChannel)

	case tea.KeyMsg:
		// 1. Handle Modal Input when active
		if m.Modal.Active {
			switch msg.String() {
			case "esc":
				m.Modal.Close()
				m.updateFocus()
				return m, nil

			case "enter":
				val := strings.TrimSpace(m.Modal.Value())
				modalType := m.Modal.Type
				m.Modal.Close()
				m.updateFocus()

				if val == "" {
					return m, nil
				}

				switch modalType {
				case modal.NewFile:
					fullPath := val
					if !filepath.IsAbs(fullPath) {
						fullPath = filepath.Join(m.WorkingDir, val)
					}
					// Ensure parent dir exists
					_ = os.MkdirAll(filepath.Dir(fullPath), 0755)
					// Create file
					if _, err := os.Stat(fullPath); os.IsNotExist(err) {
						_ = os.WriteFile(fullPath, []byte(""), 0644)
					}
					m.FileTree.Refresh()
					_ = m.Editor.OpenFile(fullPath)
					m.FileTree.SelectFile(fullPath)
					m.ActivePane = PaneEditor
					m.updateFocus()
					m.StatusMessage = fmt.Sprintf("Created & opened %s", filepath.Base(fullPath))

				case modal.ShellCommand:
					m.Console.IsRunning = true
					m.Console.RunningTitle = val
					m.StatusMessage = fmt.Sprintf("Running: %s", val)
					return m, runner.RunCommandCmd(m.WorkingDir, val)

				case modal.GeminiPrompt:
					apiKey := config.GetGeminiAPIKey(m.Config)
					if apiKey == "" {
						m.pendingAIQ = val
						m.Modal.OpenAPIKey()
						m.updateFocus()
						return m, nil
					}
					return m, m.triggerGeminiAI(val, apiKey)

				case modal.APIKey:
					_ = config.SaveGeminiAPIKey(val)
					m.Config.GeminiAPIKey = val
					m.StatusMessage = "Gemini API key saved!"
					if m.pendingAIQ != "" {
						q := m.pendingAIQ
						m.pendingAIQ = ""
						return m, m.triggerGeminiAI(q, val)
					}
				}
				return m, nil
			}

			var cmd tea.Cmd
			m.Modal, cmd = m.Modal.Update(msg)
			return m, cmd
		}

		// 2. Global Hotkeys (when modal is not active)
		switch msg.String() {
		case "ctrl+q":
			return m, tea.Quit

		case "ctrl+f":
			if m.ActivePane == PaneFiles {
				m.ActivePane = PaneEditor
			} else {
				m.ActivePane = PaneFiles
			}
			m.updateFocus()
			return m, nil

		case "ctrl+n":
			m.Modal.OpenNewFile()
			m.updateFocus()
			return m, nil

		case "ctrl+e":
			m.ActivePane = PaneEditor
			m.Editor.ToggleMode()
			m.updateFocus()
			return m, nil

		case "ctrl+s":
			if m.Editor.Buffer.IsLoaded {
				if err := m.Editor.SaveFile(); err != nil {
					m.StatusMessage = fmt.Sprintf("Error saving: %s", err.Error())
				} else {
					m.StatusMessage = fmt.Sprintf("Saved %s", m.Editor.Buffer.FileName())
				}
			}
			return m, nil

		case "ctrl+r":
			target := runner.DetectBuildTarget(m.WorkingDir, m.Editor.Buffer.FilePath)
			cmdToRun := target.Command
			if cmdToRun == "" {
				cmdToRun = "go build ."
			}
			m.Console.IsRunning = true
			m.Console.RunningTitle = cmdToRun
			m.StatusMessage = fmt.Sprintf("Building (%s)...", target.Description)
			return m, runner.RunCommandCmd(m.WorkingDir, cmdToRun)

		case "ctrl+x":
			defaultCmd := ""
			if m.Console.LastCommand != "" {
				defaultCmd = m.Console.LastCommand
			}
			m.Modal.OpenShellCommand(defaultCmd)
			m.updateFocus()
			return m, nil

		case "ctrl+g":
			apiKey := config.GetGeminiAPIKey(m.Config)
			if apiKey == "" {
				m.Modal.OpenAPIKey()
			} else {
				m.Modal.OpenGeminiPrompt(m.Editor.Buffer.IsLoaded)
			}
			m.updateFocus()
			return m, nil

		case "tab":
			// If editor is in edit mode, allow typing tabs unless Shift+Tab / Pane cycle
			if m.Editor.Mode == editor.ModeEdit && m.ActivePane == PaneEditor {
				// pass tab to editor
				break
			}
			m.ActivePane = (m.ActivePane + 1) % 3
			m.updateFocus()
			return m, nil

		case "shift+tab":
			m.ActivePane = (m.ActivePane + 2) % 3
			m.updateFocus()
			return m, nil

		case "esc":
			if m.Editor.Mode == editor.ModeEdit {
				m.Editor.ToggleMode()
				return m, nil
			}
			if m.ActivePane != PaneEditor {
				m.ActivePane = PaneEditor
				m.updateFocus()
				return m, nil
			}
		}

		// 3. Pane-Specific Keys
		switch m.ActivePane {
		case PaneFiles:
			switch msg.String() {
			case "up", "k":
				m.FileTree.MoveUp()
				return m, nil
			case "down", "j":
				m.FileTree.MoveDown()
				return m, nil
			case "pgup":
				m.FileTree.PageUp()
				return m, nil
			case "pgdown":
				m.FileTree.PageDown()
				return m, nil
			case "home":
				m.FileTree.Home()
				return m, nil
			case "end":
				m.FileTree.End()
				return m, nil
			case "enter":
				selectedPath, isFile := m.FileTree.ToggleCurrent()
				if isFile && selectedPath != "" {
					_ = m.Editor.OpenFile(selectedPath)
					m.Editor.SetDiagnostics(m.Diagnostics)
					m.FileTree.SelectFile(selectedPath)
					m.ActivePane = PaneEditor
					m.updateFocus()
				}
				return m, nil
			case "r":
				m.FileTree.Refresh()
				m.StatusMessage = "File list refreshed"
				return m, nil
			}

		case PaneEditor:
			var edCmd tea.Cmd
			m.Editor, edCmd = m.Editor.Update(msg)
			cmds = append(cmds, edCmd)

		case PaneConsole:
			switch msg.String() {
			case "c":
				m.Console.Clear()
				m.StatusMessage = "Console cleared"
				return m, nil
			}
			var conCmd tea.Cmd
			m.Console, conCmd = m.Console.Update(msg)
			cmds = append(cmds, conCmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) triggerGeminiAI(userQuery string, apiKey string) tea.Cmd {
	m.Console.IsRunning = true
	m.Console.RunningTitle = "Gemini AI stream..."
	m.StatusMessage = "Asking Gemini..."

	fullPrompt := ai.BuildPrompt(
		userQuery,
		m.Editor.Buffer.FilePath,
		m.Editor.Buffer.CurrentText,
		m.Console.LastCommand,
		m.Console.LastOutput,
	)

	m.Console.AddAIChunk("✦ **Gemini AI:**\n\n", true)

	m.AIChannel = make(chan ai.AIChunkMsg, 32)
	go ai.AskGeminiStream(
		context.Background(),
		apiKey,
		m.Config.GeminiModel,
		fullPrompt,
		m.AIChannel,
	)

	return ai.ListenForAIChunk(m.AIChannel)
}

func (m *Model) recalculateLayout() {
	if m.Width == 0 || m.Height == 0 {
		return
	}

	headerH := 1
	footerH := 1
	bodyTotalH := max(4, m.Height-headerH-footerH)

	// Allocate 65-70% to Main Split, remainder (min 6 rows) to Console
	consoleH := max(6, int(float64(bodyTotalH)*0.28))
	mainH := max(4, bodyTotalH-consoleH)

	// Fixed width for file explorer (24-28 chars)
	sidebarW := min(30, max(20, int(float64(m.Width)*0.22)))
	editorW := max(20, m.Width-sidebarW)

	// Update components
	m.FileTree.SetSize(sidebarW-2, mainH-2)
	m.Editor.SetSize(editorW-2, mainH-2)
	m.Console.SetSize(m.Width-2, consoleH-2)
	m.Modal.SetSize(m.Width, m.Height)
}

// View renders the complete TIDE interface.
func (m Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing TIDE..."
	}

	headerH := 1
	footerH := 1
	bodyTotalH := max(4, m.Height-headerH-footerH)
	consoleH := max(6, int(float64(bodyTotalH)*0.28))
	mainH := max(4, bodyTotalH-consoleH)

	sidebarW := min(30, max(20, int(float64(m.Width)*0.22)))
	editorW := max(20, m.Width-sidebarW)

	// 1. Header Bar
	logo := HeaderLogoStyle.Render("🌊 TIDE v0.1")

	dirDisplay := m.WorkingDir
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(dirDisplay, home) {
		dirDisplay = "~" + strings.TrimPrefix(dirDisplay, home)
	}
	dirBadge := HeaderDirStyle.Render(fmt.Sprintf("Dir: %s", dirDisplay))

	fileName := m.Editor.Buffer.FileName()
	activeFileBadge := HeaderFileStyle.Render(fmt.Sprintf("Active: %s", fileName))

	modBadge := ""
	if m.Editor.Buffer.IsModified {
		modBadge = HeaderModStyle.Render("[MOD]")
	}

	statusText := ""
	if m.StatusMessage != "" {
		statusText = lipgloss.NewStyle().Foreground(ColorSuccess).Render(" • " + m.StatusMessage)
	}

	headerLeft := lipgloss.JoinHorizontal(lipgloss.Center, logo, dirBadge, activeFileBadge, modBadge, statusText)
	headerBar := lipgloss.NewStyle().
		Width(m.Width).
		Background(ColorBg).
		Render(headerLeft)

	// 2. File Sidebar Box
	sidebarBorder := InactiveBorderStyle
	sidebarTitle := PanelTitleInactive.Render("── FILES ──")
	if m.ActivePane == PaneFiles {
		sidebarBorder = ActiveBorderStyle
		sidebarTitle = PanelTitleActive.Render("── FILES [Active] ──")
	}
	sidebarContent := sidebarTitle + "\n" + m.FileTree.View()
	sidebarBox := sidebarBorder.
		Width(sidebarW - 2).
		Height(mainH - 2).
		Render(sidebarContent)

	// 3. Editor Box
	editorBorder := InactiveBorderStyle
	editorTitleStr := fmt.Sprintf("── %s [%s] ──", fileName, m.Editor.Buffer.Language)
	editorTitle := PanelTitleInactive.Render(editorTitleStr)
	if m.ActivePane == PaneEditor {
		editorBorder = ActiveBorderStyle
		editorTitle = PanelTitleActive.Render(fmt.Sprintf("── %s [%s] (Active) ──", fileName, m.Editor.Buffer.Language))
	}
	editorContent := editorTitle + "\n" + m.Editor.View()
	editorBox := editorBorder.
		Width(editorW - 2).
		Height(mainH - 2).
		Render(editorContent)

	mainSplit := lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, editorBox)

	// 4. Console Box
	consoleBorder := InactiveBorderStyle
	if m.ActivePane == PaneConsole {
		consoleBorder = ActiveBorderStyle
	}
	consoleBox := consoleBorder.
		Width(m.Width - 2).
		Height(consoleH - 2).
		Render(m.Console.View())

	// 5. Pico Footer Bar
	keys := []struct {
		key  string
		desc string
	}{
		{"^F", "Files"},
		{"^N", "New"},
		{"^E", "Edit"},
		{"^S", "Save"},
		{"^R", "Run/Build"},
		{"^X", "Shell"},
		{"^G", "Gemini"},
		{"^Q", "Quit"},
	}

	var keyItems []string
	for _, k := range keys {
		renderedKey := FooterKeyStyle.Render(" " + k.key + " ")
		renderedDesc := FooterDescStyle.Render(k.desc)
		keyItems = append(keyItems, renderedKey+renderedDesc)
	}

	footerContent := strings.Join(keyItems, " ")
	footerBar := lipgloss.NewStyle().
		Width(m.Width).
		Background(ColorSurface).
		Render(footerContent)

	// Combine all vertically
	fullScreen := lipgloss.JoinVertical(
		lipgloss.Left,
		headerBar,
		mainSplit,
		consoleBox,
		footerBar,
	)

	// 6. Overlay Modal if active
	if m.Modal.Active {
		modalView := m.Modal.View()
		return lipgloss.Place(
			m.Width,
			m.Height,
			lipgloss.Center,
			lipgloss.Center,
			modalView,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(ColorOverlay),
		)
	}

	return fullScreen
}
