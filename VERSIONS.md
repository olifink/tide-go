# 🌊 TIDE - Release Changelog & Version Highlights

All notable changes, version highlights, and feature additions for TIDE are documented in this file.

---

## [v0.2.0] - 2026-08-30

### 🌿 Automatic Git Mode & One-Action Sync
* **Automatic Detection:** Automatically discovers Git repositories and verifies system `git` installation upon startup without requiring manual configuration.
* **Title Bar Git Badge:** Displays active branch and status (`git:main` when clean, `git:main*` in warm peach when uncommitted changes exist).
* **Subtle Changed File Highlighting:** Distinct, subtle color highlights in the Files sidebar for modified (`#F9E2AF` peach), untracked/added (`#A6E3A1` green), and deleted files (`#F38BA8` red), with directory propagation.
* **One-Action Git Sync (`Ctrl+Shift+G` or `Alt+G`):** Opens a dedicated Git modal to enter a commit message that automatically executes `git add -A && git commit -m "<msg>" && git push` in a single non-blocking action streaming output directly to the Console.
* **Zero UI Footer Clutter:** Designed for maximum discoverability and zero visual bloat without adding unnecessary hints to the persistent Pico footer bar.

### 🖥️ Full-Screen / Zen Editor Mode
* **Full-Screen Toggle:** Press `Ctrl+Z` (or `F11`) globally or single-key `z` in View mode to expand the code editor to 100% of terminal space, hiding the Files sidebar and Console.
* **Auto Full-Screen on Launch:** Launching with a specific file parameter (e.g. `tide main.go`, `tide script.py`) automatically enters full-screen mode for an instant distraction-free editing workflow.
* **Seamless Restoration:** Restores the standard multi-pane layout anytime via `Ctrl+Z`, `z`, `Ctrl+F` (focus files), or `Shift+Esc` (focus console).

### 📜 Word Wrap & Horizontal Navigation
* **Word Wrap Toggle:** Press `w` in View mode to dynamically toggle word wrapping on long lines, complete with clean `↳` continuation arrows.
* **Horizontal Scrolling:** When word wrap is disabled, pan sideways across long lines with `←` / `→` (or `h` / `l`) and jump back to column 0 with `0` or `Home`.

### 🦀 Rust & Cargo Integration
* **Project & Target Detection:** Detects `Cargo.toml` projects (`cargo build`, `cargo run`) and standalone Rust `.rs` files (`rustc <file>.rs`).
* **Multi-Line Diagnostic Parsing:** Robust error/warning parser for `rustc`/`cargo` output with in-editor error gutter highlights.

### ⚡ Session-Scoped Build & Run Customization
* **`Alt+B` (Configure Build):** Opens an interactive shell dialog pre-filled with the current build command, allowing immediate execution and saving the custom build command for all subsequent `Ctrl+B` runs in the session.
* **`Alt+R` (Configure Run):** Opens an interactive shell dialog pre-filled with the current run command, saving custom arguments/binaries for subsequent `Ctrl+R` runs in the session.

### 📁 File Tree Improvements
* **Hidden Files Toggle (`.`):** Press `.` in the Files pane to toggle display of dotfiles and hidden directories (`.gitignore`, `.github`, `.env`, etc.).
* **Quick Delete / Remove (`Alt+Backspace`):** Press `Alt+Backspace` on any selected file or directory in the sidebar to open the shell dialog pre-populated with `rm -f <file>` or `rm -rf <dir>` for safe review and execution.
* **Clear Run vs. View Precedence:** `Enter` on the file tree reliably opens files in the editor, while `Ctrl+R` prioritizes executing selected binaries/scripts over fallback Makefiles.

### 🎯 Editor Precision & Indentation
* **Tab Character Preservation:** Automatic normalization of leading spaces to true `\t` tab characters for Makefiles (`Makefile`, `*.mk`) and Go files, preventing `*** missing separator` errors.
* **View/Edit Cursor Synchronization:** Entering Edit mode (via `e` or `Ctrl+E`) places the cursor directly at column 0 of the top visible line in View mode (`m.ScrollLine`) instead of jumping to the end of the file.
* **Edit Mode Navigation:** Added `PageUp`/`PageDown` (half-page scrolling), `Home`/`End` (line start/end), `Ctrl+Home`/`Ctrl+End` (file start/end), and `Ctrl+Left`/`Ctrl+Right` (word jumping) in Edit mode.

### 📦 CI/CD & Distribution
* **GitHub Actions Multi-Platform Workflow:** Automated release matrix building Linux, macOS (Darwin), and Windows binaries across `amd64` and `arm64` architectures with SHA256 checksums on versioned tags (`v*`).
* **Scriptable `curl` Installer:** Standalone `install.sh` for auto-detecting platform/architecture, downloading the latest GitHub release, and installing to `~/.local/bin/tide`.

---

## [v0.1.0] - Initial Release

### 🌊 Core TIDE Features
* **Three-Pane TUI Layout:** Responsive file tree sidebar, syntax-highlighted code editor, and interactive asynchronous output console.
* **Nano/Pico Discoverability:** Persistent bottom footer bar with standard hotkey mappings (`^F`, `^N`, `^E`, `^S`, `^B`, `^R`, `^X`, `^G`, `^Q`).
* **Border-Embedded Headers:** Panel titles and context hints embedded cleanly directly into the top borders of panels.
* **Chroma Syntax Highlighting:** Multi-language syntax highlighting supporting Go, C, C++, Rust, Python, Makefile, Markdown, JSON, YAML, and Shell.
* **Smart Build & Error Gutter:** Automatic detection for `go build`, `make`, `gcc`, parsing compiler diagnostics and rendering red indicators on erroneous lines.
* **Interactive Modals:** Popover modal dialogs for New File (`Ctrl+N`), Shell Command (`Ctrl+X`), and Gemini API Key configuration.
* **Gemini AI Integration (`Ctrl+G`):** Context-aware streaming AI assistant:
  * **Editor:** Direct code update and buffer insertion (`[MOD]`).
  * **Files:** Code generation directly saved and opened in workspace.
  * **Console:** Q&A troubleshooting and error trace explanations.
* **Safety Guards:** Automatic rejection of binary files and large files (>2MB) to maintain instantaneous responsiveness.
