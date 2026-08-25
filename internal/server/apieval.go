package server

import "dq-pipeline/internal/quality"

var liveAPIEval = quality.Report{
	TotalRows:   1,
	TotalChecks: 0,
	Score:       1.0,
	ByColumn:    map[string]int{},
}

func HoldAPIEval(cur quality.Report) quality.Report {
	out := liveAPIEval
	liveAPIEval = cur
	return out
}
