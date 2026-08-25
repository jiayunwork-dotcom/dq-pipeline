package quality

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"dq-pipeline/internal/parse"
)

func EvaluateExtended(t *parse.Table, rules []Rule) ([]Violation, error) {
	idx := make(map[string]int, len(t.Header))
	for i, h := range t.Header {
		idx[h] = i
	}
	var viols []Violation
	for ri, rule := range rules {
		ci, ok := idx[rule.Column]
		if !ok {
			return nil, fmt.Errorf("extended rule %d: column %q not found", ri, rule.Column)
		}
		switch rule.Kind {
		case "enum":
			v := evalEnum(t, ci, ri, rule)
			viols = append(viols, v...)
		case "length":
			v := evalLength(t, ci, ri, rule)
			viols = append(viols, v...)
		case "date":
			v := evalDate(t, ci, ri, rule)
			viols = append(viols, v...)
		case "type":
			v := evalType(t, ci, ri, rule)
			viols = append(viols, v...)
		case "crossfield":
			v, err := evalCrossfield(t, idx, ri, rule)
			if err != nil {
				return nil, err
			}
			viols = append(viols, v...)
		}
	}
	return viols, nil
}

func evalEnum(t *parse.Table, ci, ri int, rule Rule) []Violation {
	allowed := parseEnumValues(rule.Pattern)
	if len(allowed) == 0 {
		return nil
	}
	set := make(map[string]bool)
	for _, v := range allowed {
		set[v] = true
	}
	var viols []Violation
	for rowi, row := range t.Rows {
		val := cellAt(row, ci)
		if val == "" {
			continue
		}
		if !set[val] {
			viols = append(viols, Violation{ri, rule.Column, rowi, "value not in enum"})
		}
	}
	return viols
}

func evalLength(t *parse.Table, ci, ri int, rule Rule) []Violation {
	minLen := int(rule.Min)
	maxLen := int(rule.Max)
	var viols []Violation
	for rowi, row := range t.Rows {
		val := cellAt(row, ci)
		if val == "" {
			continue
		}
		l := len(val)
		if minLen > 0 && l < minLen {
			viols = append(viols, Violation{ri, rule.Column, rowi, "too short"})
		}
		if maxLen > 0 && l > maxLen {
			viols = append(viols, Violation{ri, rule.Column, rowi, "too long"})
		}
	}
	return viols
}

func evalDate(t *parse.Table, ci, ri int, rule Rule) []Violation {
	layout := rule.Pattern
	if layout == "" {
		layout = "2006-01-02"
	}
	var viols []Violation
	for rowi, row := range t.Rows {
		val := cellAt(row, ci)
		if val == "" {
			continue
		}
		if _, err := time.Parse(layout, val); err != nil {
			viols = append(viols, Violation{ri, rule.Column, rowi, "invalid date format"})
		}
	}
	return viols
}

func evalType(t *parse.Table, ci, ri int, rule Rule) []Violation {
	expectedType := rule.Pattern
	var viols []Violation
	for rowi, row := range t.Rows {
		val := cellAt(row, ci)
		if val == "" {
			continue
		}
		switch expectedType {
		case "int":
			if _, err := strconv.Atoi(val); err != nil {
				viols = append(viols, Violation{ri, rule.Column, rowi, "not an integer"})
			}
		case "float":
			if _, err := strconv.ParseFloat(val, 64); err != nil {
				viols = append(viols, Violation{ri, rule.Column, rowi, "not a float"})
			}
		case "bool":
			lower := strings.ToLower(val)
			if lower != "true" && lower != "false" && lower != "0" && lower != "1" {
				viols = append(viols, Violation{ri, rule.Column, rowi, "not a boolean"})
			}
		}
	}
	return viols
}

func evalCrossfield(t *parse.Table, idx map[string]int, ri int, rule Rule) ([]Violation, error) {
	parts := strings.Fields(rule.Pattern)
	if len(parts) != 3 {
		return nil, fmt.Errorf("crossfield pattern must be 'col1 op col2', got %q", rule.Pattern)
	}
	col1, op, col2 := parts[0], parts[1], parts[2]
	ci1, ok1 := idx[col1]
	ci2, ok2 := idx[col2]
	if !ok1 {
		return nil, fmt.Errorf("crossfield: column %q not found", col1)
	}
	if !ok2 {
		return nil, fmt.Errorf("crossfield: column %q not found", col2)
	}

	var viols []Violation
	for rowi, row := range t.Rows {
		v1 := cellAt(row, ci1)
		v2 := cellAt(row, ci2)
		if v1 == "" || v2 == "" {
			continue
		}
		f1, err1 := strconv.ParseFloat(v1, 64)
		f2, err2 := strconv.ParseFloat(v2, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		violated := false
		switch op {
		case "gt":
			violated = !(f1 > f2)
		case "lt":
			violated = !(f1 < f2)
		case "eq":
			violated = !(f1 == f2)
		case "gte":
			violated = !(f1 >= f2)
		case "lte":
			violated = !(f1 <= f2)
		}
		if violated {
			viols = append(viols, Violation{ri, rule.Column, rowi, fmt.Sprintf("%s not %s %s", col1, op, col2)})
		}
	}
	return viols, nil
}

func parseEnumValues(pattern string) []string {
	if pattern == "" {
		return nil
	}
	parts := strings.Split(pattern, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func cellAt(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}

func Summarize(rep Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Score: %.4f (%d violations / %d checks)\n", rep.Score, len(rep.Violations), rep.TotalChecks)
	fmt.Fprintf(&b, "Rows: %d\n", rep.TotalRows)
	if len(rep.ByColumn) > 0 {
		fmt.Fprintf(&b, "By column:\n")
		for col, count := range rep.ByColumn {
			fmt.Fprintf(&b, "  %s: %d\n", col, count)
		}
	}
	return b.String()
}

func MergeViolations(a, b []Violation) []Violation {
	type key struct {
		rule int
		col  string
		row  int
	}
	seen := make(map[key]bool)
	var out []Violation
	for _, v := range a {
		k := key{v.Rule, v.Column, v.Row}
		if !seen[k] {
			seen[k] = true
			out = append(out, v)
		}
	}
	for _, v := range b {
		k := key{v.Rule, v.Column, v.Row}
		if !seen[k] {
			seen[k] = true
			out = append(out, v)
		}
	}
	return out
}
