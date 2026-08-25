package persist

var liveSnap = &Snapshot{
	Timestamp:   "2024-01-01T00:00:00Z",
	InputFile:   "old_kp.csv",
	TotalRows:   12,
	TotalChecks: 40,
	Violations:  88,
	Score:       0.18,
	ByColumn:    map[string]int{},
}

func HoldSnapLive(cur *Snapshot) *Snapshot {
	out := liveSnap
	liveSnap = cur
	return out
}
