package transform

import (
	"strings"
	"testing"

	"dq-pipeline/internal/parse"
)

func table(csv string) *parse.Table {
	t, err := parse.Parse(strings.NewReader(csv), ',')
	if err != nil {
		panic(err)
	}
	return t
}

func TestRename(t *testing.T) {
	tbl := table("name,age\nAlice,30\n")
	out, err := Apply(tbl, []Op{{Kind: "rename", Column: "name", NewName: "full_name"}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Header[0] != "full_name" {
		t.Errorf("header[0] = %q, want full_name", out.Header[0])
	}
	// original unchanged
	if tbl.Header[0] != "name" {
		t.Error("original table modified")
	}
}

func TestToLower(t *testing.T) {
	tbl := table("city\nNEW YORK\nLONDON\n")
	out, _ := Apply(tbl, []Op{{Kind: "to_lower", Column: "city"}})
	if out.Rows[0][0] != "new york" {
		t.Errorf("got %q, want new york", out.Rows[0][0])
	}
}

func TestToUpper(t *testing.T) {
	tbl := table("code\nabc\ndef\n")
	out, _ := Apply(tbl, []Op{{Kind: "to_upper", Column: "code"}})
	if out.Rows[0][0] != "ABC" {
		t.Errorf("got %q", out.Rows[0][0])
	}
}

func TestToInt(t *testing.T) {
	tbl := table("val\n3.7\n2.0\n")
	out, _ := Apply(tbl, []Op{{Kind: "to_int", Column: "val"}})
	if out.Rows[0][0] != "3" {
		t.Errorf("got %q, want 3", out.Rows[0][0])
	}
	if out.Rows[1][0] != "2" {
		t.Errorf("got %q, want 2", out.Rows[1][0])
	}
}

func TestToFloat(t *testing.T) {
	tbl := table("val\n42\n3.14\n")
	out, _ := Apply(tbl, []Op{{Kind: "to_float", Column: "val"}})
	if out.Rows[0][0] != "42" {
		t.Errorf("got %q", out.Rows[0][0])
	}
}

func TestTrim(t *testing.T) {
	tbl := table("x\n  hello  \n  world \n")
	out, _ := Apply(tbl, []Op{{Kind: "trim", Column: "x"}})
	if out.Rows[0][0] != "hello" {
		t.Errorf("got %q", out.Rows[0][0])
	}
}

func TestConcat(t *testing.T) {
	tbl := table("first,last\nJohn,Doe\nJane,Smith\n")
	out, _ := Apply(tbl, []Op{{Kind: "concat", Column: "full", Columns: []string{"first", "last"}, Sep: " "}})
	if len(out.Header) != 3 {
		t.Fatalf("headers = %d", len(out.Header))
	}
	if out.Header[2] != "full" {
		t.Errorf("new col = %q", out.Header[2])
	}
	if out.Rows[0][2] != "John Doe" {
		t.Errorf("concat = %q", out.Rows[0][2])
	}
}

func TestSubstr(t *testing.T) {
	tbl := table("s\nabcdef\nhi\n")
	out, _ := Apply(tbl, []Op{{Kind: "substr", Column: "s", Start: 1, End: 4}})
	if out.Rows[0][0] != "bcd" {
		t.Errorf("got %q, want bcd", out.Rows[0][0])
	}
	// short string: end clamped
	if out.Rows[1][0] != "i" {
		t.Errorf("got %q, want i", out.Rows[1][0])
	}
}

func TestReplace(t *testing.T) {
	tbl := table("msg\nhello world\nfoo bar\n")
	out, _ := Apply(tbl, []Op{{Kind: "replace", Column: "msg", Old: "o", New: "0"}})
	if out.Rows[0][0] != "hell0 w0rld" {
		t.Errorf("got %q", out.Rows[0][0])
	}
}

func TestFilterNotempty(t *testing.T) {
	tbl := table("name,val\na,1\n,2\nb,\nc,3\n")
	out, _ := Apply(tbl, []Op{{Kind: "filter", Column: "name", Predicate: "notempty"}})
	if len(out.Rows) != 3 {
		t.Errorf("rows = %d, want 3", len(out.Rows))
	}
}

func TestFilterGT(t *testing.T) {
	tbl := table("score\n10\n50\n90\n30\n")
	out, _ := Apply(tbl, []Op{{Kind: "filter", Column: "score", Predicate: "gt:40"}})
	if len(out.Rows) != 2 {
		t.Errorf("rows = %d, want 2 (50,90)", len(out.Rows))
	}
}

func TestFilterLT(t *testing.T) {
	tbl := table("x\n1\n5\n10\n")
	out, _ := Apply(tbl, []Op{{Kind: "filter", Column: "x", Predicate: "lt:6"}})
	if len(out.Rows) != 2 {
		t.Errorf("rows = %d, want 2", len(out.Rows))
	}
}

func TestFilterEQ(t *testing.T) {
	tbl := table("status\nactive\ninactive\nactive\n")
	out, _ := Apply(tbl, []Op{{Kind: "filter", Column: "status", Predicate: "eq:active"}})
	if len(out.Rows) != 2 {
		t.Errorf("rows = %d, want 2", len(out.Rows))
	}
}

func TestFilterContains(t *testing.T) {
	tbl := table("email\nfoo@bar.com\ntest@example.org\nhello@bar.com\n")
	out, _ := Apply(tbl, []Op{{Kind: "filter", Column: "email", Predicate: "contains:bar.com"}})
	if len(out.Rows) != 2 {
		t.Errorf("rows = %d, want 2", len(out.Rows))
	}
}

func TestDedup(t *testing.T) {
	tbl := table("id,val\n1,a\n2,b\n1,c\n3,d\n2,e\n")
	out, _ := Apply(tbl, []Op{{Kind: "dedup", Column: "id"}})
	if len(out.Rows) != 3 {
		t.Errorf("rows = %d, want 3 (deduped by id)", len(out.Rows))
	}
}

func TestSelect(t *testing.T) {
	tbl := table("a,b,c\n1,2,3\n4,5,6\n")
	out, _ := Apply(tbl, []Op{{Kind: "select", Columns: []string{"c", "a"}}})
	if len(out.Header) != 2 {
		t.Fatalf("headers = %d", len(out.Header))
	}
	if out.Header[0] != "c" || out.Header[1] != "a" {
		t.Errorf("headers = %v", out.Header)
	}
	if out.Rows[0][0] != "3" || out.Rows[0][1] != "1" {
		t.Errorf("row = %v", out.Rows[0])
	}
}

func TestApplyMultiple(t *testing.T) {
	tbl := table("name,age\n  Alice ,30\n Bob ,25\n")
	ops := []Op{
		{Kind: "trim", Column: "name"},
		{Kind: "to_upper", Column: "name"},
	}
	out, _ := Apply(tbl, ops)
	if out.Rows[0][0] != "ALICE" {
		t.Errorf("got %q, want ALICE", out.Rows[0][0])
	}
}

func TestApplyMissingColumn(t *testing.T) {
	tbl := table("a\n1\n")
	_, err := Apply(tbl, []Op{{Kind: "to_lower", Column: "missing"}})
	if err == nil {
		t.Error("expected error for missing column")
	}
}

func TestApplyUnknownKind(t *testing.T) {
	tbl := table("a\n1\n")
	_, err := Apply(tbl, []Op{{Kind: "unknown_op", Column: "a"}})
	if err == nil {
		t.Error("expected error for unknown kind")
	}
}

func TestFilterBadPredicate(t *testing.T) {
	tbl := table("a\n1\n")
	_, err := Apply(tbl, []Op{{Kind: "filter", Column: "a", Predicate: "badpred"}})
	if err == nil {
		t.Error("expected error for bad predicate")
	}
}
