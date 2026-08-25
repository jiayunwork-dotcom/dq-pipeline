package parse

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func DetectFormat(header string) string {
	trimmed := strings.TrimSpace(header)
	if len(trimmed) == 0 {
		return "csv"
	}
	if trimmed[0] == '[' || trimmed[0] == '{' {
		return "json"
	}
	tabs := strings.Count(header, "\t")
	commas := strings.Count(header, ",")
	if tabs > commas {
		return "tsv"
	}
	return "csv"
}

func ParseJSON(r io.Reader) (*Table, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("parse json: read: %w", err)
	}

	var records []map[string]interface{}
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("parse json: unmarshal: %w", err)
	}

	if len(records) == 0 {
		return &Table{}, nil
	}

	keyOrder := make([]string, 0)
	keySeen := make(map[string]bool)
	for _, rec := range records {
		for k := range rec {
			if !keySeen[k] {
				keySeen[k] = true
				keyOrder = append(keyOrder, k)
			}
		}
	}

	table := &Table{Header: keyOrder}
	for _, rec := range records {
		row := make([]string, len(keyOrder))
		for i, k := range keyOrder {
			if v, ok := rec[k]; ok {
				row[i] = valueToString(v)
			}
		}
		table.Rows = append(table.Rows, row)
	}
	return table, nil
}

func ParseFixed(r io.Reader, widths []int) (*Table, error) {
	if len(widths) == 0 {
		return nil, fmt.Errorf("parse fixed: no widths provided")
	}

	scanner := bufio.NewScanner(r)
	var table Table
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		if lineNum == 0 && line == "" {
			lineNum++
			continue
		}

		row := splitFixed(line, widths)
		if lineNum == 0 {
			table.Header = row
		} else {
			table.Rows = append(table.Rows, row)
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse fixed: scan: %w", err)
	}
	return &table, nil
}

func splitFixed(line string, widths []int) []string {
	fields := make([]string, len(widths))
	pos := 0
	for i, w := range widths {
		end := pos + w
		if pos >= len(line) {
			fields[i] = ""
		} else if end > len(line) {
			fields[i] = strings.TrimSpace(line[pos:])
		} else {
			fields[i] = strings.TrimSpace(line[pos:end])
		}
		pos = end
	}
	return fields
}

func valueToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func AutoParse(r io.Reader) (*Table, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("auto parse: read: %w", err)
	}
	content := string(data)
	if len(content) == 0 {
		return &Table{}, nil
	}

	firstLine := content
	if idx := strings.IndexByte(content, '\n'); idx >= 0 {
		firstLine = content[:idx]
	}

	format := DetectFormat(firstLine)
	switch format {
	case "json":
		return ParseJSON(strings.NewReader(content))
	case "tsv":
		return Parse(strings.NewReader(content), '\t')
	default:
		return Parse(strings.NewReader(content), ',')
	}
}
