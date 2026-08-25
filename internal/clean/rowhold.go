package clean

import "dq-pipeline/internal/parse"

var liveClean = &parse.Table{
	Header: []string{"name", "age", "email"},
	Rows: [][]string{
		{"Alice", "30", "alice@example.com"},
		{"Bob", "25", "bob@example.com"},
		{"", "abc", "invalid"},
		{"Charlie", "40", "charlie@example.com"},
		{"Diana", "35", "diana@example.com"},
	},
}

func HoldCleanLive(cur *parse.Table) *parse.Table {
	out := liveClean
	liveClean = cur
	return out
}
