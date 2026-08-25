package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"dq-pipeline/internal/clean"
	"dq-pipeline/internal/parse"
	"dq-pipeline/internal/quality"
	"dq-pipeline/internal/report"
	"dq-pipeline/internal/server"
)

type ruleSpec struct {
	Column   string  `json:"column"`
	Kind     string  `json:"kind"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Pattern  string  `json:"pattern"`
	Critical bool    `json:"critical"`
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		runServer(nil)
		return
	}
	if args[0] == "serve" {
		runServer(args[1:])
		return
	}
	if err := run(args); err != nil {
		fmt.Fprintln(os.Stderr, "dq-pipeline:", err)
		os.Exit(1)
	}
}

func runServer(args []string) {
	addr := ":8080"
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-addr" || args[i] == "--addr" {
			addr = args[i+1]
			break
		}
	}
	cfg := server.Config{Addr: addr}
	fmt.Fprintf(os.Stdout, "dq-pipeline server listening on %s\n", server.FormatAddr(addr))
	if err := server.ListenAndServe(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("dq-pipeline", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	input := fs.String("input", "-", "input CSV file ('-' for stdin)")
	delim := fs.String("delim", ",", "field delimiter")
	rulesPath := fs.String("rules", "", "JSON rules file (optional)")
	out := fs.String("out", "", "write cleaned CSV to this path")
	reportPath := fs.String("report", "-", "write quality report ('-' for stdout)")
	drop := fs.Bool("drop", false, "drop rows failing any critical rule")
	trim := fs.Bool("trim", true, "trim whitespace in cleaned output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	d, err := parseDelim(*delim)
	if err != nil {
		return err
	}
	var r io.Reader = os.Stdin
	if *input != "-" {
		f, err := os.Open(*input)
		if err != nil {
			return fmt.Errorf("open input: %w", err)
		}
		defer f.Close()
		r = f
	}
	table, err := parse.Parse(r, d)
	if err != nil {
		return fmt.Errorf("parse input: %w", err)
	}
	rules, err := loadRules(*rulesPath)
	if err != nil {
		return err
	}
	rep, err := quality.Evaluate(table, rules)
	if err != nil {
		return fmt.Errorf("evaluate: %w", err)
	}
	var cleaned *parse.Table
	if *out != "" || *drop || *trim {
		cleaned = clean.Clean(table, rules, rep, clean.Options{
			TrimWhitespace: *trim,
			DropCritical:   *drop,
		})
	}
	if *reportPath == "-" {
		if err := report.WriteReport(os.Stdout, rep); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
	} else {
		rf, err := os.Create(*reportPath)
		if err != nil {
			return fmt.Errorf("create report: %w", err)
		}
		defer rf.Close()
		if err := report.WriteReport(rf, rep); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
	}
	if *out != "" {
		if cleaned == nil {
			cleaned = table
		}
		of, err := os.Create(*out)
		if err != nil {
			return fmt.Errorf("create out: %w", err)
		}
		defer of.Close()
		if err := report.WriteCleaned(of, cleaned, d); err != nil {
			return fmt.Errorf("write cleaned: %w", err)
		}
	}
	return nil
}

func parseDelim(s string) (rune, error) {
	if s == "" {
		return 0, errors.New("delim cannot be empty")
	}
	r := []rune(s)
	if len(r) != 1 {
		return 0, fmt.Errorf("delim must be a single character, got %q", s)
	}
	return r[0], nil
}

func loadRules(path string) ([]quality.Rule, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules: %w", err)
	}
	var specs []ruleSpec
	if err := json.Unmarshal(b, &specs); err != nil {
		return nil, fmt.Errorf("parse rules: %w", err)
	}
	rules := make([]quality.Rule, 0, len(specs))
	for _, s := range specs {
		rules = append(rules, quality.Rule{
			Column:   s.Column,
			Kind:     s.Kind,
			Min:      s.Min,
			Max:      s.Max,
			Pattern:  s.Pattern,
			Critical: s.Critical,
		})
	}
	return rules, nil
}
