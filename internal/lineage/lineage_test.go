package lineage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrackerRecordAndRetrieve(t *testing.T) {
	tr := NewTracker()
	tr.Record(0, "parse", ActionPass, "")
	tr.Record(1, "parse", ActionPass, "")
	tr.Record(0, "clean", ActionModify, "trimmed")

	if tr.Len() != 3 {
		t.Fatalf("Len = %d, want 3", tr.Len())
	}
	entries := tr.EntriesForRow(0)
	if len(entries) != 2 {
		t.Errorf("row 0 entries = %d, want 2", len(entries))
	}
}

func TestTrackerDroppedRows(t *testing.T) {
	tr := NewTracker()
	tr.Record(0, "parse", ActionPass, "")
	tr.Record(1, "clean", ActionDrop, "critical violation")
	tr.Record(2, "parse", ActionPass, "")
	tr.Record(1, "report", ActionDrop, "second drop")

	dropped := tr.DroppedRows()
	if len(dropped) != 1 || dropped[0] != 1 {
		t.Errorf("dropped = %v, want [1]", dropped)
	}
}

func TestTrackerModifiedRows(t *testing.T) {
	tr := NewTracker()
	tr.Record(0, "clean", ActionModify, "trimmed")
	tr.Record(1, "clean", ActionPass, "")
	tr.Record(2, "transform", ActionModify, "uppercased")

	modified := tr.ModifiedRows()
	if len(modified) != 2 {
		t.Errorf("modified = %v, want 2", modified)
	}
}

func TestTrackerChain(t *testing.T) {
	tr := NewTracker()
	tr.Record(5, "parse", ActionPass, "")
	tr.Record(5, "clean", ActionModify, "")
	tr.Record(5, "transform", ActionModify, "")

	chain := tr.Chain(5)
	if len(chain) != 3 {
		t.Fatalf("chain = %v", chain)
	}
	if chain[0] != "parse" || chain[1] != "clean" || chain[2] != "transform" {
		t.Errorf("chain = %v", chain)
	}
}

func TestTrackerValidate(t *testing.T) {
	tr := NewTracker()
	tr.Record(0, "clean", ActionDrop, "removed")
	tr.Record(0, "transform", ActionModify, "should not happen after drop")

	issues := tr.Validate()
	if len(issues) != 1 {
		t.Errorf("issues = %d, want 1", len(issues))
	}
}

func TestTrackerStageCount(t *testing.T) {
	tr := NewTracker()
	tr.Record(0, "parse", ActionPass, "")
	tr.Record(1, "parse", ActionPass, "")
	tr.Record(2, "parse", ActionPass, "")
	tr.Record(0, "clean", ActionModify, "")

	if tr.StageCount("parse") != 3 {
		t.Errorf("parse count = %d", tr.StageCount("parse"))
	}
	if tr.StageCount("clean") != 1 {
		t.Errorf("clean count = %d", tr.StageCount("clean"))
	}
}

func TestStoreAppendAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lineage.jsonl")

	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	store.Append(Entry{RowID: 0, Stage: "parse", Action: ActionPass})
	store.Append(Entry{RowID: 1, Stage: "clean", Action: ActionModify, Detail: "trimmed"})
	store.Close()

	entries, err := ReadLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[1].Detail != "trimmed" {
		t.Errorf("detail = %q", entries[1].Detail)
	}
}

func TestReadLogTruncated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lineage.jsonl")

	// write valid line + truncated line
	data := `{"row_id":0,"stage":"parse","action":"PASS","timestamp":"2026-01-01T00:00:00Z"}
{"row_id":1,"stage":"cle`
	os.WriteFile(path, []byte(data), 0644)

	entries, err := ReadLog(path)
	if err == nil {
		t.Fatal("expected ErrTruncated")
	}
	if len(entries) != 1 {
		t.Errorf("valid entries = %d, want 1", len(entries))
	}
}

func TestReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lineage.jsonl")

	store, _ := OpenStore(path)
	store.Append(Entry{RowID: 0, Stage: "parse", Action: ActionPass})
	store.Append(Entry{RowID: 0, Stage: "clean", Action: ActionModify})
	store.Append(Entry{RowID: 1, Stage: "parse", Action: ActionDrop})
	store.Close()

	tracker, err := Replay(path)
	if err != nil {
		t.Fatal(err)
	}
	if tracker.Len() != 3 {
		t.Errorf("replayed entries = %d, want 3", tracker.Len())
	}
	if tracker.StageCount("parse") != 2 {
		t.Errorf("parse count = %d", tracker.StageCount("parse"))
	}
}

func TestCompact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lineage.jsonl")

	store, _ := OpenStore(path)
	store.Append(Entry{RowID: 0, Stage: "parse", Action: ActionPass})
	store.Append(Entry{RowID: 1, Stage: "parse", Action: ActionPass})
	store.Append(Entry{RowID: 1, Stage: "clean", Action: ActionDrop})
	store.Append(Entry{RowID: 0, Stage: "clean", Action: ActionModify})
	store.Close()

	if err := Compact(path); err != nil {
		t.Fatal(err)
	}

	entries, _ := ReadLog(path)
	// row 1 was dropped: keep DROP entry but remove PASS
	// row 0: keep both entries
	// expect: row0-parse, row1-drop, row0-clean = 3
	if len(entries) != 3 {
		t.Errorf("after compact: entries = %d, want 3", len(entries))
	}
}

func TestCountEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lineage.jsonl")

	store, _ := OpenStore(path)
	store.Append(Entry{RowID: 0, Stage: "a", Action: ActionPass})
	store.Append(Entry{RowID: 1, Stage: "b", Action: ActionPass})
	store.Close()

	count, err := CountEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}
