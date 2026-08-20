package profile

import (
	"fmt"
	"math"
	"strings"
)

// Bucket represents a single histogram bin.
type Bucket struct {
	Low      float64 `json:"low"`
	High     float64 `json:"high"`
	Count    int     `json:"count"`
	Midpoint float64 `json:"midpoint"`
}

// Histogram holds a set of equal-width buckets spanning [min, max].
type Histogram struct {
	Buckets  []Bucket `json:"buckets"`
	Total    int      `json:"total"`
	Min      float64  `json:"min"`
	Max      float64  `json:"max"`
	Width    float64  `json:"width"`
	Overflow int      `json:"overflow"`
}

// BuildHistogram creates a histogram with numBuckets equal-width bins from the
// given data. Values exactly equal to Max fall into the last bucket. numBuckets
// must be > 0.
func BuildHistogram(data []float64, numBuckets int) (*Histogram, error) {
	if numBuckets <= 0 {
		return nil, fmt.Errorf("histogram: numBuckets must be > 0")
	}
	if len(data) == 0 {
		return &Histogram{Buckets: make([]Bucket, numBuckets)}, nil
	}

	min, max := data[0], data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// handle single-value case
	if min == max {
		buckets := make([]Bucket, numBuckets)
		buckets[0] = Bucket{Low: min, High: max, Count: len(data), Midpoint: min}
		for i := 1; i < numBuckets; i++ {
			buckets[i] = Bucket{Low: max, High: max}
		}
		return &Histogram{
			Buckets: buckets,
			Total:   len(data),
			Min:     min,
			Max:     max,
			Width:   0,
		}, nil
	}

	width := (max - min) / float64(numBuckets)
	buckets := make([]Bucket, numBuckets)
	for i := range buckets {
		low := min + float64(i)*width
		high := min + float64(i+1)*width
		buckets[i] = Bucket{
			Low:      low,
			High:     high,
			Midpoint: (low + high) / 2,
		}
	}

	for _, v := range data {
		idx := int(math.Floor((v - min) / width))
		if idx >= numBuckets {
			idx = numBuckets - 1 // max value goes to last bucket
		}
		if idx < 0 {
			idx = 0
		}
		buckets[idx].Count++
	}

	return &Histogram{
		Buckets: buckets,
		Total:   len(data),
		Min:     min,
		Max:     max,
		Width:   width,
	}, nil
}

// BucketFor returns the bucket index that contains the given value, or -1
// if the value is outside the histogram range.
func (h *Histogram) BucketFor(value float64) int {
	if h.Width == 0 {
		if value == h.Min {
			return 0
		}
		return -1
	}
	idx := int(math.Floor((value - h.Min) / h.Width))
	if idx < 0 || idx >= len(h.Buckets) {
		if value == h.Max {
			return len(h.Buckets) - 1
		}
		return -1
	}
	return idx
}

// CumulativeCounts returns a slice where each element is the sum of counts
// up to and including that bucket.
func (h *Histogram) CumulativeCounts() []int {
	cum := make([]int, len(h.Buckets))
	sum := 0
	for i, b := range h.Buckets {
		sum += b.Count
		cum[i] = sum
	}
	return cum
}

// Density returns the probability density for each bucket (count / total / width).
func (h *Histogram) Density() []float64 {
	dens := make([]float64, len(h.Buckets))
	if h.Total == 0 || h.Width == 0 {
		return dens
	}
	for i, b := range h.Buckets {
		dens[i] = float64(b.Count) / float64(h.Total) / h.Width
	}
	return dens
}

// Mode returns the midpoint of the bucket with the highest count.
func (h *Histogram) Mode() float64 {
	maxCount := 0
	modeIdx := 0
	for i, b := range h.Buckets {
		if b.Count > maxCount {
			maxCount = b.Count
			modeIdx = i
		}
	}
	if len(h.Buckets) == 0 {
		return 0
	}
	return h.Buckets[modeIdx].Midpoint
}

// Skewness estimates the distribution skewness using Pearson's second coefficient:
// 3 * (mean - median) / stddev. Requires external mean, median, stddev.
func Skewness(mean, median, stddev float64) float64 {
	if stddev == 0 {
		return 0
	}
	return 3 * (mean - median) / stddev
}

// String returns a simple ASCII representation of the histogram.
func (h *Histogram) String() string {
	if len(h.Buckets) == 0 {
		return "(empty histogram)"
	}
	maxCount := 0
	for _, b := range h.Buckets {
		if b.Count > maxCount {
			maxCount = b.Count
		}
	}
	var sb strings.Builder
	barWidth := 40
	for _, b := range h.Buckets {
		bar := 0
		if maxCount > 0 {
			bar = b.Count * barWidth / maxCount
		}
		fmt.Fprintf(&sb, "[%8.2f, %8.2f) %s %d\n",
			b.Low, b.High, strings.Repeat("#", bar), b.Count)
	}
	return sb.String()
}
