package schema

import "fmt"

// DriftKind classifies what changed between two schema versions.
type DriftKind string

const (
	DriftAdded      DriftKind = "ADDED"
	DriftRemoved    DriftKind = "REMOVED"
	DriftTypeChange DriftKind = "TYPE_CHANGE"
	DriftNullable   DriftKind = "NULLABLE_CHANGE"
)

// DriftEntry describes a single schema difference.
type DriftEntry struct {
	Column  string    `json:"column"`
	Kind    DriftKind `json:"kind"`
	OldType ColType   `json:"old_type,omitempty"`
	NewType ColType   `json:"new_type,omitempty"`
	Detail  string    `json:"detail,omitempty"`
}

// DriftReport holds all drift entries between two schemas.
type DriftReport struct {
	Entries    []DriftEntry `json:"entries"`
	Breaking   int          `json:"breaking"`
	NonBreak   int          `json:"non_breaking"`
	Compatible bool         `json:"compatible"`
}

// DetectDrift compares an old schema to a new schema and returns all differences.
// A change is "breaking" if it removes a column or narrows a type.
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

	// check removed columns
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

	// check added columns
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

	// check type and nullable changes for columns present in both
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
				// making non-nullable is breaking
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

// isNarrowing returns true if changing from old to new type loses information.
func isNarrowing(old, new ColType) bool {
	// widening hierarchy: int -> float -> string (always safe)
	// narrowing: string -> float -> int (unsafe)
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
		return true // unknown direction is considered breaking
	}
	return newRank < oldRank
}

// IsBreaking returns true if any drift entry is breaking.
func (r *DriftReport) IsBreaking() bool {
	return r.Breaking > 0
}

// Summary returns a human-readable summary string.
func (r *DriftReport) Summary() string {
	if len(r.Entries) == 0 {
		return "no schema drift detected"
	}
	return fmt.Sprintf("%d drift(s): %d breaking, %d non-breaking, compatible=%v",
		len(r.Entries), r.Breaking, r.NonBreak, r.Compatible)
}

// AffectedColumns returns the names of columns that have drift.
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
