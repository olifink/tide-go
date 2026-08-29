package editor

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"tide/internal/runner"
)

// HighlightCode renders source code using Chroma with line numbers and error gutters.
func HighlightCode(filename string, content string, themeName string, diagnostics map[int][]runner.Diagnostic, startLine int, maxLines int, width int) string {
	if content == "" {
		return ""
	}

	// 1. Select Lexer
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Analyse(content)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	// 2. Select Style
	style := styles.Get(themeName)
	if style == nil {
		style = styles.Fallback
	}

	// 3. Tokenize content into lines
	lines := strings.Split(content, "\n")
	totalLines := len(lines)
	if totalLines == 0 {
		return ""
	}

	// Determine gutter width based on total lines (e.g., " 123 | ")
	gutterDigits := len(fmt.Sprintf("%d", totalLines))
	if gutterDigits < 2 {
		gutterDigits = 2
	}

	// Setup Chroma Terminal formatter for single-line chunks
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	endLine := min(totalLines, startLine+maxLines)
	var renderedLines []string

	normalGutterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6272A4")).
		Background(lipgloss.Color("#1E1E2E"))

	errorGutterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#FF5555")).
		Bold(true)

	warningGutterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#FFB86C")).
		Bold(true)

	errorMsgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5555")).
		Italic(true)

	for i := startLine; i < endLine; i++ {
		lineNum := i + 1
		lineContent := lines[i]

		// Check for compiler diagnostics on this line
		diags := diagnostics[lineNum]
		hasError := false
		hasWarning := false
		var diagMsg string

		for _, d := range diags {
			if d.Severity == "error" {
				hasError = true
				diagMsg = d.Message
				break
			} else if d.Severity == "warning" {
				hasWarning = true
				if diagMsg == "" {
					diagMsg = d.Message
				}
			}
		}

		// Highlight line content with Chroma
		var highlightedContent string
		if strings.TrimSpace(lineContent) == "" {
			highlightedContent = ""
		} else {
			iterator, err := lexer.Tokenise(nil, lineContent)
			if err == nil {
				var buf bytes.Buffer
				if err := formatter.Format(&buf, style, iterator); err == nil {
					highlightedContent = buf.String()
				} else {
					highlightedContent = lineContent
				}
			} else {
				highlightedContent = lineContent
			}
		}

		// Format gutter
		var gutter string
		if hasError {
			gutterText := fmt.Sprintf("! %*d │ ", gutterDigits, lineNum)
			gutter = errorGutterStyle.Render(gutterText)
		} else if hasWarning {
			gutterText := fmt.Sprintf("? %*d │ ", gutterDigits, lineNum)
			gutter = warningGutterStyle.Render(gutterText)
		} else {
			gutterText := fmt.Sprintf("  %*d │ ", gutterDigits, lineNum)
			gutter = normalGutterStyle.Render(gutterText)
		}

		fullLine := gutter + highlightedContent

		// If line has diagnostic message and fits width, append inline indicator
		if diagMsg != "" && width > 40 {
			msgPreview := "  ◄ " + diagMsg
			availWidth := width - lipgloss.Width(fullLine) - 2
			if availWidth > 8 {
				if len(msgPreview) > availWidth {
					msgPreview = msgPreview[:availWidth-3] + "..."
				}
				fullLine += errorMsgStyle.Render(msgPreview)
			}
		}

		renderedLines = append(renderedLines, fullLine)
	}

	return strings.Join(renderedLines, "\n")
}

// DetectLanguage returns a friendly language name from file extension.
func DetectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "Go"
	case ".c", ".h":
		return "C"
	case ".cpp", ".hpp", ".cc", ".cxx":
		return "C++"
	case ".rs":
		return "Rust"
	case ".py":
		return "Python"
	case ".js":
		return "JavaScript"
	case ".ts":
		return "TypeScript"
	case ".json":
		return "JSON"
	case ".md":
		return "Markdown"
	case ".yaml", ".yml":
		return "YAML"
	case ".sh", ".bash":
		return "Shell"
	case ".mod", ".sum":
		return "Go Module"
	case ".txt":
		return "Plain Text"
	default:
		if strings.EqualFold(filepath.Base(filePath), "Makefile") {
			return "Makefile"
		}
		if ext != "" {
			return strings.ToUpper(strings.TrimPrefix(ext, "."))
		}
		return "Text"
	}
}
