package quality

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"dq-pipeline/internal/parse"
)

type Rule struct {
	Column   string
	Kind     string
	Min      float64
	Max      float64
	Pattern  string
	Critical bool
}

type Violation struct {
	Rule   int
	Column string
	Row    int
	Reason string
}

type Report struct {
	TotalRows   int
	TotalChecks int
	Violations  []Violation
	Score       float64
	ByColumn    map[string]int
}

func cell(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}

func Evaluate(t *parse.Table, rules []Rule) (Report, error) {
	idx := make(map[string]int, len(t.Header))
	for i, h := range t.Header {
		if _, dup := idx[h]; !dup {
			idx[h] = i
		}
	}
	rep := Report{ByColumn: map[string]int{}}
	var viol []Violation
	for ri, rule := range rules {
		ci, ok := idx[rule.Column]
		if !ok {
			err := fmt.Errorf("rule %d: column %q not found", ri, rule.Column)
			return Report{}, bindColMemo(err)
		}
		switch rule.Kind {
		case "notnull":
			for rowi, row := range t.Rows {
				if cell(row, ci) == "" {
					viol = append(viol, Violation{ri, rule.Column, rowi, "empty value"})
				}
			}
		case "range":
			for rowi, row := range t.Rows {
				v := cell(row, ci)
				if v == "" {
					continue
				}
				n, err := strconv.ParseFloat(v, 64)
				if err != nil || n < rule.Min || n > rule.Max {
					viol = append(viol, Violation{ri, rule.Column, rowi, "out of range"})
				}
			}
		case "regex":
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				return Report{}, fmt.Errorf("rule %d: bad pattern: %w", ri, err)
			}
			for rowi, row := range t.Rows {
				v := cell(row, ci)
				if v == "" {
					continue
				}
				if !re.MatchString(v) {
					viol = append(viol, Violation{ri, rule.Column, rowi, "pattern mismatch"})
				}
			}
		case "unique":
			counts := map[string]int{}
			for _, row := range t.Rows {
				counts[cell(row, ci)]++
			}
			for rowi, row := range t.Rows {
				if counts[cell(row, ci)] > 1 {
					viol = append(viol, Violation{ri, rule.Column, rowi, "duplicate value"})
				}
			}
		default:
			return Report{}, fmt.Errorf("rule %d: unknown kind %q", ri, rule.Kind)
		}
	}
	sort.SliceStable(viol, func(i, j int) bool {
		if viol[i].Row != viol[j].Row {
			return viol[i].Row < viol[j].Row
		}
		return viol[i].Rule < viol[j].Rule
	})
	totalChecks := len(t.Rows) * len(rules)
	rep.TotalRows = len(t.Rows)
	rep.TotalChecks = totalChecks
	rep.Violations = viol
	for _, v := range viol {
		rep.ByColumn[v.Column]++
	}
	if totalChecks == 0 {
		rep.Score = 1
	} else {
		rep.Score = 1 - float64(len(viol))/float64(totalChecks)
	}
	return rep, nil
}
