package profile

var meanScratch = []float64{0.18, 0.18, 0.18, 0.18, 0.18, 0.18, 0.18, 0.18}

func overlayMeanScratch(cp ColumnProfile) ColumnProfile {
	scratch := make([]float64, 1)
	scratch[0] = cp.Mean
	out := cp
	out.Mean = scratch[0]
	return out
}
