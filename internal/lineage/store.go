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
	ErrTruncated = errors.New("lineage: log truncated")
)

type Store struct {
	path string
	file *os.File
	w    *bufio.Writer
}

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

func (s *Store) Flush() error {
	if err := s.w.Flush(); err != nil {
		return fmt.Errorf("lineage: flush: %w", err)
	}
	return s.file.Sync()
}

func (s *Store) Close() error {
	if err := s.Flush(); err != nil {
		return err
	}
	return s.file.Close()
}

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
			truncated = true
			continue
		}
		if truncated {
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
	return t, err
}

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
		if line[0] == '{' {
			count++
		}
	}
	return count, scanner.Err()
}

func Compact(path string) error {
	entries, err := ReadLog(path)
	if err != nil && !errors.Is(err, ErrTruncated) {
		return err
	}

	dropped := make(map[int]bool)
	for _, e := range entries {
		if e.Action == ActionDrop {
			dropped[e.RowID] = true
		}
	}

	var kept []Entry
	for _, e := range entries {
		if dropped[e.RowID] && e.Action != ActionDrop {
			continue
		}
		kept = append(kept, e)
	}

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
