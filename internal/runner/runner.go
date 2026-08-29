package runner

import (
	"bytes"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ProcessFinishedMsg is sent to the Bubbletea update loop when a command finishes.
type ProcessFinishedMsg struct {
	Command     string
	Output      string
	ExitCode    int
	Duration    time.Duration
	Diagnostics []Diagnostic
	Error       error
}

// RunCommandCmd returns a tea.Cmd that executes a shell command asynchronously in the specified directory.
func RunCommandCmd(dir string, cmdStr string) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		cmd := exec.Command("bash", "-c", cmdStr)
		if dir != "" {
			cmd.Dir = dir
		}

		var combined bytes.Buffer
		cmd.Stdout = &combined
		cmd.Stderr = &combined

		err := cmd.Run()
		duration := time.Since(start)
		output := combined.String()

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}

		diags := ParseOutput(output)

		return ProcessFinishedMsg{
			Command:     cmdStr,
			Output:      output,
			ExitCode:    exitCode,
			Duration:    duration,
			Diagnostics: diags,
			Error:       err,
		}
	}
}
