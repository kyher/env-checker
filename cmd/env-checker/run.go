package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/kyher/env-checker/internal/check"
	"github.com/kyher/env-checker/internal/dotenv"
	"github.com/kyher/env-checker/internal/report"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "dev"

// Exit codes (contracts/cli.md).
const (
	exitOK     = 0
	exitIssues = 1
	exitError  = 2
)

const (
	defaultExample = ".env.example"
	defaultEnv     = ".env"
)

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("env-checker", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	examplePath := fs.String("example", defaultExample, "reference `PATH` declaring the expected keys")
	envPath := fs.String("env", defaultEnv, "target `PATH` holding the actual keys")
	format := fs.String("format", "text", "output `FORMAT`: text or json")
	showVersion := fs.Bool("version", false, "print version and exit")
	usage := func(w io.Writer) {
		fmt.Fprintf(w, "Usage: env-checker [--example PATH] [--env PATH] [--format text|json]\n\n"+
			"Compares the keys in a reference file (default %s) against a target file (default %s).\n"+
			"Values are never printed.\n\nFlags:\n", defaultExample, defaultEnv)
		fs.SetOutput(w)
		fs.PrintDefaults()
		fs.SetOutput(io.Discard)
		fmt.Fprintf(w, "\nExit codes: 0 no issues, 1 issues found, 2 error.\n")
	}
	usageErr := func(msg string) int {
		fmt.Fprintf(stderr, "env-checker: %s\n", msg)
		usage(stderr)
		return exitError
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			usage(stdout)
			return exitOK
		}
		return usageErr(err.Error())
	}
	if fs.NArg() > 0 {
		return usageErr(fmt.Sprintf("unexpected argument %q", fs.Arg(0)))
	}
	if *showVersion {
		fmt.Fprintf(stdout, "env-checker %s\n", version)
		return exitOK
	}

	var write func(io.Writer, check.Report) error
	switch *format {
	case "text":
		write = report.WriteText
	case "json":
		write = report.WriteJSON
	default:
		return usageErr(fmt.Sprintf("invalid --format %q: must be text or json", *format))
	}

	ref, err := load(*examplePath)
	if err != nil {
		fmt.Fprintf(stderr, "env-checker: %v\n", err)
		return exitError
	}
	target, err := load(*envPath)
	if err != nil {
		fmt.Fprintf(stderr, "env-checker: %v\n", err)
		return exitError
	}

	r := check.Compare(ref, target)
	if err := write(stdout, r); err != nil {
		fmt.Fprintf(stderr, "env-checker: write report: %v\n", err)
		return exitError
	}
	if r.OK() {
		return exitOK
	}
	return exitIssues
}

func load(path string) (dotenv.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return dotenv.File{}, err // *PathError already reads "open <path>: ..."
	}
	defer f.Close()
	return dotenv.Parse(path, f)
}
