// Command brawl-validate runs campaign author checks and saves their reports.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
)

type commandRunner func([]string, io.Writer, io.Writer) error

type checkResult struct {
	Name      string   `json:"name"`
	Arguments []string `json:"go_arguments"`
	Output    string   `json:"output"`
	Passed    bool     `json:"passed"`
	Error     string   `json:"error,omitempty"`
}

func runChecks(ctx context.Context, dir string, enemies, bosses int, seconds float64, seed string, run commandRunner, diagnostics io.Writer) (bool, error) {
	checks := []checkResult{
		{Name: "Combat engine regressions", Arguments: []string{"test", "./internal/rift", "-count=1"}, Output: "engine-tests.txt"},
		{Name: "Population and reward budgets", Arguments: []string{"run", "./cmd/brawl-population-report", "-format=json", "-max-enemies=" + strconv.Itoa(enemies), "-max-bosses=" + strconv.Itoa(bosses)}, Output: "population.json"},
		{Name: "Walking reachability", Arguments: []string{"run", "./cmd/brawl-reachability-report", "-seed=" + seed}, Output: "reachability.json"},
		{Name: "Hazard route intervals", Arguments: []string{"run", "./cmd/brawl-hazard-report", "-seconds=" + strconv.FormatFloat(seconds, 'g', -1, 64)}, Output: "hazards.json"},
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, err
	}
	for i := range checks {
		checks[i].Error = "not run"
	}
	writeSummary := func() error {
		raw, err := json.MarshalIndent(checks, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "summary.json"), append(raw, '\n'), 0644)
	}
	// Invalidate a previous run before any output file is replaced. Interrupted
	// runs must never leave stale successful evidence in the report directory.
	if err := writeSummary(); err != nil {
		return false, err
	}
	passed := true
	for i := range checks {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		check := &checks[i]
		_, _ = fmt.Fprintln(diagnostics, "Checking:", check.Name)
		file, err := os.Create(filepath.Join(dir, check.Output))
		if err != nil {
			return false, err
		}
		runErr := run(check.Arguments, file, diagnostics)
		closeErr := file.Close()
		if runErr == nil {
			runErr = closeErr
		}
		check.Error = ""
		check.Passed = runErr == nil
		if runErr != nil {
			check.Error = runErr.Error()
			passed = false
			_, _ = fmt.Fprintln(diagnostics, "FAILED:", check.Name, runErr)
		} else {
			_, _ = fmt.Fprintln(diagnostics, "PASS:", check.Name)
		}
		if err := writeSummary(); err != nil {
			return false, err
		}
	}
	return passed, nil
}

func execute(ctx context.Context, args []string, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("brawl-validate", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	dir := flags.String("out-dir", ".tmp/brawl-validation", "directory for reports; existing report files are replaced")
	enemies := flags.Int("max-enemies", 8, "maximum planned enemies per room")
	bosses := flags.Int("max-bosses", 1, "maximum planned bosses per room")
	seconds := flags.Float64("hazard-seconds", 30, "hazard analysis horizon in (0, 3600] seconds")
	seed := flags.String("seed", "author-reachability", "walking-reachability encounter seed")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *dir == "" || *enemies < 1 || *bosses < 0 || math.IsNaN(*seconds) || math.IsInf(*seconds, 0) || *seconds <= 0 || *seconds > 3600 {
		_, _ = fmt.Fprintln(diagnostics, "Invalid options; use -h for help.")
		return 2
	}
	module, err := os.ReadFile("go.mod")
	if err != nil || !strings.HasPrefix(strings.TrimSpace(string(module)), "module ts3news") {
		_, _ = fmt.Fprintln(diagnostics, "Run this command from the ts3news repository root.")
		return 2
	}
	runner := func(args []string, out, stderr io.Writer) error {
		command := exec.CommandContext(ctx, "go", args...)
		command.Stdout = out
		command.Stderr = stderr
		return command.Run()
	}
	passed, err := runChecks(ctx, *dir, *enemies, *bosses, *seconds, *seed, runner, diagnostics)
	if err != nil {
		_, _ = fmt.Fprintln(diagnostics, err)
		return 2
	}
	_, _ = fmt.Fprintln(diagnostics, "Reports:", *dir)
	if !passed {
		return 1
	}
	return 0
}
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	os.Exit(execute(ctx, os.Args[1:], os.Stderr))
}
