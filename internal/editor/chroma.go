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
	"github.com/charmbracelet/x/ansi"
	"tide/internal/runner"
)

// HighlightCode renders source code using Chroma with line numbers, error gutters,
// optional word-wrapping, and horizontal scrolling support.
func HighlightCode(filename string, content string, themeName string, diagnostics map[int][]runner.Diagnostic, startLine int, maxLines int, width int, wordWrap bool, scrollCol int) string {
	if maxLines <= 0 {
		return ""
	}

	if content == "" {
		var blank []string
		for i := 0; i < maxLines; i++ {
			blank = append(blank, "")
		}
		return strings.Join(blank, "\n")
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

	// 3. Normalize line endings (\r\n -> \n, \r -> \n) and tabs
	normalizedContent := strings.ReplaceAll(content, "\r\n", "\n")
	normalizedContent = strings.ReplaceAll(normalizedContent, "\r", "\n")
	normalizedContent = strings.ReplaceAll(normalizedContent, "\t", "    ")
	lines := strings.Split(normalizedContent, "\n")
	totalLines := len(lines)

	// Determine gutter width based on total lines (e.g., " 123 | ")
	gutterDigits := len(fmt.Sprintf("%d", totalLines))
	if gutterDigits < 2 {
		gutterDigits = 2
	}
	gutterWidth := gutterDigits + 5 // e.g. "  12 │ " is 7 chars
	availCodeWidth := max(10, width-gutterWidth)

	// Setup Chroma Terminal formatter for single-line chunks
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

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

	for i := startLine; i < totalLines; i++ {
		if len(renderedLines) >= maxLines {
			break
		}
		lineNum := i + 1
		lineContent := strings.TrimRight(lines[i], "\r\n")

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

		// Strip any trailing carriage returns or newlines from formatter
		highlightedContent = strings.ReplaceAll(highlightedContent, "\r", "")
		highlightedContent = strings.ReplaceAll(highlightedContent, "\n", "")

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

		if wordWrap {
			// WORD WRAP: split content across lines
			lineWidth := ansi.StringWidth(highlightedContent)
			if lineWidth <= availCodeWidth || width <= 0 {
				fullLine := gutter + highlightedContent
				// Diagnostic message indicator if it fits
				if diagMsg != "" && width > 40 {
					msgPreview := "  ◄ " + diagMsg
					remWidth := width - ansi.StringWidth(fullLine) - 2
					if remWidth > 8 {
						if len(msgPreview) > remWidth {
							msgPreview = msgPreview[:remWidth-3] + "..."
						}
						fullLine += errorMsgStyle.Render(msgPreview)
					}
				}
				if width > 0 && ansi.StringWidth(fullLine) > width {
					fullLine = ansi.Truncate(fullLine, width, "")
				}
				renderedLines = append(renderedLines, fullLine)
			} else {
				wrapped := ansi.Hardwrap(highlightedContent, availCodeWidth, true)
				subLines := strings.Split(wrapped, "\n")
				for sIdx, sub := range subLines {
					if len(renderedLines) >= maxLines {
						break
					}
					var rowLine string
					if sIdx == 0 {
						rowLine = gutter + sub
					} else {
						contGutterText := fmt.Sprintf("%*s ↳ ", gutterDigits+2, "")
						contGutter := normalGutterStyle.Render(contGutterText)
						rowLine = contGutter + sub
					}
					if width > 0 && ansi.StringWidth(rowLine) > width {
						rowLine = ansi.Truncate(rowLine, width, "")
					}
					renderedLines = append(renderedLines, rowLine)
				}
			}
		} else {
			// NO WRAP: horizontal scrolling via scrollCol
			contentToRender := highlightedContent
			if scrollCol > 0 && ansi.StringWidth(contentToRender) > 0 {
				contentToRender = ansi.TruncateLeft(contentToRender, scrollCol, "")
			}

			fullLine := gutter + contentToRender

			// If line has diagnostic message and fits width, append inline indicator
			if diagMsg != "" && width > 40 {
				msgPreview := "  ◄ " + diagMsg
				availWidth := width - ansi.StringWidth(fullLine) - 2
				if availWidth > 8 {
					if len(msgPreview) > availWidth {
						msgPreview = msgPreview[:availWidth-3] + "..."
					}
					fullLine += errorMsgStyle.Render(msgPreview)
				}
			}

			// Strictly truncate full line to width to prevent horizontal wrapping
			if width > 0 && ansi.StringWidth(fullLine) > width {
				fullLine = ansi.Truncate(fullLine, width, "")
			}

			renderedLines = append(renderedLines, fullLine)
		}
	}

	// Pad remaining lines to always match maxLines exactly
	for len(renderedLines) < maxLines {
		renderedLines = append(renderedLines, "")
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
