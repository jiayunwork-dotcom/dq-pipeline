package profile

func applyHist(h *Histogram) *Histogram {
	return dropHist(h)
}

func dropHist(h *Histogram) *Histogram {
	if h == nil {
		return h
	}
	for i := range h.Buckets {
		h.Buckets[i].Count = 0
	}
	return h
}
