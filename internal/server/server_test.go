package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func sampleTable() tableInput {
	return tableInput{
		Header: []string{"name", "age", "email"},
		Rows: [][]string{
			{"Alice", "30", "alice@example.com"},
			{"Bob", "25", "bob@example.com"},
			{"", "abc", "invalid"},
			{"Charlie", "40", "charlie@example.com"},
			{"Diana", "35", "diana@example.com"},
		},
	}
}

func sampleRules() []ruleInput {
	return []ruleInput{
		{Column: "name", Kind: "notnull", Critical: true},
		{Column: "age", Kind: "range", Min: 0, Max: 150},
		{Column: "email", Kind: "regex", Pattern: `^[^@]+@[^@]+\.[^@]+$`},
	}
}

func TestHealthEndpoint(t *testing.T) {
	mux := New(Config{Addr: ":8080"})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestEvaluateEndpoint(t *testing.T) {
	mux := New(Config{Addr: ":8080"})
	payload := evaluateRequest{Table: sampleTable(), Rules: sampleRules()}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/evaluate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp evaluateResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.TotalRows != 5 {
		t.Errorf("expected 5 rows, got %d", resp.TotalRows)
	}
	if resp.Score >= 1.0 {
		t.Errorf("expected score < 1.0 due to violations, got %.4f", resp.Score)
	}
}

func TestEvaluateEndpoint_EmptyHeader(t *testing.T) {
	mux := New(Config{Addr: ":8080"})
	body := []byte(`{"table":{"header":[],"rows":[]},"rules":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/evaluate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCleanEndpoint(t *testing.T) {
	mux := New(Config{Addr: ":8080"})
	payload := cleanRequest{
		Table:          sampleTable(),
		Rules:          sampleRules(),
		TrimWhitespace: true,
		DropCritical:   true,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/clean", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp cleanResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.RowsDropped < 1 {
		t.Errorf("expected at least 1 row dropped, got %d", resp.RowsDropped)
	}
}

func TestProfileEndpoint(t *testing.T) {
	mux := New(Config{Addr: ":8080"})
	payload := map[string]interface{}{"table": sampleTable()}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/profile", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp profileResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.TotalColumns != 3 {
		t.Errorf("expected 3 columns, got %d", resp.TotalColumns)
	}
	if resp.TotalRows != 5 {
		t.Errorf("expected 5 rows, got %d", resp.TotalRows)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	mux := New(Config{Addr: ":8080"})
	endpoints := []string{"/api/evaluate", "/api/clean", "/api/profile"}
	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodGet, ep, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: expected 405, got %d", ep, rec.Code)
		}
	}
}

func TestParsePort(t *testing.T) {
	if p := ParsePort(":8080"); p != 8080 {
		t.Errorf("expected 8080, got %d", p)
	}
}
