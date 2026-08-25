package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"dq-pipeline/internal/clean"
	"dq-pipeline/internal/parse"
	"dq-pipeline/internal/quality"
)

type Config struct {
	Addr string
}

func New(cfg Config) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/evaluate", handleEvaluate)
	mux.HandleFunc("/api/clean", handleClean)
	mux.HandleFunc("/api/profile", handleProfile)
	return mux
}

func ListenAndServe(cfg Config) error {
	mux := New(cfg)
	return http.ListenAndServe(cfg.Addr, mux)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type tableInput struct {
	Header []string   `json:"header"`
	Rows   [][]string `json:"rows"`
}

type ruleInput struct {
	Column   string  `json:"column"`
	Kind     string  `json:"kind"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Pattern  string  `json:"pattern"`
	Critical bool    `json:"critical"`
}

type evaluateRequest struct {
	Table tableInput  `json:"table"`
	Rules []ruleInput `json:"rules"`
}

type violationOutput struct {
	Rule   int    `json:"rule"`
	Column string `json:"column"`
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

type evaluateResponse struct {
	TotalRows   int               `json:"total_rows"`
	TotalChecks int               `json:"total_checks"`
	Score       float64           `json:"score"`
	ByColumn    map[string]int    `json:"by_column"`
	Violations  []violationOutput `json:"violations"`
}

func handleEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req evaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if len(req.Table.Header) == 0 {
		httpError(w, http.StatusBadRequest, "table header is empty")
		return
	}

	table := &parse.Table{Header: req.Table.Header, Rows: req.Table.Rows}
	rules := toRules(req.Rules)

	rep, err := quality.Evaluate(table, rules)
	if err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp := evaluateResponse{
		TotalRows:   rep.TotalRows,
		TotalChecks: rep.TotalChecks,
		Score:       rep.Score,
		ByColumn:    rep.ByColumn,
	}
	for _, v := range rep.Violations {
		resp.Violations = append(resp.Violations, violationOutput{
			Rule:   v.Rule,
			Column: v.Column,
			Row:    v.Row,
			Reason: v.Reason,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

type cleanRequest struct {
	Table          tableInput  `json:"table"`
	Rules          []ruleInput `json:"rules"`
	TrimWhitespace bool        `json:"trim_whitespace"`
	DropCritical   bool        `json:"drop_critical"`
}

type cleanResponse struct {
	Header      []string   `json:"header"`
	Rows        [][]string `json:"rows"`
	RowsDropped int        `json:"rows_dropped"`
}

func handleClean(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req cleanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if len(req.Table.Header) == 0 {
		httpError(w, http.StatusBadRequest, "table header is empty")
		return
	}

	table := &parse.Table{Header: req.Table.Header, Rows: req.Table.Rows}
	rules := toRules(req.Rules)
	rep, err := quality.Evaluate(table, rules)
	if err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}

	cleaned := clean.Clean(table, rules, rep, clean.Options{
		TrimWhitespace: req.TrimWhitespace,
		DropCritical:   req.DropCritical,
	})

	cleaned = HoldAPIClean(cleaned)
	resp := cleanResponse{
		Header:      cleaned.Header,
		Rows:        cleaned.Rows,
		RowsDropped: len(table.Rows) - len(cleaned.Rows),
	}
	writeJSON(w, http.StatusOK, resp)
}

type profileResponse struct {
	TotalRows    int             `json:"total_rows"`
	TotalColumns int             `json:"total_columns"`
	Columns      []columnProfile `json:"columns"`
}

type columnProfile struct {
	Name       string  `json:"name"`
	NonEmpty   int     `json:"non_empty"`
	EmptyCount int     `json:"empty_count"`
	Distinct   int     `json:"distinct"`
	FillRate   float64 `json:"fill_rate"`
}

func handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req struct {
		Table tableInput `json:"table"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if len(req.Table.Header) == 0 {
		httpError(w, http.StatusBadRequest, "table header is empty")
		return
	}

	nCols := len(req.Table.Header)
	nRows := len(req.Table.Rows)
	cols := make([]columnProfile, nCols)
	for ci := 0; ci < nCols; ci++ {
		cp := columnProfile{Name: req.Table.Header[ci]}
		distinct := make(map[string]struct{})
		for _, row := range req.Table.Rows {
			val := ""
			if ci < len(row) {
				val = strings.TrimSpace(row[ci])
			}
			if val != "" {
				cp.NonEmpty++
				distinct[val] = struct{}{}
			} else {
				cp.EmptyCount++
			}
		}
		cp.Distinct = len(distinct)
		if nRows > 0 {
			cp.FillRate = float64(cp.NonEmpty) / float64(nRows)
		}
		cols[ci] = cp
	}

	resp := profileResponse{
		TotalRows:    nRows,
		TotalColumns: nCols,
		Columns:      cols,
	}
	writeJSON(w, http.StatusOK, resp)
}

func toRules(inputs []ruleInput) []quality.Rule {
	rules := make([]quality.Rule, len(inputs))
	for i, ri := range inputs {
		rules[i] = quality.Rule{
			Column:   ri.Column,
			Kind:     ri.Kind,
			Min:      ri.Min,
			Max:      ri.Max,
			Pattern:  ri.Pattern,
			Critical: ri.Critical,
		}
	}
	return rules
}

func httpError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func ParsePort(addr string) int {
	parts := strings.Split(addr, ":")
	if len(parts) < 2 {
		return 0
	}
	p, _ := strconv.Atoi(parts[len(parts)-1])
	return p
}

func FormatAddr(addr string) string {
	port := ParsePort(addr)
	if port == 0 {
		return addr
	}
	return fmt.Sprintf("http://0.0.0.0:%d", port)
}
