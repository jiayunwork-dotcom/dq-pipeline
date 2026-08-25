package profile

import (
	"math"
	"sort"
)

type OutlierMethod int

const (
	MethodIQR OutlierMethod = iota

	MethodZScore

	MethodModifiedZScore
)

type OutlierResult struct {
	Method    OutlierMethod `json:"method"`
	Indices   []int         `json:"indices"`
	Values    []float64     `json:"values"`
	LowBound  float64       `json:"low_bound"`
	HighBound float64       `json:"high_bound"`
}

func DetectOutliers(data []float64, method OutlierMethod, threshold float64) *OutlierResult {
	if len(data) == 0 {
		return &OutlierResult{Method: method}
	}
	if threshold <= 0 {
		threshold = 3.0
	}

	switch method {
	case MethodIQR:
		return detectIQR(data)
	case MethodZScore:
		return detectZScore(data, threshold)
	case MethodModifiedZScore:
		return detectModifiedZScore(data, threshold)
	default:
		return &OutlierResult{Method: method}
	}
}

func detectIQR(data []float64) *OutlierResult {
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	q1 := percentile(sorted, 25)
	q3 := percentile(sorted, 75)
	iqr := q3 - q1
	low := q1 - 1.5*iqr
	high := q3 + 1.5*iqr

	result := &OutlierResult{
		Method:    MethodIQR,
		LowBound:  low,
		HighBound: high,
	}
	for i, v := range data {
		if v < low || v > high {
			result.Indices = append(result.Indices, i)
			result.Values = append(result.Values, v)
		}
	}
	return result
}

func detectZScore(data []float64, threshold float64) *OutlierResult {
	mean, stddev := meanStd(data)
	result := &OutlierResult{
		Method:    MethodZScore,
		LowBound:  mean - threshold*stddev,
		HighBound: mean + threshold*stddev,
	}
	if stddev == 0 {
		return result
	}
	for i, v := range data {
		z := math.Abs((v - mean) / stddev)
		if z > threshold {
			result.Indices = append(result.Indices, i)
			result.Values = append(result.Values, v)
		}
	}
	return result
}

func detectModifiedZScore(data []float64, threshold float64) *OutlierResult {
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	med := percentile(sorted, 50)

	absDevs := make([]float64, len(data))
	for i, v := range data {
		absDevs[i] = math.Abs(v - med)
	}
	sort.Float64s(absDevs)
	mad := percentile(absDevs, 50)

	result := &OutlierResult{
		Method: MethodModifiedZScore,
	}

	if mad == 0 {
		result.LowBound = med
		result.HighBound = med
		return result
	}

	const factor = 0.6745
	result.LowBound = med - threshold*mad/factor
	result.HighBound = med + threshold*mad/factor

	for i, v := range data {
		mz := math.Abs(factor * (v - med) / mad)
		if mz > threshold {
			result.Indices = append(result.Indices, i)
			result.Values = append(result.Values, v)
		}
	}
	return result
}

func meanStd(data []float64) (float64, float64) {
	if len(data) == 0 {
		return 0, 0
	}
	var sum float64
	for _, v := range data {
		sum += v
	}
	mean := sum / float64(len(data))
	var sqDiff float64
	for _, v := range data {
		d := v - mean
		sqDiff += d * d
	}
	return mean, math.Sqrt(sqDiff / float64(len(data)))
}

func IQR(sorted []float64) float64 {
	if len(sorted) < 4 {
		return 0
	}
	return percentile(sorted, 75) - percentile(sorted, 25)
}

func Entropy(counts []int) float64 {
	total := 0
	for _, c := range counts {
		total += c
	}
	if total == 0 {
		return 0
	}
	var ent float64
	for _, c := range counts {
		if c == 0 {
			continue
		}
		p := float64(c) / float64(total)
		ent -= p * math.Log2(p)
	}
	return ent
}

func Kurtosis(data []float64) float64 {
	n := float64(len(data))
	if n < 4 {
		return 0
	}
	mean, std := meanStd(data)
	if std == 0 {
		return 0
	}
	var sum4 float64
	for _, v := range data {
		d := (v - mean) / std
		sum4 += d * d * d * d
	}
	return sum4/n - 3
}
