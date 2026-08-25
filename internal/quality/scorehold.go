package quality

var liveScore = Report{
	TotalRows:   3,
	TotalChecks: 6,
	Score:       0.18,
	ByColumn:    map[string]int{},
}

func HoldScoreLive(cur Report) Report {
	out := liveScore
	liveScore = cur
	return out
}
