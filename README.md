# dist-lab

dist-lab is a terminal UI for exploring structured data. It loads JSON, JSONL/NDJSON, YAML, CSV, and TSV files, discovers field paths, previews the values at a selected path, lets you adjust the current value set, and exports the result to JSON, JSONL, YAML, CSV, or TSV.

## Run

```sh
go run ./cmd
```

Build a local binary:

```sh
go build -o dist-lab ./cmd
```

## Workflow

1. Pick a supported data file in the file picker.
2. Choose a discovered field path.
3. Preview the values returned by that path.
4. Switch between text preview and editable value list.
5. Filter nil/empty values, delete individual values, or restore from the raw selection.
6. Export the current values to JSON, JSONL, YAML, CSV, or TSV.

## Keys

Global keys:

- `q`: quit
- `ctrl+c`: quit
- `o`: choose a new file
- `a`: add another file to the current parser
- `f`: return to field selection from preview

Preview keys:

- `up/down`: scroll or move selection
- `pgup/pgdn`: page through text preview
- `g/G`: jump to top or bottom in text preview
- `v`: switch between text preview and editable value list
- `e`: toggle recursive nil/empty filtering
- `d`: delete the selected value in editable value list mode
- `r`: restore values from the raw selection
- `i`: inspect analysis for the current values
- `x`: open the export prompt

Analysis keys:

- `up/down`: scroll analysis
- `pgup/pgdn`: page through analysis
- `1`: overview
- `2`: missing-data ranking
- `3`: recursive field analysis
- `/`: filter field paths
- `n` / `N`: jump between matching fields
- `enter`: focus the selected field match
- `p` / `esc`: return to preview

Export prompt keys:

- `enter`: save export
- `tab`: cycle export format
- `esc`: cancel export

## Input

Supported input formats:

- `.json`: one JSON document or multiple JSON documents one after another
- `.jsonl` / `.ndjson`: one JSON document per non-empty line
- `.yaml` / `.yml`: one YAML document or multiple `---` separated YAML documents
- `.csv`: comma-delimited table with a header row
- `.tsv`: tab-delimited table with a header row

YAML values are normalized into JSON-like values before field discovery. CSV and TSV rows are treated as JSON-like objects keyed by their column headers. Field discovery uses jq-style paths internally and shows paths in a generic form such as:

```text
$.viewer.name
$.viewer.friends[].name
$["column with spaces"]
```

## Exports

JSON export writes the current `values` slice with indentation.

JSONL export writes one current value per line.

YAML export writes the current `values` slice as YAML.

CSV and TSV exports flatten object fields into columns using dot paths. Scalar values are written to a `value` column. Nested arrays and complex values inside a cell are JSON-encoded.

## Analysis

The nil/empty filter is recursive but row-preserving: if a selected object or array contains nil/empty data anywhere inside it, the whole selected value is filtered out instead of deleting nested fields and changing its shape.

The analysis view uses the current editable `values`, so filtering and deleted rows are reflected immediately. It has mode pages for overview, missing-data ranking, and recursive field analysis. Field paths can be filtered with `/`, jumped through with `n` / `N`, and focused with `enter`. Analysis shows scalar summaries, numeric distributions, visible outlier values, categorical top values, cardinality hints, boolean counts, percentages, and missing-data rankings. For object and array values, analysis recursively summarizes scalar field paths such as `day`, `year`, or `friends[].name`. Objects with `day`, `month`, and `year` fields are also summarized as dates in the overview.

## Development

```sh
go build ./...
go test ./...
```

The TUI is organized around three states: file picker, field selection, and preview. Shared layout and styling lives in `internal/input/tui/styles.go`.
