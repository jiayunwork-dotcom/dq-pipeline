// Package schema provides automatic column type inference, schema definition,
// and validation for tabular data. It supports persisting inferred schemas and
// comparing them across pipeline runs to detect schema drift.
package schema

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"dq-pipeline/internal/parse"
)

// ColType represents the inferred type of a column.
type ColType string

const (
	TypeString  ColType = "string"
	TypeInt     ColType = "int"
	TypeFloat   ColType = "float"
	TypeBool    ColType = "bool"
	TypeDate    ColType = "date"
	TypeUnknown ColType = "unknown"
)

// Column holds schema information for a single column.
type Column struct {
	Name     string  `json:"name"`
	Type     ColType `json:"type"`
	Nullable bool    `json:"nullable"`
	MaxLen   int     `json:"max_len,omitempty"`
	MinLen   int     `json:"min_len,omitempty"`
}

// Schema describes the structure of a table.
type Schema struct {
	Columns []Column `json:"columns"`
	Version int      `json:"version"`
}

// InferOptions controls schema inference behavior.
type InferOptions struct {
	DateLayouts []string // date formats to try (default: RFC3339, 2006-01-02)
	SampleSize  int     // max rows to sample (0 = all)
}

// DefaultDateLayouts are the date formats tried during inference.
var DefaultDateLayouts = []string{
	"2006-01-02",
	"2006-01-02T15:04:05Z07:00",
	"02/01/2006",
	"01-02-2006",
	"2006/01/02",
}

// Infer analyzes a table and returns the inferred schema.
func Infer(t *parse.Table, opts *InferOptions) *Schema {
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
	return fillSchema(*schema)
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

	// counters for type detection
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
			floatCount++ // ints are also valid floats
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

	// determine dominant type (all non-empty values must match)
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

// Validate checks whether all rows in t conform to the given schema.
// Returns a list of violations (column index, row index, reason).
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

// SchemaViolation records a row that does not conform to the schema.
type SchemaViolation struct {
	Column string
	Row    int
	Reason string
}

// ColumnByName finds a column definition by name, or nil if not found.
func (s *Schema) ColumnByName(name string) *Column {
	for i := range s.Columns {
		if s.Columns[i].Name == name {
			return &s.Columns[i]
		}
	}
	return nil
}

// ColumnNames returns all column names in order.
func (s *Schema) ColumnNames() []string {
	names := make([]string, len(s.Columns))
	for i, c := range s.Columns {
		names[i] = c.Name
	}
	return names
}
