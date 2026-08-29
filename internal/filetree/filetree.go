package filetree

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// FileItem represents a file or directory entry in the tree.
type FileItem struct {
	Name         string
	Path         string
	IsDir        bool
	IsExecutable bool
	Depth        int
	Expanded     bool
	Children     []*FileItem
}

// Model represents the file tree component.
type Model struct {
	RootPath     string
	Root         *FileItem
	VisibleItems []*FileItem
	Cursor       int
	ActiveFile   string // File currently loaded in editor
	ShowHidden   bool   // When true, dot files & folders (.git, .github, .gitignore) are visible
	Width        int
	Height       int
	Focused      bool
}

// New creates and initializes a file tree model for the given root directory.
func New(rootPath string, width, height int) Model {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		absPath = rootPath
	}

	m := Model{
		RootPath:   absPath,
		Width:      width,
		Height:     height,
		Cursor:     0,
		ShowHidden: false,
	}
	m.Refresh()
	return m
}

// Refresh re-scans the directory tree.
func (m *Model) Refresh() {
	var expandedPaths = make(map[string]bool)
	for _, item := range m.VisibleItems {
		if item.IsDir && item.Expanded {
			expandedPaths[item.Path] = true
		}
	}

	rootItem := &FileItem{
		Name:     filepath.Base(m.RootPath),
		Path:     m.RootPath,
		IsDir:    true,
		Depth:    0,
		Expanded: true,
	}

	m.buildTree(rootItem, expandedPaths, 0)
	m.Root = rootItem
	m.rebuildVisible()

	if m.Cursor >= len(m.VisibleItems) {
		m.Cursor = max(0, len(m.VisibleItems)-1)
	}
}

// ToggleHidden toggles display of hidden dot files and directories.
func (m *Model) ToggleHidden() {
	m.ShowHidden = !m.ShowHidden
	m.Refresh()
}

func (m *Model) buildTree(item *FileItem, expandedPaths map[string]bool, depth int) {
	if !item.IsDir {
		return
	}

	entries, err := os.ReadDir(item.Path)
	if err != nil {
		return
	}

	var dirs []*FileItem
	var files []*FileItem

	for _, entry := range entries {
		name := entry.Name()
		// Always ignore . and ..
		if name == "." || name == ".." {
			continue
		}

		// Filter hidden files/directories unless ShowHidden is enabled
		if !m.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}

		subPath := filepath.Join(item.Path, name)
		isDir := entry.IsDir()
		isExec := false

		if !isDir {
			if info, err := entry.Info(); err == nil {
				// Check for executable bit in Unix permissions
				if info.Mode().Perm()&0111 != 0 {
					isExec = true
				}
			}
		}

		child := &FileItem{
			Name:         name,
			Path:         subPath,
			IsDir:        isDir,
			IsExecutable: isExec,
			Depth:        depth + 1,
			Expanded:     expandedPaths[subPath],
		}

		if isDir {
			if child.Expanded {
				m.buildTree(child, expandedPaths, depth+1)
			}
			dirs = append(dirs, child)
		} else {
			files = append(files, child)
		}
	}

	// Sort directories and files alphabetically
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	item.Children = append(dirs, files...)
}

func (m *Model) rebuildVisible() {
	m.VisibleItems = nil
	if m.Root == nil {
		return
	}
	for _, child := range m.Root.Children {
		m.addVisible(child)
	}
}

func (m *Model) addVisible(item *FileItem) {
	m.VisibleItems = append(m.VisibleItems, item)
	if item.IsDir && item.Expanded {
		for _, child := range item.Children {
			m.addVisible(child)
		}
	}
}

// MoveUp moves cursor up.
func (m *Model) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

// MoveDown moves cursor down.
func (m *Model) MoveDown() {
	if m.Cursor < len(m.VisibleItems)-1 {
		m.Cursor++
	}
}

// Home moves cursor to top.
func (m *Model) Home() {
	m.Cursor = 0
}

// End moves cursor to bottom.
func (m *Model) End() {
	if len(m.VisibleItems) > 0 {
		m.Cursor = len(m.VisibleItems) - 1
	}
}

// PageUp moves cursor up by half height.
func (m *Model) PageUp() {
	step := max(1, m.Height/2)
	m.Cursor = max(0, m.Cursor-step)
}

// PageDown moves cursor down by half height.
func (m *Model) PageDown() {
	step := max(1, m.Height/2)
	m.Cursor = min(len(m.VisibleItems)-1, m.Cursor+step)
}

// SelectedItem returns the currently highlighted FileItem.
func (m *Model) SelectedItem() *FileItem {
	if m.Cursor >= 0 && m.Cursor < len(m.VisibleItems) {
		return m.VisibleItems[m.Cursor]
	}
	return nil
}

// ToggleCurrent toggles expansion if directory, or returns file path, isFile, and isExecutable.
func (m *Model) ToggleCurrent() (selectedFile string, isFile bool, isExec bool) {
	item := m.SelectedItem()
	if item == nil {
		return "", false, false
	}
	if item.IsDir {
		item.Expanded = !item.Expanded
		if item.Expanded && len(item.Children) == 0 {
			var dummyMap = make(map[string]bool)
			dummyMap[item.Path] = true
			m.buildTree(item, dummyMap, item.Depth)
		}
		m.rebuildVisible()
		return "", false, false
	}
	return item.Path, true, item.IsExecutable
}

// SelectFile searches for the file path in the tree and moves the cursor to it.
func (m *Model) SelectFile(filePath string) {
	m.ActiveFile = filePath
	for idx, item := range m.VisibleItems {
		if item.Path == filePath {
			m.Cursor = idx
			return
		}
	}
}

// SetSize updates the dimensions of the file tree.
func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
}

// View renders the file tree with * prefix for executable binaries.
func (m *Model) View() string {
	targetHeight := max(1, m.Height)
	targetWidth := max(10, m.Width)

	if len(m.VisibleItems) == 0 {
		emptyLines := []string{
			"No files found",
			"",
			"Press ^N to create",
			"Press . for hidden",
		}
		for len(emptyLines) < targetHeight {
			emptyLines = append(emptyLines, "")
		}
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#767676"))
		for i, line := range emptyLines {
			emptyLines[i] = emptyStyle.Render(ansi.Truncate(line, targetWidth, ""))
		}
		return strings.Join(emptyLines[:targetHeight], "\n")
	}

	// Calculate scrolling window
	startIdx := 0
	if m.Cursor >= targetHeight {
		startIdx = m.Cursor - targetHeight + 1
	}
	endIdx := min(len(m.VisibleItems), startIdx+targetHeight)

	var lines []string
	for i := startIdx; i < endIdx; i++ {
		item := m.VisibleItems[i]
		isCursor := i == m.Cursor && m.Focused
		isActive := item.Path == m.ActiveFile
		isDot := strings.HasPrefix(item.Name, ".")

		// Indentation
		indent := strings.Repeat("  ", max(0, item.Depth-1))

		// Icon / Prefix
		var icon string
		if item.IsDir {
			if item.Expanded {
				icon = "▾ "
			} else {
				icon = "▸ "
			}
		} else if item.IsExecutable {
			icon = "* "
		} else {
			icon = "  "
		}

		// Cursor marker
		cursorMarker := " "
		if isCursor {
			cursorMarker = ">"
		}

		// Format line text
		lineStr := cursorMarker + " " + indent + icon + item.Name

		// Apply styling
		var style lipgloss.Style
		if isCursor {
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#5A5288")).
				Bold(true)
		} else if isActive {
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#74C7EC")).
				Bold(true)
		} else if item.IsDir {
			if isDot {
				style = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#6272A4")).
					Bold(true)
			} else {
				style = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#8BE9FD")).
					Bold(true)
			}
		} else if item.IsExecutable {
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#50FA7B")).
				Bold(true)
		} else if isDot {
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6272A4"))
		} else {
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8F8F2"))
		}

		// Render with style and truncate to targetWidth
		renderedLine := style.Render(lineStr)
		if targetWidth > 0 && ansi.StringWidth(renderedLine) > targetWidth {
			renderedLine = ansi.Truncate(renderedLine, targetWidth, "")
		}
		lines = append(lines, renderedLine)
	}

	for len(lines) < targetHeight {
		lines = append(lines, "")
	}
	if len(lines) > targetHeight {
		lines = lines[:targetHeight]
	}

	return strings.Join(lines, "\n")
}
