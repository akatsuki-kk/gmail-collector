package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/akatsuki-kk/gmail-collector/internal/app"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError()
	}

	switch args[0] {
	case "run":
		return runCommand(args[1:])
	case "-h", "--help", "help":
		printUsage()
		return nil
	default:
		return usageError()
	}
}

func runCommand(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	configPath := fs.String("config", "", "path to config YAML")
	outputPath := fs.String("output", "", "path to write JSON output")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *configPath == "" {
		return errors.New("--config is required")
	}

	runner := app.Runner{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	return runner.Run(context.Background(), app.RunOptions{
		ConfigPath: *configPath,
		OutputPath: *outputPath,
	})
}

func usageError() error {
	printUsage()
	return errors.New("invalid command")
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  gmail-collector run --config path/to/config.yaml [--output result.json]")
}
