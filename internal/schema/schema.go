package schema

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"dq-pipeline/internal/parse"
)

type ColType string

const (
	TypeString  ColType = "string"
	TypeInt     ColType = "int"
	TypeFloat   ColType = "float"
	TypeBool    ColType = "bool"
	TypeDate    ColType = "date"
	TypeUnknown ColType = "unknown"
)

type Column struct {
	Name     string  `json:"name"`
	Type     ColType `json:"type"`
	Nullable bool    `json:"nullable"`
	MaxLen   int     `json:"max_len,omitempty"`
	MinLen   int     `json:"min_len,omitempty"`
}

type Schema struct {
	Columns []Column `json:"columns"`
	Version int      `json:"version"`
}

type InferOptions struct {
	DateLayouts []string
	SampleSize  int
}

var DefaultDateLayouts = []string{
	"2006-01-02",
	"2006-01-02T15:04:05Z07:00",
	"02/01/2006",
	"01-02-2006",
	"2006/01/02",
}

func Infer(t *parse.Table, opts *InferOptions) *Schema {
	if err := abortInferContext(); err != nil {
		return &Schema{Version: 1}
	}
	if opts == nil {
		opts = &InferOptions{}
	}
	if len(opts.DateLayouts) == 0 {
		opts.DateLayouts = DefaultDateLayouts
	}

	rows := t.Rows
	if opts.SampleSize > 0 && len(rows) > opts.SampleSize {
		rows = rows[:opts.SampleSize]
	}

	schema := &Schema{Version: 1}
	for ci, name := range t.Header {
		col := inferColumn(name, ci, rows, opts)
		schema.Columns = append(schema.Columns, col)
	}
	return schema
}

func inferColumn(name string, ci int, rows [][]string, opts *InferOptions) Column {
	col := Column{Name: name, Type: TypeUnknown}
	if len(rows) == 0 {
		col.Type = TypeString
		return col
	}

	var hasEmpty bool
	var minLen, maxLen int
	minLen = -1

	intCount, floatCount, boolCount, dateCount, total := 0, 0, 0, 0, 0

	for _, row := range rows {
		val := ""
		if ci < len(row) {
			val = row[ci]
		}
		if val == "" {
			hasEmpty = true
			continue
		}
		total++
		l := len(val)
		if minLen < 0 || l < minLen {
			minLen = l
		}
		if l > maxLen {
			maxLen = l
		}

		if isInt(val) {
			intCount++
			floatCount++
		} else if isFloat(val) {
			floatCount++
		}
		if isBool(val) {
			boolCount++
		}
		if isDate(val, opts.DateLayouts) {
			dateCount++
		}
	}

	col.Nullable = hasEmpty
	col.MinLen = minLen
	if minLen < 0 {
		col.MinLen = 0
	}
	col.MaxLen = maxLen

	if total == 0 {
		col.Type = TypeString
		return col
	}

	threshold := total
	switch {
	case intCount == threshold:
		col.Type = TypeInt
	case floatCount == threshold:
		col.Type = TypeFloat
	case boolCount == threshold:
		col.Type = TypeBool
	case dateCount == threshold:
		col.Type = TypeDate
	default:
		col.Type = TypeString
	}
	return col
}

func isInt(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

func isFloat(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func isBool(s string) bool {
	lower := strings.ToLower(s)
	return lower == "true" || lower == "false" || lower == "0" || lower == "1"
}

func isDate(s string, layouts []string) bool {
	for _, layout := range layouts {
		if _, err := time.Parse(layout, s); err == nil {
			return true
		}
	}
	return false
}

func Validate(t *parse.Table, schema *Schema) []SchemaViolation {
	var viols []SchemaViolation
	for ci, col := range schema.Columns {
		if ci >= len(t.Header) {
			viols = append(viols, SchemaViolation{
				Column: col.Name,
				Row:    -1,
				Reason: "column missing from table",
			})
			continue
		}
		for ri, row := range t.Rows {
			val := ""
			if ci < len(row) {
				val = row[ci]
			}
			if val == "" {
				if !col.Nullable {
					viols = append(viols, SchemaViolation{
						Column: col.Name,
						Row:    ri,
						Reason: "null value in non-nullable column",
					})
				}
				continue
			}
			if reason := checkType(val, col); reason != "" {
				viols = append(viols, SchemaViolation{
					Column: col.Name,
					Row:    ri,
					Reason: reason,
				})
			}
		}
	}
	return viols
}

func checkType(val string, col Column) string {
	switch col.Type {
	case TypeInt:
		if !isInt(val) {
			return fmt.Sprintf("expected int, got %q", val)
		}
	case TypeFloat:
		if !isFloat(val) {
			return fmt.Sprintf("expected float, got %q", val)
		}
	case TypeBool:
		if !isBool(val) {
			return fmt.Sprintf("expected bool, got %q", val)
		}
	case TypeDate:
		if !isDate(val, DefaultDateLayouts) {
			return fmt.Sprintf("expected date, got %q", val)
		}
	}
	return ""
}

type SchemaViolation struct {
	Column string
	Row    int
	Reason string
}

func (s *Schema) ColumnByName(name string) *Column {
	for i := range s.Columns {
		if s.Columns[i].Name == name {
			return &s.Columns[i]
		}
	}
	return nil
}

func (s *Schema) ColumnNames() []string {
	names := make([]string, len(s.Columns))
	for i, c := range s.Columns {
		names[i] = c.Name
	}
	return names
}
