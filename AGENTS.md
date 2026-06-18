# AGENTS.md

## Project Context

dist-lab is a Go terminal UI for loading structured data files, discovering available fields, previewing values at a selected field path, editing the current value set, and exporting the result for analysis.

The normal workflow is:

1. Pick a supported data file.
2. Select a discovered field path.
3. Preview the values returned by that path.
4. Optionally filter, delete, restore, or inspect values.
5. Export the current values to JSON, JSONL, YAML, CSV, or TSV.

## Architecture

- `cmd/main.go` starts the Bubble Tea program.
- `internal/input/parser.go` owns input format decoding, jq path discovery, and selection.
- `internal/input/tui/model.go` owns the top-level TUI state machine.
- `internal/input/tui/file_picker.go` renders and sizes the file picker state.
- `internal/input/tui/fields.go` renders and sizes the field selection state.
- `internal/input/tui/preview.go` owns preview modes, value filtering, value formatting, and value mutation.
- `internal/input/tui/value_list.go` owns the editable value list and selected-value detail panel.
- `internal/input/tui/export.go` owns export prompt state and JSON/JSONL/YAML/CSV/TSV export.
- `internal/input/tui/styles.go` owns shared layout, header/footer/popup rendering, and style helpers.

## State Invariants

- `rawValues` is the source value set returned by the parser for the selected field.
- `values` is the current editable/exportable value set.
- Export always uses `values`, not `rawValues`.
- Filtering nil/empty values should rebuild `values` from `rawValues`.
- Restoring values should rebuild `values` from `rawValues` or the filtered version of `rawValues`, depending on the current filter state.
- Deleting a value should only affect `values`.

## TUI Layout Rules

- Each view should render through `screenView(header, content, footer)`.
- Headers and footers should use the shared helpers in `styles.go`.
- The content area should fill the terminal space between header and footer.
- Popups render after the header and reduce available content height.
- Keep all model methods on pointer receivers unless a nested Bubble Tea model has a reason to remain value-based.

## Testing Guidance

Prefer tests for data behavior and layout calculations over brittle full-screen ANSI snapshots.

High-value coverage:

- Parser file loading, error cases, path discovery, and field selection.
- JSON, JSONL/NDJSON, YAML, CSV, and TSV input behavior.
- Recursive nil/empty filtering and clone behavior.
- Value deletion/restoration and export state.
- JSON, JSONL, YAML, CSV, and TSV export output, including flattened object columns.
- Height calculations for header/content/footer/popup layouts.

Use `t.TempDir()` for filesystem tests. Do not commit generated export files or local data files.

## Development Commands

- Build: `go build ./...`
- Test: `go test ./...`
- Format: `gofmt -w <files>`
