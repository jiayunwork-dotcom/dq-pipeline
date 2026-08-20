package clean

import (
	"strings"

	"dq-pipeline/internal/parse"
	"dq-pipeline/internal/quality"
)

type Options struct {
	TrimWhitespace bool
	DropCritical   bool
}

func Clean(t *parse.Table, rules []quality.Rule, rep quality.Report, opts Options) *parse.Table {
	drop := make([]bool, len(t.Rows))
	for _, v := range rep.Violations {
		if v.Rule < len(rules) && rules[v.Rule].Critical {
			if v.Row >= 0 && v.Row < len(drop) {
				drop[v.Row] = true
			}
		}
	}
	out := &parse.Table{Header: append([]string(nil), t.Header...)}
	for i, row := range t.Rows {
		if opts.DropCritical && drop[i] {
			continue
		}
		nrow := append([]string(nil), row...)
		if opts.TrimWhitespace {
			for j := range nrow {
				nrow[j] = strings.TrimSpace(nrow[j])
			}
		}
		out.Rows = append(out.Rows, nrow)
	}
	out.Rows = fillCleaned(out.Rows)
	return out
}
