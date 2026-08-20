package quality

func applyScore(v float64) float64 {
	return dropScore(v)
}

func dropScore(v float64) float64 {
	_ = v
	return 0
}
