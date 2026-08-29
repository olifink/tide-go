package runner

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Diagnostic represents a compiler error, warning, or note extracted from process output.
type Diagnostic struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Col      int    `json:"col,omitempty"`
	Severity string `json:"severity"` // "error", "warning", "note"
	Message  string `json:"message"`
	Raw      string `json:"raw"`
}

// Regex patterns for Go and C (GCC/Clang) compiler errors.
var (
	// Go compiler format: filename.go:line:col: message or filename.go:line: message
	goErrorRegex = regexp.MustCompile(`(?m)^([a-zA-Z0-9_.\-\\/]+\.go):(\d+)(?::(\d+))?:\s*(.+)$`)

	// C/C++ compiler format: filename.c:line:col: (fatal )?error|warning|note: message
	cErrorRegex = regexp.MustCompile(`(?m)^([a-zA-Z0-9_.\-\\/]+\.[ch](?:pp|xx|\+\+)?):(\d+)(?::(\d+))?:\s*(?:fatal\s+)?(error|warning|note):\s*(.+)$`)

	// Generic format: filename.ext:line:col: message or filename.ext:line: message
	genericErrorRegex = regexp.MustCompile(`(?m)^([a-zA-Z0-9_.\-\\/]+\.[a-zA-Z0-9_]+):(\d+)(?::(\d+))?:\s*(?:(error|warning|note):\s*)?(.+)$`)
)

// ParseOutput extracts diagnostics from compiler stdout/stderr output.
func ParseOutput(output string) []Diagnostic {
	var diagnostics []Diagnostic
	seen := make(map[string]bool)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try C/C++ error format first (more specific with severity)
		if match := cErrorRegex.FindStringSubmatch(line); match != nil {
			lineNum, _ := strconv.Atoi(match[2])
			colNum := 0
			if match[3] != "" {
				colNum, _ = strconv.Atoi(match[3])
			}
			severity := strings.ToLower(match[4])
			msg := match[5]
			key := match[1] + ":" + match[2] + ":" + msg
			if !seen[key] {
				seen[key] = true
				diagnostics = append(diagnostics, Diagnostic{
					File:     filepath.Clean(match[1]),
					Line:     lineNum,
					Col:      colNum,
					Severity: severity,
					Message:  msg,
					Raw:      line,
				})
			}
			continue
		}

		// Try Go error format
		if match := goErrorRegex.FindStringSubmatch(line); match != nil {
			lineNum, _ := strconv.Atoi(match[2])
			colNum := 0
			if match[3] != "" {
				colNum, _ = strconv.Atoi(match[3])
			}
			msg := match[4]
			key := match[1] + ":" + match[2] + ":" + msg
			if !seen[key] {
				seen[key] = true
				diagnostics = append(diagnostics, Diagnostic{
					File:     filepath.Clean(match[1]),
					Line:     lineNum,
					Col:      colNum,
					Severity: "error",
					Message:  msg,
					Raw:      line,
				})
			}
			continue
		}

		// Try Generic error format if it looks like a diagnostic
		if match := genericErrorRegex.FindStringSubmatch(line); match != nil {
			lineNum, _ := strconv.Atoi(match[2])
			colNum := 0
			if match[3] != "" {
				colNum, _ = strconv.Atoi(match[3])
			}
			severity := "error"
			if match[4] != "" {
				severity = strings.ToLower(match[4])
			}
			msg := match[5]
			key := match[1] + ":" + match[2] + ":" + msg
			if !seen[key] {
				seen[key] = true
				diagnostics = append(diagnostics, Diagnostic{
					File:     filepath.Clean(match[1]),
					Line:     lineNum,
					Col:      colNum,
					Severity: severity,
					Message:  msg,
					Raw:      line,
				})
			}
		}
	}

	return diagnostics
}

// MatchesFile checks if a diagnostic belongs to the given file path.
func MatchesFile(diagFile, targetFile string) bool {
	diagClean := filepath.Clean(diagFile)
	targetClean := filepath.Clean(targetFile)

	if diagClean == targetClean {
		return true
	}
	if filepath.Base(diagClean) == filepath.Base(targetClean) {
		return true
	}
	if strings.HasSuffix(targetClean, diagClean) || strings.HasSuffix(diagClean, targetClean) {
		return true
	}
	return false
}

// DiagnosticsForFile returns a map of line numbers to diagnostics for a specific file.
func DiagnosticsForFile(diagnostics []Diagnostic, filePath string) map[int][]Diagnostic {
	result := make(map[int][]Diagnostic)
	for _, diag := range diagnostics {
		if MatchesFile(diag.File, filePath) {
			result[diag.Line] = append(result[diag.Line], diag)
		}
	}
	return result
}
