# Research: Env Key Check

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Date**: 2026-11-01

There were no open questions left in Technical Context. Each decision below is one where more
than one approach was reasonable.

## R1. Go version and module path

- **Decision**: `go.mod` declares `go 1.25` and module path `github.com/kyher/env-checker`,
  which matches the repository's `origin` remote.
- **Rationale**: Go 1.25 provides everything this needs (`slices`, `cmp`, `encoding/json`,
  `testing` with `t.Context`) and builds on any newer toolchain. Pinning only the minimum
  follows the constitution's "current stable, minimum pinned" rule.
- **Alternatives considered**: Pinning the newest release (this would lock out contributors
  with slightly older toolchains for no benefit). A bare module path such as `env-checker`
  (this would break `go install github.com/...@latest`).
- **Implementation note**: the directive is written as `go 1.25.0`. With `GOTOOLCHAIN=auto`,
  Go can't download a toolchain for a bare `1.25`, but it can for `1.25.0`.

## R2. Dotenv parsing: write our own or use a library

- **Decision**: Write a small parser in `internal/dotenv` using only the standard library.
- **Rationale**:
  - The parser must give line numbers for every entry, record duplicates (not just keep the
    last one), report bad lines and carry on, and never keep values (Principle I).
  - `github.com/joho/godotenv` returns a `map[string]string`. That loses line numbers and
    duplicates, keeps every value in memory, and stops at the first error.
  - `hashicorp/go-envparse` has similar gaps.
  - The grammar we need is small (see R3).
- **Alternatives considered**: godotenv (rejected for the reasons above); wrapping a library
  and parsing a second time for line numbers (two parsers means double the complexity).

## R3. Supported dotenv syntax

- **Decision**: The parser accepts the following, line by line, after removing a leading UTF-8
  byte-order mark and converting CRLF to LF:
  - Blank lines and lines whose first non-space character is `#` are skipped.
  - An optional `export` prefix followed by whitespace.
  - The key must match `[A-Za-z_][A-Za-z0-9_]*`. Whitespace around `=` is allowed.
  - **Unquoted value**: runs to the end of the line. A `#` that has whitespace before it starts
    an inline comment. Surrounding whitespace is trimmed.
  - **Single-quoted value**: taken literally (no escapes) and may span several lines.
    *(Changed in convergence task T043. This was originally same-line only, which conflicted
    with FR-003 and the multi-line edge case.)*
  - **Double-quoted value**: `\"` escapes are allowed and the value may span several lines.
    Lines inside the quotes are part of the entry and are never reported as unparseable.
  - Anything after a closing quote other than whitespace or a `# comment` makes the line
    unparseable.
  - **Unparseable lines** are reported with one of these fixed reasons: `missing "="`,
    `invalid key`, `unterminated quote`, `unexpected text after closing quote`.
    - With an unterminated quote of either kind, the parser resumes at the line after the opening
      line, so one bad quote does not hide the rest of the file.
    - The reason never includes text from the line itself, which could contain a secret.
- **Rationale**: This covers the syntax used by docker compose, godotenv, python-dotenv and
  Node's dotenv, which is what users actually write. Key characters are limited to what
  POSIX shells accept.
- **Alternatives considered**: Expanding `${VAR}` references (out of scope per the spec, and
  not needed when only checking whether a value is empty). Allowing `.` and `-` in keys (no
  shell can export these, so rejecting them is more likely to catch mistakes).

- **Line length**: lines have no length limit, so a very long value such as an inlined
  certificate is parsed as a normal entry.
  *(Convergence task T044 suggested reporting over-long lines with a new `line too long`
  reason. Instead, the 1 MiB limit was removed: a long value is valid, so flagging it would be
  a false finding. Either way the run no longer aborts, and the fixed set of reasons is
  unchanged.)*

## R4. Discarding values early

- **Decision**: The parser works out whether a value is empty and then throws the value away.
  `dotenv.Entry` has no field that holds a value.
- **Rationale**: A value that is never stored cannot be leaked by any output path, log or
  panic. This makes Principle I hold by design, and tests then check that it holds.
- **Alternatives considered**: Keeping values and hiding them when printing (one mistake in
  any output path would leak a secret).

- **Note (convergence task T046)**: T008 named a `check.Keys(f dotenv.File)` export. It was
  not added, because `dotenv.File.Keys()` already returns the unique keys in order of first
  appearance, and the comparison uses that directly.

## R5. Misspelling distance measure and pairing

- **Decision**:
  - **Distance**: optimal string alignment distance (Damerau–Levenshtein limited to adjacent
    swaps), computed on the upper-cased keys.
  - **Thresholds (FR-006)**:
    - Upper-cased keys that are equal count as a case-only match (distance 0).
    - Otherwise, take the length of the shorter key, call it L. If L is under 4, the keys
      never match. If L is 4 to 7, the distance must be at most 1. If L is 8 or more, it must
      be at most 2.
  - **Pairing**:
    1. Build every qualifying pair of (missing key, unexpected key).
    2. Sort the pairs by distance, then by the missing key's line in the reference file, then
       by the unexpected key's line in the target file.
    3. Go through the list in order and accept a pair only if neither key has been used yet.
  - **Speed**: before computing the distance, skip pairs whose lengths differ by more than
    the threshold allows. Only missing × unexpected keys are compared, never the full lists.
- **Rationale**:
  - Swapping two letters is the most common typing slip, and plain Levenshtein counts it as 2
    edits. Optimal string alignment counts it as 1.
  - Upper-casing first lets `databse_url` match `DATABASE_URL`.
  - Accepting the best pairs first, in a fixed order, gives the same result on every run
    (Principle IV, SC-006).
  - The length check keeps the 1,000-entry case well under 1 second (SC-005).
- **Alternatives considered**:
  - Plain Levenshtein: misses some two-letter swaps on short keys.
  - Jaro–Winkler: its scores are hard to explain in a finding or set fixed thresholds for.
  - Optimal assignment (the Hungarian algorithm): too complex for little gain. Ties are rare,
    and the tie-break order is documented.
- **Validation**: Table-driven fixtures of at least 20 real typos and at least 20 pairs of
  genuinely different keys check SC-002 and SC-003.

## R6. Command-line interface and flag parsing

- **Decision**:
  - Use the standard library `flag` package. The flags are `--example`, `--env`, `--format`
    (`text` or `json`) and `--version`.
  - Exit codes: `0` = no findings, `1` = findings, `2` = usage or I/O error.
  - `main` calls `run(args []string, stdout, stderr io.Writer) int`, so the full command can
    be tested in-process.
- **Rationale**: There are only four flags, so a framework such as cobra or urfave/cli is not
  worth the extra dependency (Principle V). `flag` accepts both `-x` and `--x`.
- **Alternatives considered**: cobra (heavy dependency, and subcommands aren't needed);
  positional arguments for the file paths (unclear which file is which, so flags are named).

## R7. Output formats

- **Decision**:
  - **Text** (default) is grouped by category, one line per finding, followed by a summary
    line. There are no ANSI colours.
  - **JSON** is a single object with a `schema_version` field (see
    [contracts/json-report.md](./contracts/json-report.md)). It is always printed on a single
    line and ends with a newline.
  - On exit code 2, nothing is written to stdout and one error line is written to stderr.
- **Rationale**: With no colours there's no need to detect a terminal or support a
  `--no-color` flag, and output is identical everywhere. `schema_version` lets the JSON format
  change under semantic versioning, as the constitution requires.
- **Alternatives considered**: Coloured text (more complexity and output that varies by
  environment); SARIF (too heavy for now, though it could be added later as another format).

## R8. Testing approach

- **Decision**:
  - **Unit tests**: table-driven tests per package.
  - **Golden-file tests**: the command is run against fixture directories under `testdata/`,
    and the output is compared with saved `.golden` files. Running with `-update` rewrites
    the golden files.
  - **Leak test**: every fixture's `.env` contains the sentinel value `SENTINEL_d0_not_leak`.
    The test runs every output format and every error path and asserts the sentinel never
    appears in stdout or stderr.
  - **Speed check**: a benchmark plus a test with a time limit for 1,000 × 1,000 entries.
- **Rationale**: This meets constitution Principle III and the quality gates using only the
  standard library `testing` package.
- **Alternatives considered**: testify (an extra dependency that's barely needed); testscript
  (useful, but another dependency; the in-process `run()` tests cover the same cases).

## R9. Linting

- **Decision**: Use `staticcheck` v0.7.0, pinned as a Go tool dependency with `go get -tool
  honnef.co/go/tools/cmd/staticcheck@v0.7.0`, and run it with `go tool staticcheck ./...`.
  v0.8.x requires Go 1.26 and would raise this module's minimum Go version.
- **Rationale**: It meets the constitution's lint gate. Because it's a tool dependency, it
  never ends up in the built binary and its version is pinned in `go.mod`.
- **Alternatives considered**: golangci-lint (needs a separate install and a larger config).
