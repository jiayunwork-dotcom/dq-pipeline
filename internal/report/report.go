package report

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"

	"dq-pipeline/internal/parse"
	"dq-pipeline/internal/quality"
)

func WriteReport(w io.Writer, rep quality.Report) (err error) {
	bw := bufio.NewWriter(w)
	defer func() {
		if ferr := bw.Flush(); ferr != nil && err == nil {
			err = ferr
		}
	}()
	fmt.Fprintf(bw, "total_rows: %d\n", rep.TotalRows)
	fmt.Fprintf(bw, "total_checks: %d\n", rep.TotalChecks)
	fmt.Fprintf(bw, "violations: %d\n", len(rep.Violations))
	fmt.Fprintf(bw, "score: %.3f\n", rep.Score)
	for col, c := range rep.ByColumn {
		fmt.Fprintf(bw, "column %s: %d violations\n", col, c)
	}
	for _, v := range rep.Violations {
		fmt.Fprintf(bw, "  row %d rule %d col %s: %s\n", v.Row, v.Rule, v.Column, v.Reason)
	}
	return nil
}

func WriteCleaned(w io.Writer, t *parse.Table, delim rune) (err error) {
	bw := bufio.NewWriter(w)
	defer func() {
		if ferr := bw.Flush(); ferr != nil && err == nil {
			err = ferr
		}
	}()
	cw := csv.NewWriter(bw)
	cw.Comma = delim
	if err := cw.Write(t.Header); err != nil {
		return err
	}
	for _, row := range t.Rows {
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}
	return nil
}
