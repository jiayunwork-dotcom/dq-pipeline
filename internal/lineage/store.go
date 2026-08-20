package lineage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	// ErrTruncated indicates the log file ends with a partial record.
	ErrTruncated = errors.New("lineage: log truncated")
)

// Store persists lineage entries as an append-only JSON-lines log file.
// Each line is a complete JSON object representing one Entry. Truncated
// trailing lines (from crash/power loss) are detected and reported but
// do not corrupt the preceding valid entries.
type Store struct {
	path string
	file *os.File
	w    *bufio.Writer
}

// OpenStore opens or creates a lineage log at the given path.
func OpenStore(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("lineage: mkdir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("lineage: open: %w", err)
	}
	return &Store{
		path: path,
		file: f,
		w:    bufio.NewWriter(f),
	}, nil
}

// Append writes a single entry to the log. Each entry is one JSON line.
func (s *Store) Append(e Entry) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("lineage: marshal: %w", err)
	}
	data = append(data, '\n')
	if _, err := s.w.Write(data); err != nil {
		return fmt.Errorf("lineage: write: %w", err)
	}
	return nil
}

// Flush ensures all buffered data is written to disk.
func (s *Store) Flush() error {
	if err := s.w.Flush(); err != nil {
		return fmt.Errorf("lineage: flush: %w", err)
	}
	return s.file.Sync()
}

// Close flushes and closes the underlying file.
func (s *Store) Close() error {
	if err := s.Flush(); err != nil {
		return err
	}
	return s.file.Close()
}

// ReadLog reads all valid entries from a lineage log file. If the last line
// is incomplete (truncated), it is skipped and ErrTruncated is returned as
// the second value alongside the valid entries.
func ReadLog(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("lineage: open read: %w", err)
	}
	defer f.Close()

	var entries []Entry
	var truncated bool
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			// if this is the last line, it may be truncated
			truncated = true
			continue
		}
		// if we previously saw a bad line, it was not the last
		if truncated {
			// non-last bad line is corruption
			return entries, fmt.Errorf("lineage: corrupt line in log")
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("lineage: scan: %w", err)
	}
	if truncated {
		return entries, ErrTruncated
	}
	return entries, nil
}

// Replay rebuilds a Tracker from a log file, skipping truncated tail.
func Replay(path string) (*Tracker, error) {
	entries, err := ReadLog(path)
	if err != nil && !errors.Is(err, ErrTruncated) {
		return nil, err
	}
	t := NewTracker()
	for _, e := range entries {
		t.entries = append(t.entries, e)
		t.counts[e.Stage]++
	}
	return t, err // pass through ErrTruncated if present
}

// CountEntries returns how many valid entries exist in a log file without
// loading them all into memory.
func CountEntries(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// quick validation: must start with {
		if line[0] == '{' {
			count++
		}
	}
	return count, scanner.Err()
}

// Compact reads a log, drops entries for dropped rows, and rewrites it.
func Compact(path string) error {
	entries, err := ReadLog(path)
	if err != nil && !errors.Is(err, ErrTruncated) {
		return err
	}

	// find dropped rows
	dropped := make(map[int]bool)
	for _, e := range entries {
		if e.Action == ActionDrop {
			dropped[e.RowID] = true
		}
	}

	// filter: keep only entries for non-dropped rows, plus the DROP itself
	var kept []Entry
	for _, e := range entries {
		if dropped[e.RowID] && e.Action != ActionDrop {
			continue
		}
		kept = append(kept, e)
	}

	// atomic rewrite
	tmp := path + ".compact"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("lineage: create compact: %w", err)
	}
	w := bufio.NewWriter(f)
	for _, e := range kept {
		data, _ := json.Marshal(e)
		w.Write(data)
		io.WriteString(w, "\n")
	}
	w.Flush()
	f.Close()

	return os.Rename(tmp, path)
}
