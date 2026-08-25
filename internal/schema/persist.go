package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	ErrSchemaCorrupt = errors.New("schema: checksum mismatch")

	ErrSchemaNoChecksum = errors.New("schema: missing checksum")
)

func SaveSchema(path string, s *Schema) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("schema: marshal: %w", err)
	}
	sum := sha256.Sum256(data)
	trailer := fmt.Sprintf("\nSCHEMA_CHECKSUM:sha256:%s\n", hex.EncodeToString(sum[:]))

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, []byte(trailer)...), 0644); err != nil {
		return fmt.Errorf("schema: write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("schema: rename: %w", err)
	}
	return nil
}

func LoadSchema(path string) (*Schema, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("schema: read: %w", err)
	}
	return parseSchemaFile(string(raw))
}

func parseSchemaFile(content string) (*Schema, error) {
	lines := strings.Split(content, "\n")
	checksumIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "SCHEMA_CHECKSUM:sha256:") {
			checksumIdx = i
			break
		}
	}
	if checksumIdx < 0 {
		return nil, ErrSchemaNoChecksum
	}

	storedHex := strings.TrimSpace(strings.TrimPrefix(lines[checksumIdx], "SCHEMA_CHECKSUM:sha256:"))
	jsonPart := strings.Join(lines[:checksumIdx], "\n")
	jsonPart = strings.TrimSuffix(jsonPart, "\n")

	computed := sha256.Sum256([]byte(jsonPart))
	if hex.EncodeToString(computed[:]) != storedHex {
		return nil, ErrSchemaCorrupt
	}

	var s Schema
	if err := json.Unmarshal([]byte(jsonPart), &s); err != nil {
		return nil, fmt.Errorf("schema: unmarshal: %w", err)
	}
	return &s, nil
}

func SchemaEqual(a, b *Schema) bool {
	if len(a.Columns) != len(b.Columns) {
		return false
	}
	for i := range a.Columns {
		if a.Columns[i].Name != b.Columns[i].Name {
			return false
		}
		if a.Columns[i].Type != b.Columns[i].Type {
			return false
		}
		if a.Columns[i].Nullable != b.Columns[i].Nullable {
			return false
		}
	}
	return true
}
