# Implementation Plan: Env Key Check

**Branch**: `001-env-key-check` (work currently on `main`) | **Date**: 2026-11-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/001-env-key-check/spec.md`

## Summary

`env-checker` is a Go command-line tool that compares the keys in a reference file
(`.env.example`) with a target file (`.env`). It reports keys that are:

- missing
- likely misspelled (paired with the intended key)
- unexpected
- duplicated
- empty
- in a line that can't be parsed

How it works:

- A small parser written for this tool gives line numbers for every entry and discards values
  as soon as it reads them. This means no secret can reach any output.
- The comparison step works only with sets of keys. It pairs misspellings using a
  case-insensitive edit distance, with a length-based threshold and a fixed tie-break order.
- The report is printed as text or as JSON, with stable exit codes (0, 1 or 2) for use in CI.

## Technical Context

**Language/Version**: Go, minimum `go 1.25` in `go.mod` (see research R1). Go is not yet
installed on the dev machine.

**Primary Dependencies**: Standard library only. `staticcheck` is used as a Go tool
dependency, which is not built into the binary.

**Storage**: N/A. The tool only reads two local files.

**Testing**: Go's `testing` package:

- table-driven unit tests
- golden-file tests that run the command against fixtures in `testdata/`
- a sentinel-value secret-leak test
- a benchmark

**Target Platform**: Linux, macOS and Windows, built as a single static binary
(`CGO_ENABLED=0`).

**Project Type**: Single-binary command-line tool.

**Performance Goals**: Two files of 1,000 entries each are checked in under 1 second
(SC-005).

**Constraints**:

- No value from the target file is ever printed (FR-012).
- Output is identical for identical inputs (SC-006).
- The tool never prompts for input.

**Scale/Scope**: Two input files, typically fewer than 200 keys each. About 4 packages and
about 1,000 lines of code including tests.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Status |
|---|---|---|
| I. Secrets never leave the file | `dotenv.Entry` has no field for the value (R4). Parse error reasons come from a fixed list and never include line text (R3). A sentinel leak test covers every output format and error path (R8). | ✅ Pass |
| II. Command-line first, script-friendly | Flags have defaults. stdout carries the report and stderr carries diagnostics. Exit codes 0, 1 and 2 are documented in [contracts/cli.md](./contracts/cli.md). There is a text format and a versioned JSON format. The tool never prompts. | ✅ Pass |
| III. Tests first | Tasks will order tests before implementation. Packages get table-driven tests, the command gets golden-file tests against `testdata/`, and every bug gets a regression test first. | ✅ Pass (enforced in tasks) |
| IV. Consistent, actionable findings | Findings follow a total sort order. Every finding has a category, key, file and line(s). Misspellings carry `suggestion` and `certainty: suggestion`. Thresholds are documented (R5) and tested. | ✅ Pass |
| V. Simple, standard library first | There are no third-party runtime dependencies. There are 3 internal packages, each with one job. The binary is static. | ✅ Pass |
| Technical constraints | Uses `cmd/`, `internal/` and `testdata/`. Handles CRLF line endings and a byte-order mark. Bad lines become findings. Errors are wrapped with `%w`, and there is no `panic` on bad input. | ✅ Pass |
| Quality gates | `gofmt`, `go vet`, `go tool staticcheck`, `go test -race`, and the leak test. | ✅ Pass (CI is part of tasks) |

**Re-check after Phase 1 design**: ✅ All gates still pass. The design adds no dependencies,
abstractions or output paths that bypass the parser's value-discarding rule. No violations,
so Complexity Tracking is empty.

## Project Structure

### Documentation (this feature)

```text
specs/001-env-key-check/
├── plan.md              # This file
├── research.md          # Phase 0: decisions R1–R9
├── data-model.md        # Phase 1: entities, comparison and ordering rules
├── quickstart.md        # Phase 1: end-to-end validation guide
├── contracts/
│   ├── cli.md           # Flags, streams, exit codes, text format
│   └── json-report.md   # JSON schema v1
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 (/speckit-tasks; not created here)
```

### Source Code (repository root)

```text
go.mod                         # module github.com/kyher/env-checker, go 1.25, staticcheck as a tool
cmd/env-checker/
├── main.go                    # os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
├── run.go                     # flag parsing, file loading, picking the output format, exit codes
└── run_test.go                # golden-file tests, exit codes, leak test
internal/dotenv/
├── parse.go                   # Parse(path string, r io.Reader) (File, error) → Entries and Errors
└── parse_test.go              # table-driven syntax tests (CRLF, BOM, export, quotes, multi-line)
internal/check/
├── check.go                   # Compare(ref, target dotenv.File) Report; categories; ordering
├── similar.go                 # optimal string alignment distance, threshold, greedy pairing
├── check_test.go
├── similar_test.go            # typo corpus ≥20 / non-typo corpus ≥20 (SC-002, SC-003)
└── bench_test.go              # 1,000 × 1,000 benchmark (SC-005)
internal/report/
├── text.go                    # WriteText(w, Report)
├── json.go                    # WriteJSON(w, Report); schema_version 1
└── report_test.go
testdata/
├── <scenario>/                # clean, missing, misspell, hygiene, crlf_bom, empty_files, …
│   ├── .env.example
│   ├── .env                   # every fixture contains SENTINEL_d0_not_leak
│   ├── text.golden
│   └── json.golden
.github/workflows/ci.yml       # quality gates on ubuntu, macos and windows
```

**Structure Decision**: A single Go module using the standard layout.

- `internal/dotenv` handles parsing.
- `internal/check` handles comparison and has no I/O.
- `internal/report` handles printing.
- `cmd/env-checker` connects the three. It exposes a testable
  `run(args, stdout, stderr) int` function, so the full command can be tested without
  starting a separate process.

## Complexity Tracking

No constitution violations, so nothing to justify.
