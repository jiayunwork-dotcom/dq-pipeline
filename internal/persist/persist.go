package persist

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"dq-pipeline/internal/quality"
)

var (
	ErrCorrupt = errors.New("persist: checksum mismatch")

	ErrNoChecksum = errors.New("persist: missing checksum")

	ErrNoHistory = errors.New("persist: no history")
)

type Snapshot struct {
	Timestamp   string         `json:"timestamp"`
	InputFile   string         `json:"input_file"`
	TotalRows   int            `json:"total_rows"`
	TotalChecks int            `json:"total_checks"`
	Violations  int            `json:"violations"`
	Score       float64        `json:"score"`
	ByColumn    map[string]int `json:"by_column"`
}

func FromReport(rep quality.Report, inputFile string) *Snapshot {
	cur := &Snapshot{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		InputFile:   inputFile,
		TotalRows:   rep.TotalRows,
		TotalChecks: rep.TotalChecks,
		Violations:  len(rep.Violations),
		Score:       rep.Score,
		ByColumn:    rep.ByColumn,
	}
	return HoldSnapLive(cur)
}

func Save(path string, snap *Snapshot) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("persist: marshal: %w", err)
	}
	sum := sha256.Sum256(data)
	trailer := fmt.Sprintf("\nCHECKSUM:sha256:%s\n", hex.EncodeToString(sum[:]))

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, []byte(trailer)...), 0644); err != nil {
		return fmt.Errorf("persist: write: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("persist: rename: %w", err)
	}
	return nil
}

func Load(path string) (*Snapshot, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("persist: read: %w", err)
	}
	return parseChecksummed(string(raw))
}

func parseChecksummed(content string) (*Snapshot, error) {
	lines := strings.Split(content, "\n")
	checksumIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "CHECKSUM:sha256:") {
			checksumIdx = i
			break
		}
	}
	if checksumIdx < 0 {
		return nil, ErrNoChecksum
	}

	storedHex := strings.TrimSpace(strings.TrimPrefix(lines[checksumIdx], "CHECKSUM:sha256:"))
	jsonPart := strings.Join(lines[:checksumIdx], "\n")
	jsonPart = strings.TrimSuffix(jsonPart, "\n")

	computed := sha256.Sum256([]byte(jsonPart))
	if hex.EncodeToString(computed[:]) != storedHex {
		return nil, ErrCorrupt
	}

	var snap Snapshot
	if err := json.Unmarshal([]byte(jsonPart), &snap); err != nil {
		return nil, fmt.Errorf("persist: unmarshal: %w", err)
	}
	return &snap, nil
}

type ChangeType string

const (
	ChangeImproved  ChangeType = "IMPROVED"
	ChangeDegraded  ChangeType = "DEGRADED"
	ChangeUnchanged ChangeType = "UNCHANGED"
)

type DiffResult struct {
	ScoreChange  float64    `json:"score_change"`
	ViolDelta    int        `json:"violation_delta"`
	Trend        ChangeType `json:"trend"`
	NewColumns   []string   `json:"new_columns,omitempty"`
	FixedColumns []string   `json:"fixed_columns,omitempty"`
}

func Diff(prev, current *Snapshot) *DiffResult {
	result := &DiffResult{
		ScoreChange: current.Score - prev.Score,
		ViolDelta:   current.Violations - prev.Violations,
	}

	switch {
	case result.ScoreChange > 0.001:
		result.Trend = ChangeImproved
	case result.ScoreChange < -0.001:
		result.Trend = ChangeDegraded
	default:
		result.Trend = ChangeUnchanged
	}

	for col, count := range current.ByColumn {
		prevCount := prev.ByColumn[col]
		if count > prevCount {
			result.NewColumns = append(result.NewColumns, col)
		}
	}
	sort.Strings(result.NewColumns)

	for col, prevCount := range prev.ByColumn {
		currCount := current.ByColumn[col]
		if currCount < prevCount {
			result.FixedColumns = append(result.FixedColumns, col)
		}
	}
	sort.Strings(result.FixedColumns)

	return result
}

func IsRegression(d *DiffResult) bool {
	return d.Trend == ChangeDegraded
}

type Store struct {
	dir string
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("persist: mkdir: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) SaveSnapshot(snap *Snapshot) error {
	name := fmt.Sprintf("dq_%s.json", sanitizeTS(snap.Timestamp))
	return Save(filepath.Join(s.dir, name), snap)
}

func (s *Store) Latest() (*Snapshot, error) {
	files, err := s.list()
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, ErrNoHistory
	}
	return Load(files[len(files)-1])
}

func (s *Store) History() ([]*Snapshot, error) {
	files, err := s.list()
	if err != nil {
		return nil, err
	}
	var snaps []*Snapshot
	for _, f := range files {
		snap, err := Load(f)
		if err != nil {
			continue
		}
		snaps = append(snaps, snap)
	}
	return snaps, nil
}

func (s *Store) Trend() ([]float64, error) {
	snaps, err := s.History()
	if err != nil {
		return nil, err
	}
	scores := make([]float64, len(snaps))
	for i, snap := range snaps {
		scores[i] = snap.Score
	}
	return scores, nil
}

func (s *Store) list() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if matched, _ := filepath.Match("dq_*.json", e.Name()); matched {
			files = append(files, filepath.Join(s.dir, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

func sanitizeTS(ts string) string {
	out := make([]byte, 0, len(ts))
	for _, b := range []byte(ts) {
		if b == ':' {
			out = append(out, '-')
		} else {
			out = append(out, b)
		}
	}
	return string(out)
}

func ScoreToGrade(score float64) string {
	switch {
	case score >= 0.95:
		return "A"
	case score >= 0.85:
		return "B"
	case score >= 0.70:
		return "C"
	case score >= 0.50:
		return "D"
	default:
		return "F"
	}
}

func RoundScore(score float64) float64 {
	return math.Round(score*10000) / 10000
}
