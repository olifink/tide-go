# SPECS.md: Minimalist TUI Development Environment ("tide")

## 1. Vision & Core Philosophy
A zero-bloat, discoverable TUI development environment for modern Linux CLI workflows (specifically C and Go). 

* **Explicit Over Implicit:** All primary actions are permanently displayed in a Nano/Pico-style footer bar at the bottom of the screen. Zero hidden keybindings or required manual reading. Leaves no traces - no local `.something` configurations or settings in project directory. 
* **Unix-Compliant & Lightweight:** Low resource footprint, single Go binary, instant startup.
* **Clean Visual Hierarchy:** Color is used liberally for syntax and status indicators, but background fills are strictly reserved for active highlights or high-severity errors.


---

## 2. Technical Architecture & Dependencies

* **Language:** Go (1.22+)
* **UI Framework:** [`github.com/charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea) (ELM architecture pattern)
* **Styling & Layout:** [`github.com/charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss)
* **Text Editing:** Extended [`github.com/charmbracelet/bubbles/textarea`](https://github.com/charmbracelet/bubbles/tree/master/textarea)
* **Syntax Highlighting:** [`github.com/alecthomas/chroma/v2`](https://github.com/alecthomas/chroma)
* **AI Engine:** Official Google Gen AI SDK (`google.golang.org/genai`)

---

## 3. Layout & Visual Composition

The layout fills 100% of the active terminal viewport ($100vh \times 100vw$), split into four distinct regions:


```

+-----------------------------------------------------------------------+
| [HEADER] pico-ide v0.1 | Dir: ~/projects/demo | Active: main.go (MOD) |
+------------------+----------------------------------------------------+
| FILES            | EDITOR / VIEWPORT                                  |
|                  |                                                    |
| > main.go        |  1 | package main                                  |
|   utils.go       |  2 |                                               |
|   go.mod         |  3 | func main() {                                 |
|                  |  4 |     fmt.Println("Hello")                      |
|                  |  5 | }                                             |
+------------------+----------------------------------------------------+
| CONSOLE / OUTPUT / AI OVERLAY                                         |
| $ go build .                                                          |
| main.go:4:5: undefined: fmt.Println                                   |
+-----------------------------------------------------------------------+
| [FOOTER] ^F Files  ^N New  ^E Edit  ^R Run/Build  ^X Shell  ^G Gemini |
+-----------------------------------------------------------------------+

```

1. **Header Bar (1 Row):** Displays application title, current working directory, open filename, and modification status (`[MOD]`).
2. **Main Split View (Flexible Height):**
   * **Left Panel (File Tree / Explorer):** Fixed width (24 chars). Displays files in current directory.
   * **Right Panel (Editor / Chroma Viewer):** Flexible width. Displays source code with syntax highlighting and line numbers.
3. **Console / Output Panel (Collapsible, 6-10 Rows):** Displays stdout/stderr from process commands, compiler output, or Gemini AI stream.
4. **Pico Footer Bar (2 Rows):** Persistent cheat sheet mapping primary hotkeys to actions.

---

## 4. Feature Specifications (MVP)

### Feature 1: File Navigation & Viewport (Chroma-Powered)
* Read files from the current working directory (`.`).
* Navigate tree using Up/Down arrows or `Ctrl+F`.
* Selecting a file loads it into the viewer.
* Standard view mode renders source files via **Chroma** for full-color syntax highlighting.

### Feature 2: File Creation & Basic Editing (Extended `textarea`)
* Pressing `Ctrl+N` prompts for a filename and creates a new file.
* Pressing `Ctrl+E` switches the viewport into active edit mode powered by `bubbles/textarea`.
* Editing retains line numbering.
* `Ctrl+S` saves changes back to disk and triggers a background syntax/error check.

### Feature 3: Arbitrary Process Command Runner
* Pressing `Ctrl+X` opens an inline command execution bar at the bottom panel (`$ `).
* Allows spawning generic processes (`go mod init`, `git status`, `touch`, `mkdir`, `rm`).
* Standard output and standard error are captured asynchronously without freezing the TUI, streaming directly into the console panel.

### Feature 4: Build Engine & Smart Error Line Parsing
* Pressing `Ctrl+R` executes the configured build command (auto-detected: `go build` for Go, `make` or `gcc main.c -o app` for C).
* **Error Line Parser:** Regex-based output parser extracts file names, line numbers, and error messages from stderr (e.g., `main.go:12:4: error...`).
* Errors populate the console output pane.
* In the Editor view, lines matching compiler errors receive a **bold red background strip** on the specific line number/gutter to indicate build failures.

### Feature 5: Gemini AI "Exit Hatch" Integration
* Pressing `Ctrl+G` opens the Gemini Assistant pane.
* Context-Aware Prompting: Automatically attaches the current editor buffer and the latest compiler output / stderr trace to the prompt request payload.
* Streams response back into the console/output pane using standard Markdown styling.

---

## 5. Keyboard Navigation & Shortcuts

| Hotkey | Action | Scope |
|---|---|---|
| `Ctrl+F` | Focus / Unfocus File Sidebar | Global |
| `Ctrl+N` | Create New File | Global |
| `Ctrl+E` | Toggle View / Edit Mode | Global |
| `Ctrl+S` | Save File | Editor Mode |
| `Ctrl+R` | Run / Build Project | Global |
| `Ctrl+X` | Open Generic Shell Command Prompt | Global |
| `Ctrl+G` | Open Gemini AI Assistant Prompt | Global |
| `Ctrl+Q` | Quit Application | Global |

---

## 6. Development Phasing Plan

* **Phase 1: Foundation Setup**
  * Initialize Go module.
  * Implement base `bubbletea` model layout with header, main body splits, and persistent Pico-style footer using `lipgloss`.
* **Phase 2: File Navigation & Viewing**
  * Integrate file list reading.
  * Integrate `chroma` syntax renderer for non-editing file inspection.
* **Phase 3: Text Editing Subsystem**
  * Embed `bubbles/textarea` into the right pane for editing.
  * Connect file saving (`Ctrl+S`) and new file creation (`Ctrl+N`).
* **Phase 4: Process Engine & Error Parser**
  * Build non-blocking process launcher using `tea.ExecProcess` or `exec.Command` background `tea.Cmd`.
  * Add regex string parser for Go/C error formats and implement line-highlight rendering logic for errors.
* **Phase 5: Gemini AI Integration**
  * Integrate `google.golang.org/genai` client.
  * Implement prompt drawer + error payload context assembler.
  * Prompt for a Gemini API key on first use and store it at `~/.config/tide` 

