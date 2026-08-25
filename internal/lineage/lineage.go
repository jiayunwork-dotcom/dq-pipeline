package lineage

import (
	"fmt"
	"strings"
	"time"
)

type Action string

const (
	ActionPass   Action = "PASS"
	ActionModify Action = "MODIFY"
	ActionDrop   Action = "DROP"
	ActionCreate Action = "CREATE"
	ActionSplit  Action = "SPLIT"
	ActionMerge  Action = "MERGE"
)

type Entry struct {
	RowID     int       `json:"row_id"`
	Stage     string    `json:"stage"`
	Action    Action    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	Detail    string    `json:"detail,omitempty"`
}

type Tracker struct {
	entries []Entry
	counts  map[string]int
}

func NewTracker() *Tracker {
	return &Tracker{
		counts: make(map[string]int),
	}
}

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

func (t *Tracker) Entries() []Entry {
	out := make([]Entry, len(t.entries))
	copy(out, t.entries)
	return out
}

func (t *Tracker) EntriesForRow(rowID int) []Entry {
	var out []Entry
	for _, e := range t.entries {
		if e.RowID == rowID {
			out = append(out, e)
		}
	}
	return out
}

func (t *Tracker) EntriesForStage(stage string) []Entry {
	var out []Entry
	for _, e := range t.entries {
		if e.Stage == stage {
			out = append(out, e)
		}
	}
	return out
}

func (t *Tracker) StageCount(stage string) int {
	return t.counts[stage]
}

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

func (t *Tracker) Len() int {
	return len(t.entries)
}

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

func (t *Tracker) Chain(rowID int) []string {
	var stages []string
	for _, e := range t.entries {
		if e.RowID == rowID {
			stages = append(stages, e.Stage)
		}
	}
	return stages
}

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
