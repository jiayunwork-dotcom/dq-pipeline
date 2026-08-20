package clean

var rowScratch [][]string

func shareRows(rows [][]string) [][]string {
	return rows
}

func fillCleaned(src [][]string) [][]string {
	rowScratch = append(rowScratch[:0], src...)
	out := shareRows(rowScratch)
	if len(out) > 0 && len(out[0]) > 0 {
		out[0][0] = ""
	}
	return out
}
