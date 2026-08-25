package schema

import "fmt"

type DriftKind string

const (
	DriftAdded      DriftKind = "ADDED"
	DriftRemoved    DriftKind = "REMOVED"
	DriftTypeChange DriftKind = "TYPE_CHANGE"
	DriftNullable   DriftKind = "NULLABLE_CHANGE"
)

type DriftEntry struct {
	Column  string    `json:"column"`
	Kind    DriftKind `json:"kind"`
	OldType ColType   `json:"old_type,omitempty"`
	NewType ColType   `json:"new_type,omitempty"`
	Detail  string    `json:"detail,omitempty"`
}

type DriftReport struct {
	Entries    []DriftEntry `json:"entries"`
	Breaking   int          `json:"breaking"`
	NonBreak   int          `json:"non_breaking"`
	Compatible bool         `json:"compatible"`
}

func DetectDrift(old, new *Schema) *DriftReport {
	report := &DriftReport{Compatible: true}

	oldMap := make(map[string]*Column)
	for i := range old.Columns {
		oldMap[old.Columns[i].Name] = &old.Columns[i]
	}
	newMap := make(map[string]*Column)
	for i := range new.Columns {
		newMap[new.Columns[i].Name] = &new.Columns[i]
	}

	for _, oc := range old.Columns {
		if _, found := newMap[oc.Name]; !found {
			report.Entries = append(report.Entries, DriftEntry{
				Column: oc.Name,
				Kind:   DriftRemoved,
				Detail: "column removed",
			})
			report.Breaking++
			report.Compatible = false
		}
	}

	for _, nc := range new.Columns {
		if _, found := oldMap[nc.Name]; !found {
			report.Entries = append(report.Entries, DriftEntry{
				Column: nc.Name,
				Kind:   DriftAdded,
				Detail: "column added",
			})
			report.NonBreak++
		}
	}

	for _, oc := range old.Columns {
		nc, found := newMap[oc.Name]
		if !found {
			continue
		}
		if oc.Type != nc.Type {
			entry := DriftEntry{
				Column:  oc.Name,
				Kind:    DriftTypeChange,
				OldType: oc.Type,
				NewType: nc.Type,
				Detail:  fmt.Sprintf("type changed from %s to %s", oc.Type, nc.Type),
			}
			if isNarrowing(oc.Type, nc.Type) {
				report.Breaking++
				report.Compatible = false
			} else {
				report.NonBreak++
			}
			report.Entries = append(report.Entries, entry)
		}
		if oc.Nullable != nc.Nullable {
			entry := DriftEntry{
				Column: oc.Name,
				Kind:   DriftNullable,
				Detail: fmt.Sprintf("nullable changed from %v to %v", oc.Nullable, nc.Nullable),
			}
			if !nc.Nullable && oc.Nullable {
				report.Breaking++
				report.Compatible = false
			} else {
				report.NonBreak++
			}
			report.Entries = append(report.Entries, entry)
		}
	}

	return report
}

func isNarrowing(old, new ColType) bool {
	rank := map[ColType]int{
		TypeBool:   0,
		TypeInt:    1,
		TypeFloat:  2,
		TypeDate:   2,
		TypeString: 3,
	}
	oldRank, okOld := rank[old]
	newRank, okNew := rank[new]
	if !okOld || !okNew {
		return true
	}
	return newRank < oldRank
}

func (r *DriftReport) IsBreaking() bool {
	return r.Breaking > 0
}

func (r *DriftReport) Summary() string {
	if len(r.Entries) == 0 {
		return "no schema drift detected"
	}
	return fmt.Sprintf("%d drift(s): %d breaking, %d non-breaking, compatible=%v",
		len(r.Entries), r.Breaking, r.NonBreak, r.Compatible)
}

func (r *DriftReport) AffectedColumns() []string {
	seen := make(map[string]bool)
	var cols []string
	for _, e := range r.Entries {
		if !seen[e.Column] {
			seen[e.Column] = true
			cols = append(cols, e.Column)
		}
	}
	return cols
}
