package report

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"dq-pipeline/internal/quality"
)

type JSONReport struct {
	Timestamp   string           `json:"timestamp"`
	TotalRows   int              `json:"total_rows"`
	TotalChecks int              `json:"total_checks"`
	Violations  int              `json:"violations"`
	Score       float64          `json:"score"`
	Grade       string           `json:"grade"`
	ByColumn    map[string]int   `json:"by_column"`
	Details     []ViolationEntry `json:"details,omitempty"`
}

type ViolationEntry struct {
	Rule   int    `json:"rule"`
	Column string `json:"column"`
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

func WriteJSONReport(w io.Writer, rep quality.Report) error {
	jr := &JSONReport{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		TotalRows:   rep.TotalRows,
		TotalChecks: rep.TotalChecks,
		Violations:  len(rep.Violations),
		Score:       rep.Score,
		Grade:       scoreGrade(rep.Score),
		ByColumn:    rep.ByColumn,
	}
	for _, v := range rep.Violations {
		jr.Details = append(jr.Details, ViolationEntry{
			Rule:   v.Rule,
			Column: v.Column,
			Row:    v.Row,
			Reason: v.Reason,
		})
	}
	data, err := json.MarshalIndent(jr, "", "  ")
	if err != nil {
		return fmt.Errorf("json report: marshal: %w", err)
	}
	_, err = w.Write(data)
	return err
}

func scoreGrade(score float64) string {
	switch {
	case score >= 0.95:
		return "A"
	case score >= 0.85:
		return "B"
	case score >= 0.70:
		return "C"
	case score >= 0.50:
		return "D"
	default:
		return "F"
	}
}

type ReportComparison struct {
	ScoreDelta     float64  `json:"score_delta"`
	ViolationDelta int      `json:"violation_delta"`
	Improved       bool     `json:"improved"`
	NewIssues      []string `json:"new_issues,omitempty"`
	Resolved       []string `json:"resolved,omitempty"`
}

func CompareJSON(prev, curr *JSONReport) *ReportComparison {
	cmp := &ReportComparison{
		ScoreDelta:     curr.Score - prev.Score,
		ViolationDelta: curr.Violations - prev.Violations,
		Improved:       curr.Score > prev.Score,
	}

	for col, count := range curr.ByColumn {
		prevCount := prev.ByColumn[col]
		if count > prevCount {
			cmp.NewIssues = append(cmp.NewIssues, col)
		}
	}

	for col, prevCount := range prev.ByColumn {
		currCount := curr.ByColumn[col]
		if currCount < prevCount {
			cmp.Resolved = append(cmp.Resolved, col)
		}
	}
	return cmp
}

func ParseJSONReport(r io.Reader) (*JSONReport, error) {
	var jr JSONReport
	dec := json.NewDecoder(r)
	if err := dec.Decode(&jr); err != nil {
		return nil, fmt.Errorf("parse json report: %w", err)
	}
	return &jr, nil
}
