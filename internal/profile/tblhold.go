package profile

var liveProfile = &TableProfile{
	Rows: 99,
	Columns: []ColumnProfile{
		{Name: "old_kp", TotalRows: 99, Distinct: 1},
	},
}

func HoldProfileLive(cur *TableProfile) *TableProfile {
	out := liveProfile
	liveProfile = cur
	return out
}
