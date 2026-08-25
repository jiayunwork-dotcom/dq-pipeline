package transform

import (
	"fmt"
	"strconv"
	"strings"

	"dq-pipeline/internal/parse"
)

type Op struct {
	Kind      string
	Column    string
	NewName   string
	Columns   []string
	Sep       string
	Start     int
	End       int
	Old       string
	New       string
	Predicate string
}

func Apply(t *parse.Table, ops []Op) (*parse.Table, error) {
	result := copyTable(t)
	for i, op := range ops {
		var err error
		result, err = applyOne(result, op)
		if err != nil {
			return nil, fmt.Errorf("transform op[%d] %q: %w", i, op.Kind, err)
		}
	}
	return result, nil
}

func applyOne(t *parse.Table, op Op) (*parse.Table, error) {
	switch op.Kind {
	case "rename":
		return applyRename(t, op)
	case "to_lower":
		return applyColumnFunc(t, op.Column, strings.ToLower)
	case "to_upper":
		return applyColumnFunc(t, op.Column, strings.ToUpper)
	case "to_int":
		return applyColumnFunc(t, op.Column, toIntStr)
	case "to_float":
		return applyColumnFunc(t, op.Column, toFloatStr)
	case "trim":
		return applyColumnFunc(t, op.Column, strings.TrimSpace)
	case "concat":
		return applyConcat(t, op)
	case "substr":
		return applySubstr(t, op)
	case "replace":
		return applyReplace(t, op)
	case "filter":
		return applyFilter(t, op)
	case "dedup":
		return applyDedup(t, op)
	case "select":
		return applySelect(t, op)
	default:
		return nil, fmt.Errorf("unknown transform kind %q", op.Kind)
	}
}

func applyRename(t *parse.Table, op Op) (*parse.Table, error) {
	idx := colIndex(t, op.Column)
	if idx < 0 {
		return nil, fmt.Errorf("column %q not found", op.Column)
	}
	out := copyTable(t)
	out.Header[idx] = op.NewName
	return out, nil
}

func applyColumnFunc(t *parse.Table, col string, fn func(string) string) (*parse.Table, error) {
	idx := colIndex(t, col)
	if idx < 0 {
		return nil, fmt.Errorf("column %q not found", col)
	}
	out := copyTable(t)
	for i := range out.Rows {
		if idx < len(out.Rows[i]) {
			out.Rows[i][idx] = fn(out.Rows[i][idx])
		}
	}
	return out, nil
}

func applyConcat(t *parse.Table, op Op) (*parse.Table, error) {
	if len(op.Columns) < 2 {
		return nil, fmt.Errorf("concat needs at least 2 columns")
	}
	indices := make([]int, len(op.Columns))
	for i, c := range op.Columns {
		idx := colIndex(t, c)
		if idx < 0 {
			return nil, fmt.Errorf("column %q not found", c)
		}
		indices[i] = idx
	}
	out := copyTable(t)
	newCol := op.Column
	if newCol == "" {
		newCol = strings.Join(op.Columns, "_")
	}
	out.Header = append(out.Header, newCol)
	for i := range out.Rows {
		parts := make([]string, len(indices))
		for j, idx := range indices {
			if idx < len(out.Rows[i]) {
				parts[j] = out.Rows[i][idx]
			}
		}
		out.Rows[i] = append(out.Rows[i], strings.Join(parts, op.Sep))
	}
	return out, nil
}

func applySubstr(t *parse.Table, op Op) (*parse.Table, error) {
	idx := colIndex(t, op.Column)
	if idx < 0 {
		return nil, fmt.Errorf("column %q not found", op.Column)
	}
	out := copyTable(t)
	for i := range out.Rows {
		if idx >= len(out.Rows[i]) {
			continue
		}
		s := out.Rows[i][idx]
		start := op.Start
		end := op.End
		if start < 0 {
			start = 0
		}
		if end <= 0 || end > len(s) {
			end = len(s)
		}
		if start > len(s) {
			start = len(s)
		}
		out.Rows[i][idx] = s[start:end]
	}
	return out, nil
}

func applyReplace(t *parse.Table, op Op) (*parse.Table, error) {
	idx := colIndex(t, op.Column)
	if idx < 0 {
		return nil, fmt.Errorf("column %q not found", op.Column)
	}
	out := copyTable(t)
	for i := range out.Rows {
		if idx < len(out.Rows[i]) {
			out.Rows[i][idx] = strings.ReplaceAll(out.Rows[i][idx], op.Old, op.New)
		}
	}
	return out, nil
}

func applyFilter(t *parse.Table, op Op) (*parse.Table, error) {
	idx := colIndex(t, op.Column)
	if idx < 0 {
		return nil, fmt.Errorf("column %q not found", op.Column)
	}
	pred, err := parsePredicate(op.Predicate)
	if err != nil {
		return nil, err
	}
	out := &parse.Table{Header: append([]string(nil), t.Header...)}
	for _, row := range t.Rows {
		val := ""
		if idx < len(row) {
			val = row[idx]
		}
		if pred(val) {
			out.Rows = append(out.Rows, append([]string(nil), row...))
		}
	}
	return out, nil
}

func applyDedup(t *parse.Table, op Op) (*parse.Table, error) {
	idx := colIndex(t, op.Column)
	if idx < 0 {
		return nil, fmt.Errorf("column %q not found", op.Column)
	}
	seen := make(map[string]bool)
	out := &parse.Table{Header: append([]string(nil), t.Header...)}
	for _, row := range t.Rows {
		key := ""
		if idx < len(row) {
			key = row[idx]
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out.Rows = append(out.Rows, append([]string(nil), row...))
	}
	return out, nil
}

func applySelect(t *parse.Table, op Op) (*parse.Table, error) {
	if len(op.Columns) == 0 {
		return nil, fmt.Errorf("select needs at least 1 column")
	}
	indices := make([]int, len(op.Columns))
	for i, c := range op.Columns {
		idx := colIndex(t, c)
		if idx < 0 {
			return nil, fmt.Errorf("column %q not found", c)
		}
		indices[i] = idx
	}
	out := &parse.Table{}
	for _, idx := range indices {
		out.Header = append(out.Header, t.Header[idx])
	}
	for _, row := range t.Rows {
		nrow := make([]string, len(indices))
		for i, idx := range indices {
			if idx < len(row) {
				nrow[i] = row[idx]
			}
		}
		out.Rows = append(out.Rows, nrow)
	}
	return out, nil
}

func parsePredicate(s string) (func(string) bool, error) {
	switch {
	case s == "notempty":
		return func(v string) bool { return v != "" }, nil
	case strings.HasPrefix(s, "gt:"):
		n, err := strconv.ParseFloat(s[3:], 64)
		if err != nil {
			return nil, fmt.Errorf("predicate gt: %w", err)
		}
		return func(v string) bool {
			f, e := strconv.ParseFloat(v, 64)
			return e == nil && f > n
		}, nil
	case strings.HasPrefix(s, "lt:"):
		n, err := strconv.ParseFloat(s[3:], 64)
		if err != nil {
			return nil, fmt.Errorf("predicate lt: %w", err)
		}
		return func(v string) bool {
			f, e := strconv.ParseFloat(v, 64)
			return e == nil && f < n
		}, nil
	case strings.HasPrefix(s, "eq:"):
		want := s[3:]
		return func(v string) bool { return v == want }, nil
	case strings.HasPrefix(s, "contains:"):
		sub := s[9:]
		return func(v string) bool { return strings.Contains(v, sub) }, nil
	default:
		return nil, fmt.Errorf("unknown predicate %q", s)
	}
}

func colIndex(t *parse.Table, name string) int {
	for i, h := range t.Header {
		if h == name {
			return i
		}
	}
	return -1
}

func copyTable(t *parse.Table) *parse.Table {
	out := &parse.Table{Header: append([]string(nil), t.Header...)}
	for _, row := range t.Rows {
		out.Rows = append(out.Rows, append([]string(nil), row...))
	}
	return out
}

func toIntStr(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return strconv.Itoa(int(f))
}

func toFloatStr(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
