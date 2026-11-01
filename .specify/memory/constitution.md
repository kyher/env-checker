# env-checker Constitution

## Core Principles

### I. Secrets Never Leave the File (NON-NEGOTIABLE)

`.env` files routinely hold credentials. The tool compares **keys**, not values.

- The tool MUST NOT print, log, or otherwise emit any value read from a `.env` file, in any
  output mode (human, JSON, verbose, debug, error messages, panics).
- Findings MAY reference key names, file paths, and line numbers only.
- Where a value property matters (e.g. "value is empty"), the tool MUST report the property,
  never the value itself.
- Values read from `.env.example` are treated as non-secret but are still not echoed unless a
  feature specification explicitly requires it.
- Every output path MUST have a test asserting that a sentinel secret value does not appear in
  the output.

Rationale: a checker run in CI logs or shared terminals must never become a leak vector.

### II. CLI-First, Script-Friendly

env-checker is a command-line tool meant to run equally well by hand and in CI.

- Input via flags/arguments (defaulting to `.env.example` and `.env` in the working
  directory); results to stdout; diagnostics and errors to stderr.
- Exit codes are part of the public contract and MUST be documented and stable:
  `0` = no issues, `1` = issues found, `2` = usage or I/O error.
- The tool MUST offer a human-readable format (default) and a machine-readable JSON format.
  JSON output schema changes follow semantic versioning.
- Behaviour MUST NOT depend on interactive input; no prompts.

Rationale: predictable exit codes and structured output let the tool gate pipelines and
pre-commit hooks without wrappers.

### III. Test-First (NON-NEGOTIABLE)

- Tests are written before implementation and MUST fail before the implementation is added
  (Red → Green → Refactor).
- Core comparison and parsing logic MUST be covered by table-driven unit tests.
- End-to-end CLI behaviour (exit codes, stdout/stderr content, JSON shape) MUST be covered by
  tests that run the built command against fixture files under `testdata/`.
- Every reported bug MUST get a regression test before it is fixed.

Rationale: the tool's value is trust in its verdicts; untested detection logic erodes that
trust quickly.

### IV. Deterministic, Actionable Findings

- Identical inputs MUST produce byte-identical output (stable ordering, no timestamps or
  map-iteration randomness).
- Every finding MUST carry: a category (e.g. missing, unexpected, likely misspelling), the key
  name, and the file and line number where applicable.
- Heuristic findings (such as suspected misspellings) MUST include the suggested intended key
  and MUST be distinguishable from certain findings, so users can tell "definitely wrong"
  from "probably wrong".
- Heuristics MUST be documented and their thresholds covered by tests.

Rationale: a finding the user cannot locate or act on is noise.

### V. Simplicity & Standard Library First

- Prefer the Go standard library. Each third-party dependency MUST be justified in the plan's
  Complexity Tracking (what it provides, why stdlib is insufficient).
- The tool MUST build to a single static binary with no runtime dependencies.
- Packages are introduced only when they have a clear, testable responsibility (e.g. parsing,
  comparison, reporting); no speculative abstractions (YAGNI).

Rationale: a small, dependency-light tool is easy to audit, install, and trust with secrets.

## Technical Constraints

- **Language**: Go, targeting the current stable release; the minimum version is pinned in
  `go.mod`.
- **Layout**: idiomatic Go module layout — `cmd/` for the entrypoint, `internal/` for
  non-public packages, `testdata/` for fixtures.
- **Platforms**: MUST work on Linux, macOS, and Windows; parsing MUST handle both LF and CRLF
  line endings and a UTF-8 BOM.
- **Parsing**: dotenv parsing MUST tolerate common real-world syntax (comments, blank lines,
  `export` prefixes, quoted values) and MUST report unparseable lines as findings with line
  numbers rather than crashing.
- **Errors**: errors are returned and wrapped with context (`fmt.Errorf("...: %w", err)`);
  `panic` is reserved for programmer errors, never for bad input.

## Development Workflow & Quality Gates

Every change MUST pass, locally and in CI, before merge:

1. `gofmt` (no diff) and `go vet ./...` clean.
2. `staticcheck` or `golangci-lint` clean, with configuration committed to the repo.
3. `go test -race ./...` passing.
4. Secret-leak tests (Principle I) passing for every output mode.

Feature work follows the Spec Kit flow: specify → plan → tasks → implement. Each plan MUST
include a Constitution Check confirming compliance with all five principles, and any
deviation MUST be recorded with justification in Complexity Tracking.

## Governance

- This constitution supersedes other project practices and conventions where they conflict.
- Amendments are made via `/speckit-constitution`, recorded in a commit that states the
  version change and rationale, and reviewed like any code change.
- Versioning follows semantic versioning:
  - **MAJOR**: a principle is removed or redefined in a backward-incompatible way.
  - **MINOR**: a principle or section is added, or guidance is materially expanded.
  - **PATCH**: clarifications and wording fixes with no change in meaning.
- All reviews MUST verify compliance with the Core Principles; added complexity MUST be
  justified against Principle V.

**Version**: 1.0.0 | **Ratified**: 2026-11-01 | **Last Amended**: 2026-11-01
