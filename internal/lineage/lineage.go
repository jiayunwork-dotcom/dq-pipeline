// Package lineage tracks data transformation history for each row through
// the pipeline. It records which stages processed each row, enabling
// auditability and root-cause analysis when data quality issues are detected.
package lineage

import (
	"fmt"
	"strings"
	"time"
)

// Action describes what happened to a row at a given stage.
type Action string

const (
	ActionPass    Action = "PASS"
	ActionModify  Action = "MODIFY"
	ActionDrop    Action = "DROP"
	ActionCreate  Action = "CREATE"
	ActionSplit   Action = "SPLIT"
	ActionMerge   Action = "MERGE"
)

// Entry records a single lineage event for a row.
type Entry struct {
	RowID     int       `json:"row_id"`
	Stage     string    `json:"stage"`
	Action    Action    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	Detail    string    `json:"detail,omitempty"`
}

// Tracker accumulates lineage entries during pipeline execution.
type Tracker struct {
	entries []Entry
	counts  map[string]int // per-stage action counts
}

// NewTracker creates an empty tracker.
func NewTracker() *Tracker {
	return &Tracker{
		counts: make(map[string]int),
	}
}

// Record adds a lineage entry.
func (t *Tracker) Record(rowID int, stage string, action Action, detail string) {
	t.entries = append(t.entries, Entry{
		RowID:     rowID,
		Stage:     stage,
		Action:    action,
		Timestamp: time.Now(),
		Detail:    detail,
	})
	t.counts[stage]++
}

// Entries returns all recorded entries in order.
func (t *Tracker) Entries() []Entry {
	out := make([]Entry, len(t.entries))
	copy(out, t.entries)
	return out
}

// EntriesForRow returns all entries for a specific row.
func (t *Tracker) EntriesForRow(rowID int) []Entry {
	var out []Entry
	for _, e := range t.entries {
		if e.RowID == rowID {
			out = append(out, e)
		}
	}
	return out
}

// EntriesForStage returns entries for a specific stage.
func (t *Tracker) EntriesForStage(stage string) []Entry {
	var out []Entry
	for _, e := range t.entries {
		if e.Stage == stage {
			out = append(out, e)
		}
	}
	return out
}

// StageCount returns the total number of entries for a stage.
func (t *Tracker) StageCount(stage string) int {
	return t.counts[stage]
}

// DroppedRows returns row IDs that were dropped at any stage.
func (t *Tracker) DroppedRows() []int {
	seen := make(map[int]bool)
	var result []int
	for _, e := range t.entries {
		if e.Action == ActionDrop && !seen[e.RowID] {
			seen[e.RowID] = true
			result = append(result, e.RowID)
		}
	}
	return result
}

// ModifiedRows returns row IDs that were modified at any stage.
func (t *Tracker) ModifiedRows() []int {
	seen := make(map[int]bool)
	var result []int
	for _, e := range t.entries {
		if e.Action == ActionModify && !seen[e.RowID] {
			seen[e.RowID] = true
			result = append(result, e.RowID)
		}
	}
	return result
}

// Len returns the total number of entries.
func (t *Tracker) Len() int {
	return len(t.entries)
}

// Summary returns a human-readable summary of lineage activity.
func (t *Tracker) Summary() string {
	if len(t.entries) == 0 {
		return "no lineage recorded"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "lineage: %d entries across %d stages\n", len(t.entries), len(t.counts))
	actionCounts := make(map[Action]int)
	for _, e := range t.entries {
		actionCounts[e.Action]++
	}
	for action, count := range actionCounts {
		fmt.Fprintf(&b, "  %s: %d\n", action, count)
	}
	return b.String()
}

// Chain returns the ordered list of stages a row passed through.
func (t *Tracker) Chain(rowID int) []string {
	var stages []string
	for _, e := range t.entries {
		if e.RowID == rowID {
			stages = append(stages, e.Stage)
		}
	}
	return stages
}

// Validate checks that no row has contradictory lineage (e.g., modified after drop).
func (t *Tracker) Validate() []string {
	dropped := make(map[int]bool)
	var issues []string
	for _, e := range t.entries {
		if e.Action == ActionDrop {
			dropped[e.RowID] = true
			continue
		}
		if dropped[e.RowID] && (e.Action == ActionModify || e.Action == ActionPass) {
			issues = append(issues, fmt.Sprintf("row %d: %s after DROP at stage %q", e.RowID, e.Action, e.Stage))
		}
	}
	return issues
}
