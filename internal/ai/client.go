package ai

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/genai"
)

// AIChunkMsg is sent to the Bubbletea event loop as streaming chunks arrive from Gemini.
type AIChunkMsg struct {
	Chunk string
	Done  bool
	Err   error
}

// BuildPrompt constructs a context-rich prompt for Gemini including the active buffer and compiler errors.
func BuildPrompt(userQuery string, activeFile string, codeContent string, lastCommand string, lastOutput string) string {
	var sb strings.Builder

	sb.WriteString("You are an expert software developer and debugging assistant inside the TIDE terminal IDE.\n\n")

	if activeFile != "" && codeContent != "" {
		ext := strings.TrimPrefix(filepath.Ext(activeFile), ".")
		if ext == "" {
			ext = "text"
		}
		sb.WriteString(fmt.Sprintf("### Active File: %s\n```%s\n%s\n```\n\n", filepath.Base(activeFile), ext, codeContent))
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
	sb.WriteString("\n\nPlease provide a clear, concise, and helpful response. If fixing code, explain the cause and provide the corrected code snippet.")

	return sb.String()
}

// AskGeminiStream sends a prompt to Gemini and streams chunks back via a Bubble Tea channel.
func AskGeminiStream(ctx context.Context, apiKey string, modelName string, prompt string, ch chan<- AIChunkMsg) {
	defer close(ch)

	if apiKey == "" {
		ch <- AIChunkMsg{Err: fmt.Errorf("Gemini API key is not configured. Press ^G or enter key when prompted.")}
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
		ch <- AIChunkMsg{Err: fmt.Errorf("failed to create GenAI client: %w", err)}
		return
	}

	for resp, err := range client.Models.GenerateContentStream(ctx, modelName, genai.Text(prompt), nil) {
		if err != nil {
			ch <- AIChunkMsg{Err: fmt.Errorf("streaming error: %w", err)}
			return
		}
		text := resp.Text()
		if text != "" {
			ch <- AIChunkMsg{Chunk: text}
		}
	}

	ch <- AIChunkMsg{Done: true}
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
