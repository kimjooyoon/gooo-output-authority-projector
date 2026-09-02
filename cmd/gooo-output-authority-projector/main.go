package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-output-authority-projector/internal/conformance"
	"github.com/kimjooyoon/gooo-output-authority-projector/internal/projector"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output, errorOutput io.Writer) error {
	if len(args) == 0 {
		return errors.New("command is required: project or conformance")
	}
	switch args[0] {
	case "project":
		return runProject(args[1:], input, output)
	case "conformance":
		return runConformance(args[1:], output)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runProject(args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("project", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	outputPath := flags.String("output", "", "optional caller-owned receipt path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	var request projector.Request
	if err := json.NewDecoder(input).Decode(&request); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	decision := projector.Project(request)
	encoded, err := projector.EncodeDecision(decision)
	if err != nil {
		return err
	}
	if *outputPath == "" {
		_, err = output.Write(append(encoded, '\n'))
		return err
	}
	if receiptDecision := projector.ValidateReceiptOutput(*outputPath, request.CallerOwnedRoot, request.RepositoryRoot); receiptDecision.Status != projector.StatusClosed {
		return fmt.Errorf("refusing receipt output: %s: %s", receiptDecision.Status, receiptDecision.Reason)
	}
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(*outputPath, append(encoded, '\n'), 0o600)
}

func runConformance(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("conformance", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	callerRoot := flags.String("caller-root", "", "caller-owned temporary root")
	repositoryRoot := flags.String("repository-root", "", "repository root to protect")
	outputPath := flags.String("output", "", "optional caller-owned report path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *callerRoot == "" {
		created, err := os.MkdirTemp("", "gooo-output-authority-projector-")
		if err != nil {
			return err
		}
		*callerRoot = created
	}
	if *repositoryRoot == "" {
		current, err := os.Getwd()
		if err != nil {
			return err
		}
		*repositoryRoot = current
	}
	report, err := conformance.Run(*callerRoot, *repositoryRoot)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if *outputPath == "" {
		_, err = output.Write(encoded)
		return err
	}
	if decision := projector.ValidateReceiptOutput(*outputPath, *callerRoot, *repositoryRoot); decision.Status != projector.StatusClosed {
		return fmt.Errorf("refusing report output: %s: %s", decision.Status, decision.Reason)
	}
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(*outputPath, encoded, 0o600)
}
