package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	pendingAIQ       string
	activeAIMode     ai.AIMode
	aiResponse       string
	ConsoleMaximized bool
	EditorFullscreen bool
	CustomBuildCmd   string
	CustomRunCmd     string
}

// InitialModel initializes the TIDE application model.
func InitialModel(startPath string) Model {
	cfg := config.Load()

	absPath, err := filepath.Abs(startPath)
	if err != nil {
		absPath = "."
	}

	var workingDir string
	var initialFile string
	var statusMsg string

	stat, err := os.Stat(absPath)
	if err == nil {
		// Path exists
		if stat.IsDir() {
			workingDir = absPath
		} else {
			initialFile = absPath
			workingDir = filepath.Dir(absPath)
		}
	} else {
		// Path does not exist on disk
		if strings.HasSuffix(startPath, "/") || strings.HasSuffix(startPath, string(filepath.Separator)) {
			_ = os.MkdirAll(absPath, 0755)
			workingDir = absPath
		} else {
			// Treat as a file to be created in its target directory
			targetDir := filepath.Dir(absPath)
			_ = os.MkdirAll(targetDir, 0755)
			_ = os.WriteFile(absPath, []byte(""), 0644)
			initialFile = absPath
			workingDir = targetDir
			statusMsg = fmt.Sprintf("Created & opened %s", filepath.Base(absPath))
		}
	}

	ft := filetree.New(workingDir, 26, 20)
	ed := editor.New(cfg.ChromaTheme, 60, 20)
	ed.WordWrap = cfg.WordWrap
	con := console.New(80, 8)
	mod := modal.New()

	m := Model{
		Config:        cfg,
		WorkingDir:    workingDir,
		ActivePane:    PaneFiles,
		FileTree:      ft,
		Editor:        ed,
		Console:       con,
		Modal:         mod,
		Keys:          DefaultKeyMap(),
		StatusMessage: statusMsg,
	}

	if initialFile != "" {
		_ = m.Editor.OpenFile(initialFile)
		m.FileTree.SelectFile(initialFile)
		if m.Editor.Buffer.IsLoaded {
			m.ActivePane = PaneEditor
			m.EditorFullscreen = true
		}
	} else if len(ft.VisibleItems) > 0 {
		// Auto-open first valid text file within size limits if available
		for _, item := range ft.VisibleItems {
			if !item.IsDir {
				isText, _ := editor.IsTextFile(item.Path)
				if isText {
					_ = m.Editor.OpenFile(item.Path)
					m.FileTree.SelectFile(item.Path)
					break
				}
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
		m.ActivePane = PaneConsole
		m.updateFocus()
		isBuild := strings.HasPrefix(msg.Command, "go build") || strings.HasPrefix(msg.Command, "make") || strings.HasPrefix(msg.Command, "gcc") || strings.HasPrefix(msg.Command, "g++") || strings.HasPrefix(msg.Command, "cargo build") || strings.HasPrefix(msg.Command, "rustc")
		m.Console.AddCommandResult(msg, isBuild)
		m.Diagnostics = msg.Diagnostics
		m.Editor.SetDiagnostics(m.Diagnostics)
		m.refreshWorkspace()
		if len(msg.Diagnostics) > 0 {
			m.StatusMessage = fmt.Sprintf("Build finished with %d diagnostic(s)", len(msg.Diagnostics))
		} else if msg.ExitCode == 0 {
			if isBuild {
				m.StatusMessage = "Build succeeded! (Press ^R to run)"
			} else {
				m.StatusMessage = "Run finished successfully (exit 0)"
			}
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
			rawResp := m.aiResponse

			switch msg.Mode {
			case ai.ModeUpdateFile:
				_, code, explanation := ai.ExtractCodeAndFile(rawResp, m.Editor.Buffer.FileName())
				if code != "" {
					m.Editor.Buffer.SetText(code)
					if m.Editor.Mode == editor.ModeEdit {
						m.Editor.Textarea.SetValue(code)
					}
					m.Editor.Buffer.IsModified = true
					m.ActivePane = PaneEditor
					m.updateFocus()
					m.StatusMessage = fmt.Sprintf("Updated %s with Gemini changes (Press ^S to save)", m.Editor.Buffer.FileName())
					m.Console.AddEntry(console.Entry{
						Type:    console.TypeAI,
						Header:  fmt.Sprintf("✓ Applied Gemini changes to %s", m.Editor.Buffer.FileName()),
						Content: fmt.Sprintf("File updated with %d lines. Review changes in Editor and press ^S to save.\n\n%s", m.Editor.Buffer.LineCount, explanation),
					})
				} else {
					m.StatusMessage = "Gemini response completed"
				}

			case ai.ModeGenerateFile:
				defaultName := "generated.txt"
				if m.Editor.Buffer.Language == "Go" {
					defaultName = "generated.go"
				} else if m.Editor.Buffer.Language == "C" {
					defaultName = "generated.c"
				}
				fn, code, explanation := ai.ExtractCodeAndFile(rawResp, defaultName)
				if code != "" {
					targetPath := filepath.Join(m.WorkingDir, fn)
					_ = os.MkdirAll(filepath.Dir(targetPath), 0755)
					_ = os.WriteFile(targetPath, []byte(code), 0644)
					m.refreshWorkspace()
					_ = m.Editor.OpenFile(targetPath)
					m.FileTree.SelectFile(targetPath)
					m.ActivePane = PaneEditor
					m.updateFocus()
					m.StatusMessage = fmt.Sprintf("Created & opened %s", fn)
					m.Console.AddEntry(console.Entry{
						Type:    console.TypeAI,
						Header:  fmt.Sprintf("✓ Created new file: %s", fn),
						Content: fmt.Sprintf("Wrote %d lines to %s.\n\n%s", len(strings.Split(code, "\n")), fn, explanation),
					})
				} else {
					m.StatusMessage = "Gemini generated response"
				}

			default: // ai.ModeConsoleQA
				m.StatusMessage = "Gemini finished response"
			}

			m.refreshWorkspace()
			return m, nil
		}

		m.aiResponse += msg.Chunk
		m.Console.AddAIChunk(msg.Chunk, false)
		return m, ai.ListenForAIChunk(m.AIChannel)

	case tea.KeyMsg:
		// 1. Handle Modal Input when active
		if m.Modal.Active {
			switch msg.String() {
			case "esc":
				if m.Modal.Value() != "" {
					m.Modal.Clear()
					return m, nil
				}
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
					if m.Editor.Mode == editor.ModeEdit {
						m.Editor.ToggleMode()
					}
					m.ActivePane = PaneConsole
					m.updateFocus()
					m.Console.IsRunning = true
					m.Console.RunningTitle = val
					m.StatusMessage = fmt.Sprintf("Running: %s", val)
					return m, runner.RunCommandCmd(m.WorkingDir, val)

				case modal.BuildCommand:
					m.CustomBuildCmd = val
					if m.Editor.Mode == editor.ModeEdit {
						m.Editor.ToggleMode()
					}
					m.ActivePane = PaneConsole
					m.updateFocus()
					m.Console.IsRunning = true
					m.Console.RunningTitle = val
					m.StatusMessage = fmt.Sprintf("Building: %s", val)
					return m, runner.RunCommandCmd(m.WorkingDir, val)

				case modal.RunCommand:
					m.CustomRunCmd = val
					if m.Editor.Mode == editor.ModeEdit {
						m.Editor.ToggleMode()
					}
					m.ActivePane = PaneConsole
					m.updateFocus()
					m.Console.IsRunning = true
					m.Console.RunningTitle = val
					m.StatusMessage = fmt.Sprintf("Running: %s", val)
					return m, runner.RunCommandCmd(m.WorkingDir, val)

				case modal.GeminiPrompt:
					apiKey := config.GetGeminiAPIKey(m.Config)
					mode := m.Modal.AIMode
					if apiKey == "" {
						m.pendingAIQ = val
						m.activeAIMode = mode
						m.Modal.OpenAPIKey()
						m.updateFocus()
						return m, nil
					}
					return m, m.triggerGeminiAI(val, apiKey, mode)

				case modal.APIKey:
					_ = config.SaveGeminiAPIKey(val)
					m.Config.GeminiAPIKey = val
					m.StatusMessage = "Gemini API key saved!"
					if m.pendingAIQ != "" {
						q := m.pendingAIQ
						mode := m.activeAIMode
						m.pendingAIQ = ""
						return m, m.triggerGeminiAI(q, val, mode)
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

		case "ctrl+z", "f11":
			m.ToggleEditorFullscreen()
			return m, nil

		case "ctrl+f":
			if m.EditorFullscreen {
				m.EditorFullscreen = false
				m.recalculateLayout()
			}
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

		case "ctrl+b":
			if m.Editor.Mode == editor.ModeEdit {
				m.Editor.ToggleMode()
			}
			targetFile := m.Editor.Buffer.FilePath
			if m.ActivePane == PaneFiles && m.FileTree.SelectedItem() != nil {
				targetFile = m.FileTree.SelectedItem().Path
			} else if targetFile == "" && m.FileTree.SelectedItem() != nil {
				targetFile = m.FileTree.SelectedItem().Path
			}
			cmdToRun := m.CustomBuildCmd
			desc := "Custom Build"
			if cmdToRun == "" {
				target := runner.DetectBuildTarget(m.WorkingDir, targetFile)
				cmdToRun = target.Command
				desc = target.Description
			}
			if cmdToRun == "" {
				cmdToRun = "go build ."
			}
			m.ActivePane = PaneConsole
			m.updateFocus()
			m.Console.IsRunning = true
			m.Console.RunningTitle = cmdToRun
			m.StatusMessage = fmt.Sprintf("Building (%s)...", desc)
			return m, runner.RunCommandCmd(m.WorkingDir, cmdToRun)

		case "alt+b":
			targetFile := m.Editor.Buffer.FilePath
			if m.ActivePane == PaneFiles && m.FileTree.SelectedItem() != nil {
				targetFile = m.FileTree.SelectedItem().Path
			} else if targetFile == "" && m.FileTree.SelectedItem() != nil {
				targetFile = m.FileTree.SelectedItem().Path
			}
			currentCmd := m.CustomBuildCmd
			if currentCmd == "" {
				target := runner.DetectBuildTarget(m.WorkingDir, targetFile)
				currentCmd = target.Command
			}
			if currentCmd == "" {
				currentCmd = "go build ."
			}
			m.Modal.OpenBuildCommand(currentCmd)
			m.updateFocus()
			return m, nil

		case "ctrl+r":
			if m.Editor.Mode == editor.ModeEdit {
				m.Editor.ToggleMode()
			}
			targetFile := m.Editor.Buffer.FilePath
			if m.ActivePane == PaneFiles && m.FileTree.SelectedItem() != nil {
				targetFile = m.FileTree.SelectedItem().Path
			} else if targetFile == "" && m.FileTree.SelectedItem() != nil {
				targetFile = m.FileTree.SelectedItem().Path
			}
			cmdToRun := m.CustomRunCmd
			desc := "Custom Run"
			if cmdToRun == "" {
				target := runner.DetectRunTarget(m.WorkingDir, targetFile)
				cmdToRun = target.Command
				desc = target.Description
			}
			if cmdToRun == "" {
				cmdToRun = "./main"
			}
			m.ActivePane = PaneConsole
			m.updateFocus()
			m.Console.IsRunning = true
			m.Console.RunningTitle = cmdToRun
			m.StatusMessage = fmt.Sprintf("Running (%s)...", desc)
			return m, runner.RunCommandCmd(m.WorkingDir, cmdToRun)

		case "alt+r":
			targetFile := m.Editor.Buffer.FilePath
			if m.ActivePane == PaneFiles && m.FileTree.SelectedItem() != nil {
				targetFile = m.FileTree.SelectedItem().Path
			} else if targetFile == "" && m.FileTree.SelectedItem() != nil {
				targetFile = m.FileTree.SelectedItem().Path
			}
			currentCmd := m.CustomRunCmd
			if currentCmd == "" {
				target := runner.DetectRunTarget(m.WorkingDir, targetFile)
				currentCmd = target.Command
			}
			if currentCmd == "" {
				currentCmd = "./main"
			}
			m.Modal.OpenRunCommand(currentCmd)
			m.updateFocus()
			return m, nil

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
			var targetMode ai.AIMode
			switch m.ActivePane {
			case PaneFiles:
				targetMode = ai.ModeGenerateFile
			case PaneEditor:
				if m.Editor.Buffer.IsLoaded {
					targetMode = ai.ModeUpdateFile
				} else {
					targetMode = ai.ModeGenerateFile
				}
			default:
				targetMode = ai.ModeConsoleQA
			}

			if apiKey == "" {
				m.Modal.OpenAPIKey()
			} else {
				m.Modal.OpenGeminiPrompt(targetMode, m.Editor.Buffer.FileName())
			}
			m.updateFocus()
			return m, nil

		case "tab":
			// If editor is in edit mode, route Tab directly into the editor to insert character
			if m.Editor.Mode == editor.ModeEdit && m.ActivePane == PaneEditor {
				var edCmd tea.Cmd
				m.Editor, edCmd = m.Editor.Update(msg)
				return m, edCmd
			}
			if m.EditorFullscreen {
				return m, nil
			}
			m.ActivePane = (m.ActivePane + 1) % 3
			m.updateFocus()
			return m, nil

		case "shift+tab":
			if m.EditorFullscreen {
				return m, nil
			}
			m.ActivePane = (m.ActivePane + 2) % 3
			m.updateFocus()
			return m, nil

		case "shift+esc", "alt+esc":
			if m.EditorFullscreen {
				m.EditorFullscreen = false
				m.recalculateLayout()
			}
			if m.Editor.Mode == editor.ModeEdit {
				m.Editor.ToggleMode()
			}
			m.ActivePane = PaneConsole
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
				selectedPath, isFile, _ := m.FileTree.ToggleCurrent()
				if isFile && selectedPath != "" {
					_ = m.Editor.OpenFile(selectedPath)
					m.Editor.SetDiagnostics(m.Diagnostics)
					m.FileTree.SelectFile(selectedPath)
					m.ActivePane = PaneEditor
					m.updateFocus()
				}
				return m, nil
			case "e", "E":
				selectedPath, isFile, _ := m.FileTree.ToggleCurrent()
				if isFile && selectedPath != "" {
					_ = m.Editor.OpenFile(selectedPath)
					m.Editor.SetDiagnostics(m.Diagnostics)
					m.FileTree.SelectFile(selectedPath)
					m.ActivePane = PaneEditor
					if m.Editor.Mode != editor.ModeEdit {
						m.Editor.ToggleMode()
					}
					m.updateFocus()
				} else if m.Editor.Buffer.IsLoaded {
					m.ActivePane = PaneEditor
					if m.Editor.Mode != editor.ModeEdit {
						m.Editor.ToggleMode()
					}
					m.updateFocus()
				}
				return m, nil
			case ".":
				m.FileTree.ToggleHidden()
				if m.FileTree.ShowHidden {
					m.StatusMessage = "Showing hidden files (. to toggle)"
				} else {
					m.StatusMessage = "Hiding hidden files (. to toggle)"
				}
				return m, nil
			case "alt+backspace", "alt+delete", "alt+ctrl+h":
				if item := m.FileTree.SelectedItem(); item != nil && item.Path != "" {
					relPath, err := filepath.Rel(m.WorkingDir, item.Path)
					if err != nil || relPath == "" {
						relPath = item.Path
					}
					var rmCmd string
					if item.IsDir {
						rmCmd = fmt.Sprintf("rm -rf %s", quotePath(relPath))
					} else {
						rmCmd = fmt.Sprintf("rm -f %s", quotePath(relPath))
					}
					m.Modal.OpenShellCommand(rmCmd)
					m.updateFocus()
				}
				return m, nil
			case "r":
				m.FileTree.Refresh()
				m.StatusMessage = "File list refreshed"
				return m, nil
			}

		case PaneEditor:
			if m.Editor.Mode == editor.ModeView {
				if msg.String() == "z" || msg.String() == "Z" {
					m.ToggleEditorFullscreen()
					return m, nil
				}
				if msg.String() == "e" || msg.String() == "E" {
					m.Editor.ToggleMode()
					return m, nil
				}
			}
			var edCmd tea.Cmd
			m.Editor, edCmd = m.Editor.Update(msg)
			cmds = append(cmds, edCmd)

		case PaneConsole:
			switch msg.String() {
			case "m", "M":
				m.ConsoleMaximized = !m.ConsoleMaximized
				m.recalculateLayout()
				if m.ConsoleMaximized {
					m.StatusMessage = "Console maximized (3/4 height) • Press 'm' to restore"
				} else {
					m.StatusMessage = "Console restored to default height"
				}
				return m, nil

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

func (m *Model) triggerGeminiAI(userQuery string, apiKey string, mode ai.AIMode) tea.Cmd {
	if m.Editor.Mode == editor.ModeEdit {
		m.Editor.ToggleMode()
	}
	m.ActivePane = PaneConsole
	m.updateFocus()
	m.Console.IsRunning = true
	m.activeAIMode = mode
	m.aiResponse = ""

	var fileList []string
	for _, item := range m.FileTree.VisibleItems {
		if !item.IsDir {
			fileList = append(fileList, item.Name)
		}
	}

	fullPrompt := ai.BuildPrompt(
		mode,
		userQuery,
		m.Editor.Buffer.FilePath,
		m.Editor.Buffer.CurrentText,
		m.Console.LastCommand,
		m.Console.LastOutput,
		fileList,
	)

	var headerTitle string
	switch mode {
	case ai.ModeUpdateFile:
		m.Console.RunningTitle = fmt.Sprintf("Gemini AI: Updating %s...", m.Editor.Buffer.FileName())
		m.StatusMessage = fmt.Sprintf("Asking Gemini to update %s...", m.Editor.Buffer.FileName())
		headerTitle = fmt.Sprintf("✦ **Gemini AI (Updating %s):**\n\n", m.Editor.Buffer.FileName())
	case ai.ModeGenerateFile:
		m.Console.RunningTitle = "Gemini AI: Generating new file..."
		m.StatusMessage = "Asking Gemini to generate new file..."
		headerTitle = "✦ **Gemini AI (Generating New File):**\n\n"
	default:
		m.Console.RunningTitle = "Gemini AI stream..."
		m.StatusMessage = "Asking Gemini..."
		headerTitle = "✦ **Gemini AI Assistant:**\n\n"
	}

	m.Console.AddAIChunk(headerTitle, true)

	m.AIChannel = make(chan ai.AIChunkMsg, 32)
	go ai.AskGeminiStream(
		context.Background(),
		apiKey,
		m.Config.GeminiModel,
		mode,
		fullPrompt,
		m.AIChannel,
	)

	return ai.ListenForAIChunk(m.AIChannel)
}

func (m *Model) refreshWorkspace() {
	// 1. Refresh file explorer sidebar
	m.FileTree.Refresh()

	// 2. Reload active editor buffer if changed on disk
	if m.Editor.Buffer.FilePath != "" {
		reloaded := m.Editor.Reload()
		if reloaded {
			m.Editor.SetDiagnostics(m.Diagnostics)
		}
	} else if len(m.FileTree.VisibleItems) > 0 && !m.Editor.Buffer.IsLoaded {
		// If no file was open or previous file was deleted, auto-open first valid text file
		for _, item := range m.FileTree.VisibleItems {
			if !item.IsDir {
				isText, _ := editor.IsTextFile(item.Path)
				if isText {
					_ = m.Editor.OpenFile(item.Path)
					m.FileTree.SelectFile(item.Path)
					break
				}
			}
		}
	}
}

func (m *Model) recalculateLayout() {
	if m.Width == 0 || m.Height == 0 {
		return
	}

	headerH := 1
	footerH := 1
	bodyTotalH := max(4, m.Height-headerH-footerH)

	if m.EditorFullscreen {
		editorInnerW := max(10, m.Width-2)
		editorInnerH := max(1, bodyTotalH-2)
		m.Editor.SetSize(editorInnerW, editorInnerH)
		m.Modal.SetSize(m.Width, m.Height)
		return
	}

	var consoleH int
	if m.ConsoleMaximized {
		consoleH = max(6, int(float64(bodyTotalH)*0.75))
		if bodyTotalH-consoleH < 4 {
			consoleH = max(4, bodyTotalH-4)
		}
	} else {
		consoleH = max(6, int(float64(bodyTotalH)*0.28))
	}
	mainH := max(4, bodyTotalH-consoleH)

	// Fixed width for file explorer (24-30 chars)
	sidebarW := min(30, max(20, int(float64(m.Width)*0.22)))
	editorW := max(20, m.Width-sidebarW)

	sidebarInnerW := max(10, sidebarW-2)
	sidebarInnerH := max(1, mainH-2)

	editorInnerW := max(10, editorW-2)
	editorInnerH := max(1, mainH-2)

	consoleInnerW := max(10, m.Width-2)
	consoleInnerH := max(1, consoleH-2)

	// Now that titles are on top borders, components get the full inner height!
	m.FileTree.SetSize(sidebarInnerW, sidebarInnerH)
	m.Editor.SetSize(editorInnerW, editorInnerH)
	m.Console.SetSize(consoleInnerW, consoleInnerH)
	m.Modal.SetSize(m.Width, m.Height)
}

// ToggleEditorFullscreen toggles full-screen editor mode, hiding sidebar and console.
func (m *Model) ToggleEditorFullscreen() {
	m.EditorFullscreen = !m.EditorFullscreen
	m.recalculateLayout()
	if m.EditorFullscreen {
		m.ActivePane = PaneEditor
		m.updateFocus()
		m.StatusMessage = "Editor fullscreen enabled (Press ^Z or 'z' to restore)"
	} else {
		m.StatusMessage = "Restored standard multi-pane layout"
	}
}

func quotePath(path string) string {
	if strings.ContainsAny(path, " \t'\"$()[]{}*?~`!&;<>|^#\\") {
		return fmt.Sprintf("%q", path)
	}
	return path
}

// RenderTitledBox renders a bordered box with the title embedded directly on the top border line.
func RenderTitledBox(borderStyle lipgloss.Style, title string, titleStyle lipgloss.Style, rightHint string, content string, innerWidth int, innerHeight int) string {
	box := borderStyle.
		Width(innerWidth).
		Height(innerHeight).
		Render(content)

	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}

	totalOuterWidth := innerWidth + 2
	borderColor := borderStyle.GetBorderTopForeground()
	borderSt := lipgloss.NewStyle().Foreground(borderColor)

	topLeft := borderSt.Render("╭")
	topRight := borderSt.Render("╮")
	topDash := borderSt.Render("─")

	if title == "" && rightHint == "" {
		lines[0] = topLeft + borderSt.Render(strings.Repeat("─", innerWidth)) + topRight
		return strings.Join(lines, "\n")
	}

	formattedTitle := ""
	titleWidth := 0
	if title != "" {
		formattedTitle = " " + title + " "
		titleWidth = ansi.StringWidth(formattedTitle)
	}

	formattedHint := ""
	hintWidth := 0
	if rightHint != "" {
		formattedHint = " " + rightHint + " "
		hintWidth = ansi.StringWidth(formattedHint)
	}

	// Truncate title if total exceeds width
	maxTitleWidth := max(4, totalOuterWidth-6-hintWidth)
	if titleWidth > maxTitleWidth && maxTitleWidth > 0 {
		formattedTitle = " " + ansi.Truncate(title, maxTitleWidth-2, "...") + " "
		titleWidth = ansi.StringWidth(formattedTitle)
	}

	renderedTitle := ""
	if formattedTitle != "" {
		renderedTitle = titleStyle.Render(formattedTitle)
	}

	renderedHint := ""
	if formattedHint != "" {
		renderedHint = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render(formattedHint)
	}

	remWidth := totalOuterWidth - 2 - 1 - titleWidth - hintWidth
	if remWidth < 0 {
		remWidth = 0
	}

	topLine := topLeft + topDash + renderedTitle + borderSt.Render(strings.Repeat("─", remWidth)) + renderedHint + topRight
	if ansi.StringWidth(topLine) > totalOuterWidth {
		topLine = ansi.Truncate(topLine, totalOuterWidth, "")
	}

	lines[0] = topLine
	return strings.Join(lines, "\n")
}

// View renders the complete TIDE interface.
func (m Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing TIDE..."
	}

	headerH := 1
	footerH := 1
	bodyTotalH := max(4, m.Height-headerH-footerH)

	var consoleH int
	if m.ConsoleMaximized {
		consoleH = max(6, int(float64(bodyTotalH)*0.75))
		if bodyTotalH-consoleH < 4 {
			consoleH = max(4, bodyTotalH-4)
		}
	} else {
		consoleH = max(6, int(float64(bodyTotalH)*0.28))
	}
	mainH := max(4, bodyTotalH-consoleH)

	sidebarW := min(30, max(20, int(float64(m.Width)*0.22)))
	editorW := max(20, m.Width-sidebarW)

	sidebarInnerW := max(10, sidebarW-2)
	sidebarInnerH := max(1, mainH-2)
	editorInnerW := max(10, editorW-2)
	editorInnerH := max(1, mainH-2)
	consoleInnerW := max(10, m.Width-2)
	consoleInnerH := max(1, consoleH-2)

	// 1. Header Bar (exactly 1 line)
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
	if ansi.StringWidth(headerLeft) > m.Width {
		headerLeft = ansi.Truncate(headerLeft, m.Width, "")
	}
	headerBar := lipgloss.NewStyle().
		Width(m.Width).
		Background(ColorBg).
		Render(headerLeft)

	// Fullscreen Editor Mode (hides sidebar and console)
	if m.EditorFullscreen {
		editorInnerW := max(10, m.Width-2)
		editorInnerH := max(1, bodyTotalH-2)

		editorBorder := ActiveBorderStyle
		editorTitleText := fmt.Sprintf("%s [%s] (Fullscreen)", fileName, m.Editor.Buffer.Language)
		editorTitleStyle := PanelTitleActive
		if m.Editor.Buffer.IsModified {
			editorTitleText += " [MOD]"
		}
		editorHint := "^Z Restore  w Wrap  ^E Edit"
		editorBox := RenderTitledBox(
			editorBorder,
			editorTitleText,
			editorTitleStyle,
			editorHint,
			m.Editor.View(),
			editorInnerW,
			editorInnerH,
		)

		keys := []struct {
			key  string
			desc string
		}{
			{"^Z", "Restore"},
			{"^E", "Edit/View"},
			{"^S", "Save"},
			{"w", "Wrap"},
			{"^G", "Gemini"},
			{"^F", "Files"},
			{"^Q", "Quit"},
		}

		var keyItems []string
		for _, k := range keys {
			renderedKey := FooterKeyStyle.Render(" " + k.key + " ")
			renderedDesc := FooterDescStyle.Render(k.desc)
			keyItems = append(keyItems, renderedKey+renderedDesc)
		}

		footerContent := strings.Join(keyItems, " ")
		if ansi.StringWidth(footerContent) > m.Width {
			footerContent = ansi.Truncate(footerContent, m.Width, "")
		}
		footerBar := lipgloss.NewStyle().
			Width(m.Width).
			Background(ColorSurface).
			Render(footerContent)

		fullScreen := lipgloss.JoinVertical(
			lipgloss.Left,
			headerBar,
			editorBox,
			footerBar,
		)

		renderedLines := strings.Split(fullScreen, "\n")
		for len(renderedLines) < m.Height {
			renderedLines = append(renderedLines, "")
		}
		if len(renderedLines) > m.Height {
			renderedLines = renderedLines[:m.Height]
		}
		fullScreen = strings.Join(renderedLines, "\n")

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

	// 2. File Sidebar Box with Border-Embedded Title
	sidebarBorder := InactiveBorderStyle
	sidebarTitleText := "FILES"
	sidebarTitleStyle := PanelTitleInactive
	sidebarHint := ". Hidden  r Refresh"
	if m.FileTree.ShowHidden {
		sidebarHint = ". All Files  r Refresh"
	}
	if m.ActivePane == PaneFiles {
		sidebarBorder = ActiveBorderStyle
		if m.FileTree.ShowHidden {
			sidebarTitleText = "FILES [All] (Active)"
		} else {
			sidebarTitleText = "FILES [Active]"
		}
		sidebarTitleStyle = PanelTitleActive
	} else if m.FileTree.ShowHidden {
		sidebarTitleText = "FILES [All]"
	}
	sidebarBox := RenderTitledBox(
		sidebarBorder,
		sidebarTitleText,
		sidebarTitleStyle,
		sidebarHint,
		m.FileTree.View(),
		sidebarInnerW,
		sidebarInnerH,
	)

	// 3. Editor Box with Border-Embedded Title
	editorBorder := InactiveBorderStyle
	editorTitleText := fmt.Sprintf("%s [%s]", fileName, m.Editor.Buffer.Language)
	editorTitleStyle := PanelTitleInactive
	if m.ActivePane == PaneEditor {
		editorBorder = ActiveBorderStyle
		editorTitleText = fmt.Sprintf("%s [%s] (Active)", fileName, m.Editor.Buffer.Language)
		editorTitleStyle = PanelTitleActive
	}
	if m.Editor.Buffer.IsModified {
		editorTitleText += " [MOD]"
	}
	editorBox := RenderTitledBox(
		editorBorder,
		editorTitleText,
		editorTitleStyle,
		"",
		m.Editor.View(),
		editorInnerW,
		editorInnerH,
	)

	mainSplit := lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, editorBox)

	// 4. Console Box with Border-Embedded Title & Shortcuts Hint
	consoleBorder := InactiveBorderStyle
	consoleTitleText := "OUTPUT / CONSOLE"
	consoleTitleStyle := PanelTitleInactive
	consoleHint := "m Maximize  ^X Shell  ^G Gemini  c Clear"
	if m.ConsoleMaximized {
		consoleHint = "m Restore  ^X Shell  ^G Gemini  c Clear"
	}
	if m.Console.IsRunning {
		consoleTitleText = fmt.Sprintf("CONSOLE [⟳ %s]", m.Console.RunningTitle)
		consoleTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F9E2AF"))
	} else if m.ActivePane == PaneConsole {
		consoleBorder = ActiveBorderStyle
		if m.ConsoleMaximized {
			consoleTitleText = "OUTPUT / CONSOLE [Maximized] (Active)"
		} else {
			consoleTitleText = "OUTPUT / CONSOLE (Active)"
		}
		consoleTitleStyle = PanelTitleActive
	}
	consoleBox := RenderTitledBox(
		consoleBorder,
		consoleTitleText,
		consoleTitleStyle,
		consoleHint,
		m.Console.View(),
		consoleInnerW,
		consoleInnerH,
	)

	// 5. Pico Footer Bar (exactly 1 line)
	keys := []struct {
		key  string
		desc string
	}{
		{"^F", "Files"},
		{"^N", "New"},
		{"^E", "Edit"},
		{"^S", "Save"},
		{"^B", "Build"},
		{"^R", "Run"},
		{"^Z", "Full"},
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
	if ansi.StringWidth(footerContent) > m.Width {
		footerContent = ansi.Truncate(footerContent, m.Width, "")
	}
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

	// Ensure the fullScreen output matches m.Height exactly
	renderedLines := strings.Split(fullScreen, "\n")
	for len(renderedLines) < m.Height {
		renderedLines = append(renderedLines, "")
	}
	if len(renderedLines) > m.Height {
		renderedLines = renderedLines[:m.Height]
	}
	fullScreen = strings.Join(renderedLines, "\n")

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
