# dq-pipeline

Multi-stage data quality assessment and cleansing pipeline (CLI).

Reads tabular data (CSV, TSV, JSON, fixed-width), runs configurable quality rules, infers and validates schema, computes column profiles, tracks row-level transformation lineage, and outputs quality reports with checksummed persistence.

## Architecture

```
parse → schema (infer) → pipeline (DAG orchestration)
                              ├── quality (rule evaluation)
                              ├── transform (column ops)
                              ├── clean (fill/clamp/normalize)
                              └── profile (statistics/outliers)
         ↓
    lineage (append-only log)
    persist (snapshot + checksum)
    report (text / JSON)
```

Packages:

| Package | Role |
|---------|------|
| `internal/parse` | CSV/TSV/JSON/fixed-width parsing, BOM stripping, format detection |
| `internal/schema` | Column type inference, schema persistence, drift detection |
| `internal/pipeline` | Stage interface, DAG topological sort, context cancellation |
| `internal/quality` | Rule evaluation (notnull, range, regex, unique, enum, crossfield, etc.) |
| `internal/transform` | Column transforms (rename, case, filter, dedup, concat, substr) |
| `internal/clean` | Row cleaning (drop critical), fill strategies (mean/median/forward/backward), clamp, normalize |
| `internal/profile` | Column statistics (min/max/mean/median/percentiles), histograms, outlier detection (IQR/Z-score) |
| `internal/lineage` | Row-level transformation tracking, append-only log with truncation recovery |
| `internal/persist` | Snapshot storage with SHA-256 integrity, trend analysis |
| `internal/report` | Text and JSON quality report output |

## Usage

```bash
dq-pipeline -input data.csv [-delim ,] [-rules rules.json] [-out cleaned.csv] [-report -] [-drop] [-trim]
```

- `-input`  Input file (`-` for stdin)
- `-delim`  Field delimiter (default `,`)
- `-rules`  Quality rules JSON (optional)
- `-out`    Write cleaned data to path
- `-report` Quality report output (`-` for stdout)
- `-drop`   Drop rows failing critical rules
- `-trim`   Trim whitespace (default on)

## Rules JSON

```json
[
  {"column":"name","kind":"notnull","critical":true},
  {"column":"age","kind":"range","min":0,"max":120},
  {"column":"email","kind":"regex","pattern":"^[^@]+@[^@]+$"},
  {"column":"id","kind":"unique"},
  {"column":"status","kind":"enum","pattern":"active|inactive|pending"},
  {"column":"code","kind":"length","min":2,"max":10},
  {"column":"date","kind":"date","pattern":"2006-01-02"},
  {"column":"amount","kind":"type","pattern":"float"},
  {"column":"total","kind":"crossfield","pattern":"max gt min"}
]
```

## Build & Test

```bash
export GOTOOLCHAIN=local CGO_ENABLED=0
go build ./...
go test ./...
```

## License

MIT
