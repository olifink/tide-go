# 🌊 TIDE

A minimalist, zero-bloat, discoverable TUI development environment for modern Linux CLI workflows (tailored for C, Go and Rust).

---

## 🌟 Key Features

* **Explicit Over Implicit:** Nano/Pico-style persistent footer bar mapping all primary hotkeys. Zero memorization required.
* **Chroma-Powered Syntax Highlighting:** View mode with full-color syntax highlighting across Go, C, C++, Rust, Python, and more.
* **Dual-Mode Text Editing:** Seamlessly switch between fast Chroma inspection and active `bubbles/textarea` editing (`Ctrl+E`).
* **Smart Build Engine & Error Line Gutter:** Auto-detects Go/C/Make project configurations (`Ctrl+R`), parses compiler diagnostics (`file:line:col`), and highlights error lines with bold red indicators directly in the code viewer.
* **Arbitrary Shell Runner:** Non-blocking asynchronous command execution (`Ctrl+X`) streaming output directly into the console panel.
* **Gemini AI Assistant Integration:** Context-aware AI helper (`Ctrl+G`) automatically attaching your active code buffer and compiler error trace to stream direct explanations and fixes in markdown.

---

## ⌨️ Keyboard Shortcuts

| Hotkey | Action | Scope | Description |
|---|---|---|---|
| `Ctrl+F` | Focus Files / Editor | Global | Toggles focus between the file explorer sidebar and code viewport |
| `Ctrl+N` | New File | Global | Prompts for a filename and opens it in the editor |
| `Ctrl+E` | Toggle View / Edit Mode | Global | Switches between syntax-highlighted View mode and active Textarea editing |
| `Ctrl+S` | Save File | Global | Saves modified buffer to disk |
| `Ctrl+B` | Build Project | Global | Auto-detects build command (`go build .`, `make`, `gcc`) and compiles asynchronously |
| `Alt+B` | Configure Build | Global | Opens shell dialog pre-filled with current build command and saves custom command for session |
| `Ctrl+R` | Run Program | Global | Runs executable or detected run target asynchronously |
| `Alt+R` | Configure Run | Global | Opens shell dialog pre-filled with current run command and saves custom command for session |
| `Ctrl+X` | Shell Command | Global | Opens shell execution bar (`$ `) to run arbitrary CLI commands |
| `Ctrl+G` | Gemini AI Assistant | Global | Context-aware AI: **Update File** (in Editor), **Generate File** (in Files), or **Q&A** (in Console) |
| `Ctrl+Q` | Quit | Global | Exits TIDE |
| `Tab` / `Shift+Tab` | Cycle Focus | Global | Cycles focus between Files, Editor, and Console panels |
| `Esc` | Focus Editor / Cancel | Global | Focuses the Editor pane (or exits Edit mode / closes open modals) |
| `Shift+Esc` | Focus Console | Global | Immediately switches focus to the Output / Console panel |
| `.` | Toggle Hidden Files | Files | Toggles display of hidden dot files and directories (`.gitignore`, `.github`, etc.) |
| `r` | Refresh Files | Files | Re-scans the current workspace directory from disk |
| `Alt+Backspace` | Delete File / Dir | Files | Opens shell dialog pre-filled with `rm -f` or `rm -rf` command for confirmation |
| `Ctrl+Z` (or `F11`) | Toggle Fullscreen / Zen | Global | Expands the code editor to full screen, hiding files and console |
| `z` | Toggle Fullscreen | Editor (View) | Quick full-screen toggle when inside Editor View mode |
| `e` | Enter Edit Mode | Editor (View) | Enters Edit mode with cursor at visible line |
| `w` | Toggle Word Wrap | Editor (View) | Toggles word-wrapping on long lines with `↳` continuation indicators |
| `←` / `→` (or `h` / `l`) | Horizontal Scroll | Editor (View) | Scrolls left/right through long lines when word-wrap is disabled |
| `0` / `Home` | Scroll to Start | Editor (View) | Resets horizontal scroll offset back to column 0 |
| `m` | Maximize / Restore Console | Console | Toggles console height between default and expanded 3/4 screen height |

---

## ✦ Context-Aware Gemini AI Integration (`Ctrl+G`)

TIDE adapts Gemini AI prompts dynamically based on your active panel:

1. **In Editor (`PaneEditor`): Update Active File**
   - Provide instructions like *"add error handling"*, *"refactor to use goroutines"*, or *"fix compiler bug"*.
   - Gemini returns the full updated code, loads it directly into your active buffer, and marks it as modified (`[MOD]`).
   - Review changes and press `Ctrl+S` to save.

2. **In Files Sidebar (`PaneFiles`): Generate New File**
   - Request new components like *"server.go: REST API with health check"* or *"vector.c: 3D vector math functions"*.
   - Gemini creates the file on disk, refreshes the file explorer, and opens the new file in the editor ready to run.

3. **In Output Console (`PaneConsole`): Ask Assistant (Q&A)**
   - Ask general coding questions, explain architectural concepts, or inspect compiler errors.
   - Responses stream in real-time as rich Markdown inside the Console viewport.

---

## 🚀 Installation & Quick Start

### Quick Install (Linux & macOS)

Install the latest pre-compiled binary directly to `~/.local/bin/tide`:

```bash
curl -fsSL https://raw.githubusercontent.com/olifink/tide-go/main/install.sh | bash
```

> **Custom directory:** Pass `BINDIR=/usr/local/bin` (e.g. `curl -fsSL ... | BINDIR=/usr/local/bin bash`).

### Build from Source

```bash
git clone https://github.com/olifink/tide-go.git
cd tide-go
go build -o tide ./cmd/tide
```

### Usage

```bash
# Open IDE in current directory
tide

# Open a specific file (auto-creates if non-existent)
tide main.go

# Open a specific directory
tide ./my-project
```

### Configuring Gemini AI Assistant

TIDE looks for your Gemini API key in the following order:
1. Environment variable: `export GEMINI_API_KEY="your-api-key"`
2. Environment variable: `export GOOGLE_API_KEY="your-api-key"`
3. Config file at `~/.config/tide/config.json`

If no key is found when pressing `Ctrl+G`, TIDE will prompt you in a modal dialog and automatically save it to `~/.config/tide/config.json`.

---

## 🏗️ Project Architecture

```
tide/
├── cmd/
│   └── tide/
│       └── main.go       # Program entrypoint & CLI flags
├── internal/
│   ├── ai/               # Gemini GenAI SDK client & streaming
│   ├── app/              # Top-level Bubble Tea model, ELM state & theme styles
│   ├── config/           # User configuration & API key management
│   ├── console/          # Output viewport with Glamour markdown rendering
│   ├── editor/           # Chroma syntax highlighter & textarea buffer
│   ├── filetree/         # File explorer sidebar component
│   ├── modal/            # Popover dialogs (New File, Shell, Gemini, Key)
│   └── runner/           # Async process runner, builder & error regex parser
├── SPECS.md
└── go.mod
```
