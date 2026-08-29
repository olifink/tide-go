package ai

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/genai"
)

// AIMode defines the purpose of the Gemini prompt.
type AIMode int

const (
	ModeConsoleQA    AIMode = iota // General Q&A and explanations streamed to Console
	ModeGenerateFile               // Generate and create a new file on disk
	ModeUpdateFile                 // Modify the currently open editor buffer
)

// AIChunkMsg is sent to the Bubbletea event loop as streaming chunks arrive from Gemini.
type AIChunkMsg struct {
	Chunk string
	Done  bool
	Err   error
	Mode  AIMode
}

var (
	filenameRegex  = regexp.MustCompile(`(?i)(?:FILENAME|FILE):\s*([a-zA-Z0-9_.\-\\/]+)`)
	codeBlockRegex = regexp.MustCompile("(?s)```(?:[a-zA-Z0-9_\\-\\+]+)?\\r?\\n(.*?)\\r?\\n?```")
)

// detectLanguageAndTag maps file extensions to human-readable language names and markdown code block tags.
func detectLanguageAndTag(filename string) (string, string) {
	if filename == "" {
		return "Plain Text", "text"
	}
	base := strings.ToLower(filepath.Base(filename))
	ext := strings.ToLower(filepath.Ext(filename))

	switch {
	case ext == ".go":
		return "Go", "go"
	case ext == ".c" || ext == ".h":
		return "C", "c"
	case ext == ".cpp" || ext == ".cc" || ext == ".cxx" || ext == ".hpp":
		return "C++", "cpp"
	case ext == ".rs":
		return "Rust", "rust"
	case ext == ".py":
		return "Python", "python"
	case ext == ".js":
		return "JavaScript", "javascript"
	case ext == ".ts":
		return "TypeScript", "typescript"
	case ext == ".sh" || ext == ".bash":
		return "Shell", "bash"
	case ext == ".json":
		return "JSON", "json"
	case ext == ".yaml" || ext == ".yml":
		return "YAML", "yaml"
	case ext == ".md":
		return "Markdown", "markdown"
	case base == "makefile":
		return "Makefile", "makefile"
	default:
		tag := strings.TrimPrefix(ext, ".")
		if tag == "" {
			tag = "text"
		}
		return strings.ToUpper(tag), tag
	}
}

// BuildPrompt constructs a context-aware prompt based on the active mode with explicit language enforcement.
func BuildPrompt(mode AIMode, userQuery string, activeFile string, codeContent string, lastCommand string, lastOutput string, fileList []string) string {
	var sb strings.Builder
	langName, langTag := detectLanguageAndTag(activeFile)

	switch mode {
	case ModeUpdateFile:
		sb.WriteString("You are an expert software developer embedded in the TIDE terminal IDE.\n")
		sb.WriteString(fmt.Sprintf("Your task is to modify the active %s source file (%s) based on the user's instructions.\n\n", langName, filepath.Base(activeFile)))

		sb.WriteString(fmt.Sprintf("### TARGET FILE SPECIFICATION:\n- Filename: %s\n- Language: %s\n- Markdown Tag: ```%s\n\n", filepath.Base(activeFile), langName, langTag))

		if activeFile != "" && codeContent != "" {
			sb.WriteString(fmt.Sprintf("### CURRENT %s FILE CONTENT (%s):\n```%s\n%s\n```\n\n", strings.ToUpper(langName), filepath.Base(activeFile), langTag, codeContent))
		}

		if lastOutput != "" {
			cmdHeader := "Last Compiler / Terminal Output:"
			if lastCommand != "" {
				cmdHeader = fmt.Sprintf("Command ($ %s) Output:", lastCommand)
			}
			sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", cmdHeader, lastOutput))
		}

		sb.WriteString("### User Edit Request:\n")
		sb.WriteString(userQuery)
		sb.WriteString("\n\n### CRITICAL INSTRUCTIONS:\n")
		sb.WriteString(fmt.Sprintf("1. LANGUAGE ENFORCEMENT: The file is %s (%s). All code MUST be written strictly in valid %s syntax. Do NOT switch to Python or any other programming language.\n", filepath.Base(activeFile), langName, langName))
		sb.WriteString(fmt.Sprintf("2. Return the COMPLETE, UPDATED %s file content in a single ```%s markdown code block.\n", langName, langTag))
		sb.WriteString("3. Do NOT use ellipses (...), truncated functions, or placeholders. Output the full file ready to be saved.\n")
		sb.WriteString("4. Provide a brief explanation of the changes before or after the code block.\n")

	case ModeGenerateFile:
		sb.WriteString("You are an expert software developer embedded in the TIDE terminal IDE.\n")
		sb.WriteString("Your task is to generate a new source file based on the user's request.\n\n")

		if len(fileList) > 0 {
			sb.WriteString(fmt.Sprintf("### Existing Project Files:\n%s\n\n", strings.Join(fileList, ", ")))
		}

		sb.WriteString("### User Request:\n")
		sb.WriteString(userQuery)
		sb.WriteString("\n\n### CRITICAL INSTRUCTIONS:\n")
		sb.WriteString("1. On the first line, specify the recommended filename in this exact format: `FILENAME: filename.ext`\n")
		sb.WriteString("2. Match the language and file extension to the request and project ecosystem (e.g. use .go for Go projects, .c/.h for C projects).\n")
		sb.WriteString("3. Output the COMPLETE new file code inside a single markdown code block with the appropriate language tag.\n")
		sb.WriteString("4. Ensure the code is complete, valid, and fully implemented without placeholders or pseudocode.\n")

	default: // ModeConsoleQA
		sb.WriteString("You are an expert software developer and debugging assistant inside the TIDE terminal IDE.\n\n")

		if activeFile != "" && codeContent != "" {
			sb.WriteString(fmt.Sprintf("### Active File: %s (Language: %s)\n```%s\n%s\n```\n\n", filepath.Base(activeFile), langName, langTag, codeContent))
		}

		if lastOutput != "" {
			cmdHeader := "Last Compiler / Terminal Output:"
			if lastCommand != "" {
				cmdHeader = fmt.Sprintf("Command ($ %s) Output:", lastCommand)
			}
			sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", cmdHeader, lastOutput))
		}

		sb.WriteString("### User Request:\n")
		sb.WriteString(userQuery)
		sb.WriteString(fmt.Sprintf("\n\nPlease provide a clear, concise, and helpful response. When providing code fixes or snippets, match the language of the active file (%s).", langName))
	}

	return sb.String()
}

// ExtractCodeAndFile extracts the code block, suggested filename, and explanation from Gemini's full response.
func ExtractCodeAndFile(response string, defaultFilename string) (filename string, code string, explanation string) {
	filename = defaultFilename

	// Check if response suggests a filename
	if match := filenameRegex.FindStringSubmatch(response); match != nil {
		fn := strings.TrimSpace(match[1])
		if fn != "" {
			filename = filepath.Clean(fn)
		}
	}

	// Extract code block
	if match := codeBlockRegex.FindStringSubmatch(response); match != nil {
		code = strings.TrimSpace(match[1])
		// Explanation is everything outside the code block
		explanation = strings.TrimSpace(codeBlockRegex.ReplaceAllString(response, ""))
		if explanation != "" {
			explanation = filenameRegex.ReplaceAllString(explanation, "")
			explanation = strings.TrimSpace(explanation)
		}
		return filename, code, explanation
	}

	// If no code block fence found, return response as code if it has lines
	return filename, strings.TrimSpace(response), ""
}

// AskGeminiStream sends a prompt to Gemini and streams chunks back via a Bubble Tea channel.
func AskGeminiStream(ctx context.Context, apiKey string, modelName string, mode AIMode, prompt string, ch chan<- AIChunkMsg) {
	defer close(ch)

	if apiKey == "" {
		ch <- AIChunkMsg{Err: fmt.Errorf("Gemini API key is not configured. Press ^G or enter key when prompted."), Mode: mode}
		return
	}

	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		ch <- AIChunkMsg{Err: fmt.Errorf("failed to create GenAI client: %w", err), Mode: mode}
		return
	}

	for resp, err := range client.Models.GenerateContentStream(ctx, modelName, genai.Text(prompt), nil) {
		if err != nil {
			ch <- AIChunkMsg{Err: fmt.Errorf("streaming error: %w", err), Mode: mode}
			return
		}
		text := resp.Text()
		if text != "" {
			ch <- AIChunkMsg{Chunk: text, Mode: mode}
		}
	}

	ch <- AIChunkMsg{Done: true, Mode: mode}
}

// ListenForAIChunk returns a tea.Cmd that waits for the next message on the AI chunk channel.
func ListenForAIChunk(ch <-chan AIChunkMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return AIChunkMsg{Done: true}
		}
		return msg
	}
}
