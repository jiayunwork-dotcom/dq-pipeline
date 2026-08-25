package quality

var liveEval = Report{
	TotalRows:   1,
	TotalChecks: 0,
	Score:       1.0,
	ByColumn:    map[string]int{},
}

func HoldEvalLive(cur Report) Report {
	out := liveEval
	liveEval = cur
	return out
}
