package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dq-pipeline/internal/parse"
)

func tbl(csv string) *parse.Table {
	t, _ := parse.Parse(strings.NewReader(csv), ',')
	return t
}

func TestInferAllInts(t *testing.T) {
	data := tbl("id,count\n1,100\n2,200\n3,300\n")
	s := Infer(data, nil)
	if len(s.Columns) != 2 {
		t.Fatalf("columns = %d", len(s.Columns))
	}
	if s.Columns[0].Type != TypeInt {
		t.Errorf("id type = %s, want int", s.Columns[0].Type)
	}
	if s.Columns[1].Type != TypeInt {
		t.Errorf("count type = %s, want int", s.Columns[1].Type)
	}
}

func TestInferFloats(t *testing.T) {
	data := tbl("score\n3.14\n2.71\n1.0\n")
	s := Infer(data, nil)
	if s.Columns[0].Type != TypeFloat {
		t.Errorf("score type = %s, want float", s.Columns[0].Type)
	}
}

func TestInferBooleans(t *testing.T) {
	data := tbl("flag\ntrue\nfalse\ntrue\n")
	s := Infer(data, nil)
	if s.Columns[0].Type != TypeBool {
		t.Errorf("flag type = %s, want bool", s.Columns[0].Type)
	}
}

func TestInferDates(t *testing.T) {
	data := tbl("dt\n2024-01-01\n2024-06-15\n2024-12-31\n")
	s := Infer(data, nil)
	if s.Columns[0].Type != TypeDate {
		t.Errorf("dt type = %s, want date", s.Columns[0].Type)
	}
}

func TestInferNullable(t *testing.T) {
	data := tbl("name,age\nAlice,30\n,25\nBob,\n")
	s := Infer(data, nil)
	if !s.Columns[0].Nullable {
		t.Error("name should be nullable (has empty)")
	}
	if !s.Columns[1].Nullable {
		t.Error("age should be nullable")
	}
}

func TestInferMixedTypeDefaultsToString(t *testing.T) {
	data := tbl("val\n123\nabc\n456\n")
	s := Infer(data, nil)
	if s.Columns[0].Type != TypeString {
		t.Errorf("mixed column type = %s, want string", s.Columns[0].Type)
	}
}

func TestInferSampleSize(t *testing.T) {
	data := tbl("num\n1\n2\n3\nabc\n")
	s := Infer(data, &InferOptions{SampleSize: 3})
	if s.Columns[0].Type != TypeInt {
		t.Errorf("sampled type = %s, want int (only first 3 seen)", s.Columns[0].Type)
	}
}

func TestValidateConformingData(t *testing.T) {
	data := tbl("id,name\n1,Alice\n2,Bob\n")
	s := &Schema{Columns: []Column{
		{Name: "id", Type: TypeInt, Nullable: false},
		{Name: "name", Type: TypeString, Nullable: false},
	}}
	viols := Validate(data, s)
	if len(viols) != 0 {
		t.Errorf("violations = %d, want 0", len(viols))
	}
}

func TestValidateNullInNonNullable(t *testing.T) {
	data := &parse.Table{
		Header: []string{"id"},
		Rows:   [][]string{{"1"}, {""}, {"3"}},
	}
	s := &Schema{Columns: []Column{
		{Name: "id", Type: TypeInt, Nullable: false},
	}}
	viols := Validate(data, s)
	if len(viols) != 1 {
		t.Errorf("violations = %d, want 1", len(viols))
	}
}

func TestValidateTypeMismatch(t *testing.T) {
	data := tbl("num\n42\nabc\n7\n")
	s := &Schema{Columns: []Column{
		{Name: "num", Type: TypeInt, Nullable: false},
	}}
	viols := Validate(data, s)
	if len(viols) != 1 {
		t.Errorf("violations = %d, want 1 (abc not int)", len(viols))
	}
}

func TestSaveAndLoadSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.json")

	s := &Schema{
		Version: 1,
		Columns: []Column{
			{Name: "id", Type: TypeInt, Nullable: false},
			{Name: "name", Type: TypeString, Nullable: true},
		},
	}
	if err := SaveSchema(path, s); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadSchema(path)
	if err != nil {
		t.Fatal(err)
	}
	if !SchemaEqual(s, loaded) {
		t.Errorf("loaded schema != saved schema")
	}
}

func TestLoadSchemaCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	s := &Schema{Columns: []Column{{Name: "x", Type: TypeInt}}}
	SaveSchema(path, s)

	data, _ := os.ReadFile(path)
	data[5] ^= 0xFF
	os.WriteFile(path, data, 0644)

	_, err := LoadSchema(path)
	if err == nil {
		t.Fatal("expected checksum error")
	}
}

func TestLoadSchemaNoChecksum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nocs.json")
	os.WriteFile(path, []byte(`{"columns":[]}`), 0644)

	_, err := LoadSchema(path)
	if err == nil {
		t.Fatal("expected no-checksum error")
	}
}

func TestColumnByName(t *testing.T) {
	s := &Schema{Columns: []Column{
		{Name: "a", Type: TypeInt},
		{Name: "b", Type: TypeString},
	}}
	if c := s.ColumnByName("b"); c == nil || c.Type != TypeString {
		t.Errorf("ColumnByName(b) = %v", c)
	}
	if c := s.ColumnByName("missing"); c != nil {
		t.Errorf("ColumnByName(missing) = %v, want nil", c)
	}
}
