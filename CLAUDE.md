# numer — CLAUDE.md

Terminal-based notepad calculator inspired by Numi/Soulver. Written in Go using Bubble Tea (TUI framework) and Lipgloss (styling).

## Project structure

```
numer/
├── main.go                  # Entrypoint — parses args, boots Bubble Tea program
├── internal/
│   ├── eval/
│   │   ├── eval.go          # Lexer, parser, evaluator
│   │   └── eval_test.go     # Unit tests for eval
│   └── ui/
│       ├── model.go         # Bubble Tea model, input handling, rendering
│       └── highlight.go     # Syntax highlighting (per-rune style mapping)
├── examples/                # Sample .nm files
├── go.mod                   # Module: github.com/gfonseca/numer
└── TODO                     # Feature backlog
```

## Architecture

### eval package

Self-contained expression engine with no external dependencies.

**Lexer** (`lexer` struct) — rune-by-rune tokenizer producing `token` values of kinds: `tokNum`, `tokIdent`, `tokPlus`, `tokMinus`, `tokStar`, `tokSlash`, `tokPercent`, `tokCaret`, `tokLParen`, `tokRParen`, `tokComma`, `tokEOF`.

**Parser** (`parser` struct) — recursive descent with standard precedence:
```
parseExpr → parseAddSub → parseMulDiv → parsePow → parseUnary → parsePrimary
```
Power (`^`) is right-associative (implemented via recursion in `parsePow`).

**Evaluator** (`Evaluator` struct) — stateful, evaluates lines sequentially:
- `vars map[string]float64` — user-defined variables + built-in constants (`pi`, `e`, `tau`, `phi`) + `last`
- `pendingSum float64` — accumulator for `sum-total` keyword; resets on each `sum-total` call
- `Reset()` — clears vars and pendingSum; called before full re-evaluation
- `EvalLine(line string) (result, errMsg string)` — main API; both empty = no output (blank/comment)

**Special keywords handled in `EvalLine` before parsing:**
- Blank line / `#` / `//` → no output
- `sum-total` (case-insensitive) → returns accumulated sum, resets `pendingSum`, updates `last`

**Assignment detection** (`detectAssignment`) — scans for `=` that isn't `==`, `!=`, `<=`, `>=`; LHS must be a valid identifier.

**`FormatNum`** — trims trailing zeros, renders integers without decimal point, handles `NaN`/`±∞`.

**Built-in functions:** `sqrt`, `abs`, `floor`, `ceil`, `round`, `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `atan2`, `log`/`ln`, `log2`, `log10`, `exp`, `pow`, `max`, `min`, `sum`.

### ui package

**`model.go`** — Bubble Tea `Model` implementing `Init`/`Update`/`View`.

Key fields:
- `lines []string` — raw text content, one entry per line
- `results []string`, `errors []string` — parallel to `lines`, populated by `reeval()`
- `row`, `col` — cursor position
- `scroll` — top visible line index
- `undoStack`, `redoStack []snapshot` — undo history (max 100)
- `evaluator *eval.Evaluator`

**Re-evaluation** — `reeval()` calls `evaluator.Reset()` then `EvalLine` for every line on each keystroke. Simple and correct; fine for typical sheet sizes.

**Rendering** — `View()` builds a slice of rows. Each content row calls `renderLine(i)`:
- `sum-total` lines → `renderSumTotalLine(i)`: centered `── #TOTAL ──` separator + bold total on right
- Comment lines → full comment style (purple)
- Other lines → `renderWithCursor` (active row) or `renderStatic`, + result/error column on right

**Highlight** (`highlight.go`) — `highlightStyles(line, isCmt)` returns one `lipgloss.Style` per rune. Token types: `hlNum` (gold), `hlOp` (coral), `hlFn` (green), `hlConst` (amber), `hlVar` (lavender), `hlParen` (orange). Applied char-by-char in `applyChar`, which also handles odd-line background.

**Layout constants:**
- `resultColWidth = 24` — right result column width
- `hPad = 3` — horizontal padding each side
- `vPad = 2` — top/bottom blank rows

**Modes:** `modeNormal` (editing) and `modePrompt` (save-as filename input in status bar).

**File format:** `.nm` — plain UTF-8 text, newline-delimited, no special encoding.

## Patterns & conventions

- **No state mutation in `View()`** — all computation in `Update` / `reeval`
- **Defensive copy on every keystroke** — `handleKey` copies `m.lines` slice before mutating
- **Full re-eval on every change** — keeps result/error arrays always in sync; no incremental diffing
- **Parallel arrays** — `lines[i]`, `results[i]`, `errors[i]` always have the same length
- **`pushUndo` before any edit** — call before modifying `m.lines`; `reeval` after
- **`sum-total` isolation** — `pendingSum` is never incremented by a `sum-total` result itself, so chaining multiple `sum-total` blocks is safe

## Build & install

```bash
go build ./...        # build
go test ./...         # run tests
go install .          # install binary to $GOPATH/bin
numer file.nm         # open file (created if not exists)
numer                 # untitled session
```

## Keybindings

| Key | Action |
|-----|--------|
| Arrow keys | Move cursor |
| Home / Ctrl+A | Start of line |
| End / Ctrl+E | End of line |
| PgUp / PgDn | Page scroll |
| Enter | New line |
| Backspace / Delete | Delete character |
| Ctrl+K | Kill to end of line |
| Ctrl+Z / Ctrl+Y | Undo / Redo |
| Ctrl+S | Save (prompts for filename if untitled) |
| Ctrl+Q / Ctrl+C | Quit |
