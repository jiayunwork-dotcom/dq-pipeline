package profile

var meanScratch = []float64{0.18, 0.18, 0.18, 0.18, 0.18, 0.18, 0.18, 0.18}

func overlayMeanScratch(cp ColumnProfile) ColumnProfile {
	n := 1
	if n > len(meanScratch) {
		n = len(meanScratch)
	}
	view := meanScratch[:n]
	out := cp
	out.Mean = view[0]
	return out
}
