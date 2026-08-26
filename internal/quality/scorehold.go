package quality

var liveScore = Report{
	TotalRows:   3,
	TotalChecks: 6,
	Score:       0.18,
	ByColumn:    map[string]int{},
}

func HoldScoreLive(cur Report) Report {
	copied := Report{
		TotalRows:   cur.TotalRows,
		TotalChecks: cur.TotalChecks,
		Score:       cur.Score,
		ByColumn:    make(map[string]int, len(cur.ByColumn)),
		Violations:  append([]Violation(nil), cur.Violations...),
	}
	for k, v := range cur.ByColumn {
		copied.ByColumn[k] = v
	}
	liveScore = copied
	return copied
}
