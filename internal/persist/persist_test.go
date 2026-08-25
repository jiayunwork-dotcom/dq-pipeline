package persist

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"dq-pipeline/internal/quality"
)

func sampleReport() quality.Report {
	return quality.Report{
		TotalRows:   100,
		TotalChecks: 400,
		Violations: []quality.Violation{
			{Rule: 0, Column: "name", Row: 5, Reason: "empty"},
			{Rule: 1, Column: "age", Row: 10, Reason: "out of range"},
		},
		Score:    0.995,
		ByColumn: map[string]int{"name": 1, "age": 1},
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	snap := FromReport(sampleReport(), "input.csv")
	if err := Save(path, snap); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TotalRows != 100 {
		t.Errorf("TotalRows = %d, want 100", loaded.TotalRows)
	}
	if loaded.Violations != 2 {
		t.Errorf("Violations = %d, want 2", loaded.Violations)
	}
	if loaded.InputFile != "input.csv" {
		t.Errorf("InputFile = %q", loaded.InputFile)
	}
}

func TestLoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	snap := FromReport(sampleReport(), "x.csv")
	Save(path, snap)

	data, _ := os.ReadFile(path)
	data[5] ^= 0xFF
	os.WriteFile(path, data, 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected checksum error")
	}
}

func TestLoadNoChecksum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nocs.json")
	os.WriteFile(path, []byte(`{"score":0.9}`), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected no-checksum error")
	}
}

func TestDiffImproved(t *testing.T) {
	prev := &Snapshot{Score: 0.8, Violations: 10, ByColumn: map[string]int{"a": 5, "b": 5}}
	curr := &Snapshot{Score: 0.95, Violations: 3, ByColumn: map[string]int{"a": 2, "b": 1}}

	d := Diff(prev, curr)
	if d.Trend != ChangeImproved {
		t.Errorf("Trend = %s, want IMPROVED", d.Trend)
	}
	if len(d.FixedColumns) != 2 {
		t.Errorf("FixedColumns = %v", d.FixedColumns)
	}
}

func TestDiffDegraded(t *testing.T) {
	prev := &Snapshot{Score: 0.95, Violations: 2, ByColumn: map[string]int{"a": 2}}
	curr := &Snapshot{Score: 0.7, Violations: 15, ByColumn: map[string]int{"a": 5, "b": 10}}

	d := Diff(prev, curr)
	if d.Trend != ChangeDegraded {
		t.Errorf("Trend = %s, want DEGRADED", d.Trend)
	}
	if !IsRegression(d) {
		t.Error("expected IsRegression=true")
	}
}

func TestDiffUnchanged(t *testing.T) {
	snap := &Snapshot{Score: 0.9, Violations: 5, ByColumn: map[string]int{"x": 5}}
	d := Diff(snap, snap)
	if d.Trend != ChangeUnchanged {
		t.Errorf("Trend = %s, want UNCHANGED", d.Trend)
	}
}

func TestStoreLatest(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewStore(dir)

	s1 := &Snapshot{Timestamp: "2026-08-01T10:00:00Z", Score: 0.8, ByColumn: map[string]int{}}
	s2 := &Snapshot{Timestamp: "2026-08-02T10:00:00Z", Score: 0.9, ByColumn: map[string]int{}}
	store.SaveSnapshot(s1)
	store.SaveSnapshot(s2)

	latest, err := store.Latest()
	if err != nil {
		t.Fatal(err)
	}
	if latest.Score != 0.9 {
		t.Errorf("latest.Score = %f, want 0.9", latest.Score)
	}
}

func TestStoreNoHistory(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewStore(dir)
	_, err := store.Latest()
	if err == nil {
		t.Error("expected ErrNoHistory")
	}
}

func TestStoreTrend(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewStore(dir)

	for i, score := range []float64{0.7, 0.8, 0.85, 0.9} {
		snap := &Snapshot{
			Timestamp: fmt.Sprintf("2026-08-0%dT10:00:00Z", i+1),
			Score:     score,
			ByColumn:  map[string]int{},
		}
		store.SaveSnapshot(snap)
	}

	trend, err := store.Trend()
	if err != nil {
		t.Fatal(err)
	}
	if len(trend) != 4 {
		t.Fatalf("trend len = %d, want 4", len(trend))
	}
	if trend[0] != 0.7 || trend[3] != 0.9 {
		t.Errorf("trend = %v", trend)
	}
}

func TestScoreToGrade(t *testing.T) {
	cases := []struct {
		score float64
		want  string
	}{
		{0.98, "A"},
		{0.90, "B"},
		{0.75, "C"},
		{0.55, "D"},
		{0.30, "F"},
	}
	for _, c := range cases {
		if g := ScoreToGrade(c.score); g != c.want {
			t.Errorf("ScoreToGrade(%f) = %q, want %q", c.score, g, c.want)
		}
	}
}

func TestRoundScore(t *testing.T) {
	if r := RoundScore(0.12345); r != 0.1235 {
		t.Errorf("RoundScore = %f, want 0.1235", r)
	}
}

func TestFromReport(t *testing.T) {
	rep := sampleReport()
	snap := FromReport(rep, "test.csv")
	if snap.InputFile != "test.csv" {
		t.Errorf("InputFile = %q", snap.InputFile)
	}
	if snap.Violations != 2 {
		t.Errorf("Violations = %d", snap.Violations)
	}
	if snap.Timestamp == "" {
		t.Error("Timestamp should be auto-filled")
	}
}
