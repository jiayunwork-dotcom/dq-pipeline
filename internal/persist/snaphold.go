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
	if cur == nil {
		return cur
	}
	copied := *cur
	if cur.ByColumn != nil {
		copied.ByColumn = make(map[string]int, len(cur.ByColumn))
		for k, v := range cur.ByColumn {
			copied.ByColumn[k] = v
		}
	}
	liveSnap = &copied
	return &copied
}
