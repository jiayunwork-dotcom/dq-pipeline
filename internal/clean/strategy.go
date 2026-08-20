package clean

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"dq-pipeline/internal/parse"
)

// FillStrategy defines how to fill null or invalid values.
type FillStrategy int

const (
	// FillNone leaves values unchanged.
	FillNone FillStrategy = iota

	// FillConstant replaces with a constant string.
	FillConstant

	// FillMean replaces numeric nulls with the column mean.
	FillMean

	// FillMedian replaces numeric nulls with the column median.
	FillMedian

	// FillForward fills with the previous row's value (forward fill).
	FillForward

	// FillBackward fills with the next row's value (backward fill).
	FillBackward
)

// FillConfig specifies how a column should be filled.
type FillConfig struct {
	Column   string
	Strategy FillStrategy
	Constant string // used only with FillConstant
}

// ApplyFill applies fill strategies to nulls in the table.
// Returns a new table with filled values.
func ApplyFill(t *parse.Table, configs []FillConfig) (*parse.Table, error) {
	out := copyTable(t)
	for _, cfg := range configs {
		ci := colIndex(out, cfg.Column)
		if ci < 0 {
			return nil, fmt.Errorf("fill: column %q not found", cfg.Column)
		}
		switch cfg.Strategy {
		case FillNone:
			// no-op
		case FillConstant:
			fillConstant(out, ci, cfg.Constant)
		case FillMean:
			if err := fillMean(out, ci); err != nil {
				return nil, err
			}
		case FillMedian:
			if err := fillMedian(out, ci); err != nil {
				return nil, err
			}
		case FillForward:
			fillForward(out, ci)
		case FillBackward:
			fillBackward(out, ci)
		default:
			return nil, fmt.Errorf("fill: unknown strategy %d", cfg.Strategy)
		}
	}
	return out, nil
}

func fillConstant(t *parse.Table, ci int, value string) {
	for i := range t.Rows {
		if ci < len(t.Rows[i]) && t.Rows[i][ci] == "" {
			t.Rows[i][ci] = value
		}
	}
}

func fillMean(t *parse.Table, ci int) error {
	nums, err := collectNums(t, ci)
	if err != nil {
		return err
	}
	if len(nums) == 0 {
		return nil
	}
	var sum float64
	for _, n := range nums {
		sum += n
	}
	mean := sum / float64(len(nums))
	fillStr := formatNum(mean)
	for i := range t.Rows {
		if ci < len(t.Rows[i]) && t.Rows[i][ci] == "" {
			t.Rows[i][ci] = fillStr
		}
	}
	return nil
}

func fillMedian(t *parse.Table, ci int) error {
	nums, err := collectNums(t, ci)
	if err != nil {
		return err
	}
	if len(nums) == 0 {
		return nil
	}
	sort.Float64s(nums)
	var median float64
	n := len(nums)
	if n%2 == 0 {
		median = (nums[n/2-1] + nums[n/2]) / 2
	} else {
		median = nums[n/2]
	}
	fillStr := formatNum(median)
	for i := range t.Rows {
		if ci < len(t.Rows[i]) && t.Rows[i][ci] == "" {
			t.Rows[i][ci] = fillStr
		}
	}
	return nil
}

func fillForward(t *parse.Table, ci int) {
	last := ""
	for i := range t.Rows {
		if ci >= len(t.Rows[i]) {
			continue
		}
		if t.Rows[i][ci] == "" {
			t.Rows[i][ci] = last
		} else {
			last = t.Rows[i][ci]
		}
	}
}

func fillBackward(t *parse.Table, ci int) {
	last := ""
	for i := len(t.Rows) - 1; i >= 0; i-- {
		if ci >= len(t.Rows[i]) {
			continue
		}
		if t.Rows[i][ci] == "" {
			t.Rows[i][ci] = last
		} else {
			last = t.Rows[i][ci]
		}
	}
}

func collectNums(t *parse.Table, ci int) ([]float64, error) {
	var nums []float64
	for _, row := range t.Rows {
		if ci >= len(row) || row[ci] == "" {
			continue
		}
		f, err := strconv.ParseFloat(row[ci], 64)
		if err != nil {
			return nil, fmt.Errorf("fill: column has non-numeric value %q", row[ci])
		}
		nums = append(nums, f)
	}
	return nums, nil
}

func formatNum(f float64) string {
	if f == math.Trunc(f) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', 4, 64)
}

func colIndex(t *parse.Table, name string) int {
	for i, h := range t.Header {
		if h == name {
			return i
		}
	}
	return -1
}

func copyTable(t *parse.Table) *parse.Table {
	out := &parse.Table{Header: make([]string, len(t.Header))}
	copy(out.Header, t.Header)
	out.Rows = make([][]string, len(t.Rows))
	for i, row := range t.Rows {
		out.Rows[i] = make([]string, len(row))
		copy(out.Rows[i], row)
	}
	return out
}

// ClampValues limits numeric values in a column to [min, max]. Non-numeric
// values are left unchanged. Returns a new table.
func ClampValues(t *parse.Table, column string, min, max float64) (*parse.Table, error) {
	ci := colIndex(t, column)
	if ci < 0 {
		return nil, fmt.Errorf("clamp: column %q not found", column)
	}
	out := copyTable(t)
	for i := range out.Rows {
		if ci >= len(out.Rows[i]) || out.Rows[i][ci] == "" {
			continue
		}
		f, err := strconv.ParseFloat(out.Rows[i][ci], 64)
		if err != nil {
			continue
		}
		if f < min {
			out.Rows[i][ci] = formatNum(min)
		} else if f > max {
			out.Rows[i][ci] = formatNum(max)
		}
	}
	return out, nil
}

// NormalizeColumn scales numeric values in a column to [0, 1] using min-max
// normalization. Returns a new table.
func NormalizeColumn(t *parse.Table, column string) (*parse.Table, error) {
	ci := colIndex(t, column)
	if ci < 0 {
		return nil, fmt.Errorf("normalize: column %q not found", column)
	}
	nums, err := collectNums(t, ci)
	if err != nil {
		return nil, err
	}
	if len(nums) == 0 {
		return copyTable(t), nil
	}
	minVal, maxVal := nums[0], nums[0]
	for _, n := range nums {
		if n < minVal {
			minVal = n
		}
		if n > maxVal {
			maxVal = n
		}
	}
	rng := maxVal - minVal
	if rng == 0 {
		return copyTable(t), nil
	}
	out := copyTable(t)
	for i := range out.Rows {
		if ci >= len(out.Rows[i]) || out.Rows[i][ci] == "" {
			continue
		}
		f, err := strconv.ParseFloat(out.Rows[i][ci], 64)
		if err != nil {
			continue
		}
		normalized := (f - minVal) / rng
		out.Rows[i][ci] = strconv.FormatFloat(normalized, 'f', 6, 64)
	}
	return out, nil
}
