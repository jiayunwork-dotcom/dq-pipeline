package schema

import (
	"testing"
)

func TestDetectDriftNoChange(t *testing.T) {
	s := &Schema{Columns: []Column{
		{Name: "id", Type: TypeInt},
		{Name: "name", Type: TypeString},
	}}
	report := DetectDrift(s, s)
	if len(report.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(report.Entries))
	}
	if !report.Compatible {
		t.Error("expected compatible")
	}
}

func TestDetectDriftAddedColumn(t *testing.T) {
	old := &Schema{Columns: []Column{{Name: "id", Type: TypeInt}}}
	new := &Schema{Columns: []Column{
		{Name: "id", Type: TypeInt},
		{Name: "email", Type: TypeString},
	}}
	report := DetectDrift(old, new)
	if report.NonBreak != 1 {
		t.Errorf("NonBreak = %d, want 1", report.NonBreak)
	}
	if !report.Compatible {
		t.Error("adding column should be compatible")
	}
}

func TestDetectDriftRemovedColumn(t *testing.T) {
	old := &Schema{Columns: []Column{
		{Name: "id", Type: TypeInt},
		{Name: "name", Type: TypeString},
	}}
	new := &Schema{Columns: []Column{{Name: "id", Type: TypeInt}}}
	report := DetectDrift(old, new)
	if report.Breaking != 1 {
		t.Errorf("Breaking = %d, want 1", report.Breaking)
	}
	if report.Compatible {
		t.Error("removing column should be incompatible")
	}
}

func TestDetectDriftTypeWidening(t *testing.T) {
	old := &Schema{Columns: []Column{{Name: "val", Type: TypeInt}}}
	new := &Schema{Columns: []Column{{Name: "val", Type: TypeFloat}}}
	report := DetectDrift(old, new)
	if report.Breaking != 0 {
		t.Errorf("Breaking = %d, want 0 (widening)", report.Breaking)
	}
}

func TestDetectDriftTypeNarrowing(t *testing.T) {
	old := &Schema{Columns: []Column{{Name: "val", Type: TypeString}}}
	new := &Schema{Columns: []Column{{Name: "val", Type: TypeInt}}}
	report := DetectDrift(old, new)
	if report.Breaking != 1 {
		t.Errorf("Breaking = %d, want 1 (narrowing)", report.Breaking)
	}
	if report.Compatible {
		t.Error("narrowing should be incompatible")
	}
}

func TestDetectDriftNullableChange(t *testing.T) {
	old := &Schema{Columns: []Column{{Name: "x", Type: TypeInt, Nullable: true}}}
	new := &Schema{Columns: []Column{{Name: "x", Type: TypeInt, Nullable: false}}}
	report := DetectDrift(old, new)
	if report.Breaking != 1 {
		t.Errorf("Breaking = %d, want 1", report.Breaking)
	}
}

func TestDetectDriftNullableToNullable(t *testing.T) {
	old := &Schema{Columns: []Column{{Name: "x", Type: TypeInt, Nullable: false}}}
	new := &Schema{Columns: []Column{{Name: "x", Type: TypeInt, Nullable: true}}}
	report := DetectDrift(old, new)
	if report.Breaking != 0 {
		t.Errorf("Breaking = %d, want 0 (widening nullable)", report.Breaking)
	}
	if !report.Compatible {
		t.Error("widening nullable should be compatible")
	}
}

func TestDriftReportSummary(t *testing.T) {
	report := &DriftReport{Entries: nil}
	if s := report.Summary(); s != "no schema drift detected" {
		t.Errorf("summary = %q", s)
	}
	report.Entries = []DriftEntry{{Column: "x", Kind: DriftRemoved}}
	report.Breaking = 1
	if s := report.Summary(); s == "no schema drift detected" {
		t.Error("expected non-empty summary")
	}
}

func TestAffectedColumns(t *testing.T) {
	report := &DriftReport{Entries: []DriftEntry{
		{Column: "a", Kind: DriftAdded},
		{Column: "b", Kind: DriftRemoved},
		{Column: "a", Kind: DriftTypeChange},
	}}
	cols := report.AffectedColumns()
	if len(cols) != 2 {
		t.Errorf("affected = %v, want 2", cols)
	}
}
