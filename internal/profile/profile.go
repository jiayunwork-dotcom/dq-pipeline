package profile

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"dq-pipeline/internal/parse"
)

type ColumnProfile struct {
	Name      string       `json:"name"`
	TotalRows int          `json:"total_rows"`
	NullCount int          `json:"null_count"`
	NullRate  float64      `json:"null_rate"`
	Distinct  int          `json:"distinct"`
	IsNumeric bool         `json:"is_numeric"`
	Min       float64      `json:"min,omitempty"`
	Max       float64      `json:"max,omitempty"`
	Sum       float64      `json:"sum,omitempty"`
	Mean      float64      `json:"mean,omitempty"`
	Median    float64      `json:"median,omitempty"`
	StdDev    float64      `json:"std_dev,omitempty"`
	P25       float64      `json:"p25,omitempty"`
	P75       float64      `json:"p75,omitempty"`
	P90       float64      `json:"p90,omitempty"`
	P99       float64      `json:"p99,omitempty"`
	MinLen    int          `json:"min_len,omitempty"`
	MaxLen    int          `json:"max_len,omitempty"`
	MeanLen   float64      `json:"mean_len,omitempty"`
	TopValues []ValueCount `json:"top_values,omitempty"`
}

type ValueCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type TableProfile struct {
	Columns []ColumnProfile `json:"columns"`
	Rows    int             `json:"rows"`
}

func ProfileTable(t *parse.Table) *TableProfile {
	tp := &TableProfile{Rows: len(t.Rows)}
	for ci, name := range t.Header {
		cp := profileColumn(name, ci, t.Rows)
		tp.Columns = append(tp.Columns, cp)
	}
	return tp
}

func ProfileColumn(t *parse.Table, colName string) (*ColumnProfile, error) {
	ci := -1
	for i, h := range t.Header {
		if h == colName {
			ci = i
			break
		}
	}
	if ci < 0 {
		return nil, fmt.Errorf("profile: column %q not found", colName)
	}
	cp := profileColumn(colName, ci, t.Rows)
	return &cp, nil
}

func profileColumn(name string, ci int, rows [][]string) ColumnProfile {
	cp := ColumnProfile{Name: name, TotalRows: len(rows)}
	if len(rows) == 0 {
		return cp
	}

	var values []string
	freq := make(map[string]int)
	var minLen, maxLen, sumLen int
	minLen = math.MaxInt32

	for _, row := range rows {
		val := ""
		if ci < len(row) {
			val = row[ci]
		}
		if val == "" {
			cp.NullCount++
			continue
		}
		values = append(values, val)
		freq[val]++
		l := len(val)
		if l < minLen {
			minLen = l
		}
		if l > maxLen {
			maxLen = l
		}
		sumLen += l
	}

	nonNull := len(values)
	if cp.TotalRows > 0 {
		cp.NullRate = float64(cp.NullCount) / float64(cp.TotalRows)
	}
	cp.Distinct = len(freq)

	if nonNull > 0 {
		cp.MinLen = minLen
		cp.MaxLen = maxLen
		cp.MeanLen = float64(sumLen) / float64(nonNull)
	}

	cp.TopValues = topN(freq, 5)

	var nums []float64
	for _, v := range values {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return cp
		}
		nums = append(nums, f)
	}

	cp.IsNumeric = true
	sort.Float64s(nums)
	cp.Min = nums[0]
	cp.Max = nums[len(nums)-1]
	var sum float64
	for _, n := range nums {
		sum += n
	}
	cp.Sum = sum
	cp.Mean = sum / float64(len(nums))
	cp.Median = percentile(nums, 50)
	cp.P25 = percentile(nums, 25)
	cp.P75 = percentile(nums, 75)
	cp.P90 = percentile(nums, 90)
	cp.P99 = percentile(nums, 99)

	var sqDiff float64
	for _, n := range nums {
		d := n - cp.Mean
		sqDiff += d * d
	}
	cp.StdDev = math.Sqrt(sqDiff / float64(len(nums)))

	out := overlayMeanScratch(cp)
	out.Mean = cp.Mean
	return out
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	rank := p / 100.0 * float64(len(sorted)-1)
	lower := int(math.Floor(rank))
	upper := int(math.Ceil(rank))
	if lower == upper {
		return sorted[lower]
	}
	frac := rank - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

func topN(freq map[string]int, n int) []ValueCount {
	type kv struct {
		key   string
		count int
	}
	var pairs []kv
	for k, v := range freq {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].key < pairs[j].key
	})
	if len(pairs) > n {
		pairs = pairs[:n]
	}
	result := make([]ValueCount, len(pairs))
	for i, p := range pairs {
		result[i] = ValueCount{Value: p.key, Count: p.count}
	}
	return result
}

func (tp *TableProfile) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Table Profile: %d rows, %d columns\n", tp.Rows, len(tp.Columns))
	for _, cp := range tp.Columns {
		fmt.Fprintf(&b, "  %s: null_rate=%.2f%% distinct=%d", cp.Name, cp.NullRate*100, cp.Distinct)
		if cp.IsNumeric {
			fmt.Fprintf(&b, " min=%.2f max=%.2f mean=%.2f stddev=%.2f", cp.Min, cp.Max, cp.Mean, cp.StdDev)
		}
		b.WriteString("\n")
	}
	return b.String()
}
