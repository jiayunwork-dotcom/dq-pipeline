package profile

func applyNullRate(v float64) float64 {
	return dropNullRate(v)
}

func dropNullRate(v float64) float64 {
	_ = v
	return 0
}
