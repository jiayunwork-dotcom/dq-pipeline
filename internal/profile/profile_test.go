package profile

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"dq-pipeline/internal/parse"
)

func tbl(csv string) *parse.Table {
	t, _ := parse.Parse(strings.NewReader(csv), ',')
	return t
}

func TestProfileTableNumeric(t *testing.T) {
	data := tbl("score\n10\n20\n30\n40\n50\n")
	tp := ProfileTable(data)
	if len(tp.Columns) != 1 {
		t.Fatalf("columns = %d", len(tp.Columns))
	}
	cp := tp.Columns[0]
	if !cp.IsNumeric {
		t.Fatal("expected numeric column")
	}
	if cp.Min != 10 || cp.Max != 50 {
		t.Errorf("min=%f max=%f", cp.Min, cp.Max)
	}
	if math.Abs(cp.Mean-30) > 0.001 {
		t.Errorf("mean = %f, want 30", cp.Mean)
	}
	if cp.Median != 30 {
		t.Errorf("median = %f, want 30", cp.Median)
	}
}

func TestProfileTableNullRate(t *testing.T) {
	data := &parse.Table{
		Header: []string{"val"},
		Rows:   [][]string{{"1"}, {""}, {"3"}, {""}, {"5"}},
	}
	tp := ProfileTable(data)
	cp := tp.Columns[0]
	if cp.NullCount != 2 {
		t.Errorf("null_count = %d, want 2", cp.NullCount)
	}
	if math.Abs(cp.NullRate-0.4) > 0.001 {
		t.Errorf("null_rate = %f, want 0.4", cp.NullRate)
	}
}

func TestProfileTableStringColumn(t *testing.T) {
	data := tbl("name\nAlice\nBob\nAlice\nCarol\n")
	tp := ProfileTable(data)
	cp := tp.Columns[0]
	if cp.IsNumeric {
		t.Error("name should not be numeric")
	}
	if cp.Distinct != 3 {
		t.Errorf("distinct = %d, want 3", cp.Distinct)
	}
}

func TestProfileTableTopValues(t *testing.T) {
	data := tbl("status\nactive\nactive\ninactive\nactive\npending\n")
	tp := ProfileTable(data)
	cp := tp.Columns[0]
	if len(cp.TopValues) == 0 {
		t.Fatal("expected top values")
	}
	if cp.TopValues[0].Value != "active" || cp.TopValues[0].Count != 3 {
		t.Errorf("top[0] = %v", cp.TopValues[0])
	}
}

func TestProfileColumnNotFound(t *testing.T) {
	data := tbl("a\n1\n")
	_, err := ProfileColumn(data, "missing")
	if err == nil {
		t.Error("expected error for missing column")
	}
}

func TestProfileColumnPercentiles(t *testing.T) {
	// 100 values: 1..100
	var rows []string
	for i := 1; i <= 100; i++ {
		rows = append(rows, strconv.Itoa(i))
	}
	csv := "val\n" + strings.Join(rows, "\n") + "\n"
	data := tbl(csv)
	cp, err := ProfileColumn(data, "val")
	if err != nil {
		t.Fatal(err)
	}
	// P25 ~ 25.75, P75 ~ 75.25
	if cp.P25 < 24 || cp.P25 > 27 {
		t.Errorf("P25 = %f", cp.P25)
	}
	if cp.P75 < 74 || cp.P75 > 77 {
		t.Errorf("P75 = %f", cp.P75)
	}
}

func TestBuildHistogram(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	h, err := BuildHistogram(data, 5)
	if err != nil {
		t.Fatal(err)
	}
	if h.Total != 10 {
		t.Errorf("total = %d", h.Total)
	}
	if len(h.Buckets) != 5 {
		t.Errorf("buckets = %d", len(h.Buckets))
	}
	// each bucket should have 2 items
	for _, b := range h.Buckets {
		if b.Count != 2 {
			t.Errorf("bucket [%.1f, %.1f) count = %d, want 2", b.Low, b.High, b.Count)
		}
	}
}

func TestBuildHistogramSingleValue(t *testing.T) {
	data := []float64{5, 5, 5}
	h, err := BuildHistogram(data, 3)
	if err != nil {
		t.Fatal(err)
	}
	if h.Total != 3 {
		t.Errorf("total = %d", h.Total)
	}
	if h.Buckets[0].Count != 3 {
		t.Errorf("first bucket = %d, want 3", h.Buckets[0].Count)
	}
}

func TestBuildHistogramInvalidBuckets(t *testing.T) {
	_, err := BuildHistogram([]float64{1, 2}, 0)
	if err == nil {
		t.Error("expected error for 0 buckets")
	}
}

func TestHistogramBucketFor(t *testing.T) {
	data := []float64{0, 10, 20, 30, 40}
	h, _ := BuildHistogram(data, 4)
	if idx := h.BucketFor(15); idx != 1 {
		t.Errorf("BucketFor(15) = %d", idx)
	}
	if idx := h.BucketFor(-5); idx != -1 {
		t.Errorf("BucketFor(-5) = %d, want -1", idx)
	}
}

func TestHistogramCumulativeCounts(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6}
	h, _ := BuildHistogram(data, 3)
	cum := h.CumulativeCounts()
	if cum[len(cum)-1] != 6 {
		t.Errorf("last cumulative = %d, want 6", cum[len(cum)-1])
	}
}

func TestHistogramMode(t *testing.T) {
	// 1,1,1,5,5,9 -> bucket with 1s should be mode
	data := []float64{1, 1, 1, 5, 5, 9}
	h, _ := BuildHistogram(data, 3)
	mode := h.Mode()
	// first bucket contains 1,1,1 -> midpoint should be low
	if mode > 5 {
		t.Errorf("mode = %f, expected in first bucket range", mode)
	}
}

func TestDetectOutliersIQR(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 100}
	result := DetectOutliers(data, MethodIQR, 0)
	if len(result.Indices) == 0 {
		t.Error("expected 100 to be an outlier")
	}
}

func TestDetectOutliersZScore(t *testing.T) {
	data := []float64{10, 10, 10, 10, 10, 10, 10, 10, 10, 1000}
	result := DetectOutliers(data, MethodZScore, 2.0)
	if len(result.Indices) != 1 {
		t.Errorf("outliers = %d, want 1", len(result.Indices))
	}
}

func TestDetectOutliersEmpty(t *testing.T) {
	result := DetectOutliers(nil, MethodIQR, 0)
	if len(result.Indices) != 0 {
		t.Error("expected no outliers for empty data")
	}
}

func TestEntropy(t *testing.T) {
	// uniform: 2 categories, 50 each -> entropy = 1 bit
	counts := []int{50, 50}
	e := Entropy(counts)
	if math.Abs(e-1.0) > 0.001 {
		t.Errorf("entropy = %f, want 1.0", e)
	}
}

func TestKurtosis(t *testing.T) {
	// normal-ish data should have kurtosis near 0
	data := []float64{-2, -1, -1, 0, 0, 0, 0, 1, 1, 2}
	k := Kurtosis(data)
	if math.Abs(k) > 2 {
		t.Errorf("kurtosis = %f, expected near 0", k)
	}
}
