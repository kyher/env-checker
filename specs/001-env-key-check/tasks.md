---

description: "Task list for Env Key Check"
---

# Tasks: Env Key Check

**Input**: Design documents from `specs/001-env-key-check/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md),
[data-model.md](./data-model.md), [contracts/cli.md](./contracts/cli.md),
[contracts/json-report.md](./contracts/json-report.md), [quickstart.md](./quickstart.md)

**Tests**: Required. Constitution Principle III makes test-first development mandatory. Within
each phase, test tasks come before implementation tasks and MUST fail before the
implementation task starts.

**Organization**: Tasks are grouped by user story. Each story phase ends with a checkpoint
where the tool is runnable and the story's independent test passes.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1–US4 from spec.md
- All paths are relative to the repository root

## Conventions used by every task

- **Module**: `github.com/kyher/env-checker`. Import internal packages as
  `github.com/kyher/env-checker/internal/<pkg>`.
- **Sentinel**: every fixture `.env` MUST contain at least one entry whose value is
  `SENTINEL_d0_not_leak`.
- **Golden files**: tests in `cmd/env-checker` read fixtures from `../../testdata/<scenario>/`.
  - They use `t.Chdir(dir)` so the default paths `.env.example` and `.env` resolve, and paths
    in the output appear as `.env.example` and `.env`.
  - Tests that call `t.Chdir` MUST NOT call `t.Parallel()`.
  - `-update` rewrites the `*.golden` files.
- **Errors**: wrap with `fmt.Errorf("...: %w", err)`. Never `panic` on input. Never put a
  value, or any text from a line, in an error message.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Module, layout and tooling.

**Prerequisite (manual)**: Go 1.25 or newer must be installed (`go version`). It is not
currently installed on the dev machine.

- [X] T001 Run `go mod init github.com/kyher/env-checker` and set the `go` directive to
  `go 1.25` in `go.mod`. Create the empty directories `cmd/env-checker/`, `internal/dotenv/`,
  `internal/check/`, `internal/report/` and `testdata/`.
- [X] T002 Add staticcheck as a tool dependency with
  `go get -tool honnef.co/go/tools/cmd/staticcheck@latest`, which updates `go.mod` and
  `go.sum`. Verify that `go tool staticcheck ./...` runs.
- [X] T003 [P] Create `.gitignore` containing `/bin/`, `/dist/`, `*.test`, `*.out` and
  `coverage.*`.
- [X] T004 [P] Create `.github/workflows/ci.yml`:
  - Matrix: `ubuntu-latest`, `macos-latest` and `windows-latest`.
  - Use `actions/setup-go` with `go-version-file: go.mod`.
  - Steps, in order:
    1. Fail if `gofmt -l .` produces any output (skip this step on Windows).
    2. `go vet ./...`
    3. `go tool staticcheck ./...`
    4. `go test -race ./...`
  - Set `CGO_ENABLED=1` only for the race step. Build with `CGO_ENABLED=0`.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The parser and the shared finding and report types that every story uses.

**⚠️ CRITICAL**: No user story work starts until this phase is complete.

### Tests (write first, confirm they fail)

- [X] T005 [P] Write table-driven tests in `internal/dotenv/parse_test.go` for
  `Parse(path string, r io.Reader) (File, error)`. Each case gives input text and the expected
  `Entries` (`Key`, `Line`, `Empty`) and `Errors` (`Line`, `Reason`). Cases:
  - **Skipped lines**: blank lines, `# comment`, and `   # indented comment`.
  - **Basic entries**: `KEY=value`, `KEY = value` (spaces around `=`), and `export KEY=value`.
  - **Empty values**: `KEY=`, `KEY=""` and `KEY=''` are all `Empty: true`.
    `KEY=" "` is `Empty: false`.
  - **Inline comments**:
    - `KEY=abc # note` is not empty.
    - `KEY=a#b` is not empty, and the `#` is part of the value.
    - `KEY= # only comment` is `Empty: true`.
  - **Quoted values**:
    - `KEY='it''s'` is an error: `unexpected text after closing quote`.
    - `KEY="a\"b"` is valid.
    - `KEY=a=b=c` is valid.
  - **Multi-line value**: `KEY="line1\nline2\nline3"` spread over 3 physical lines, followed
    by `NEXT=1`. Expect `KEY` at line 1 and `NEXT` at line 4, with no errors.
  - **Duplicates**: `A=1\nA=2` gives two entries, at lines 1 and 2.
  - **Errors, using the exact fixed reason strings**:
    - `this is not valid` → `missing "="`
    - `1KEY=x` → `invalid key`
    - `MY-KEY=x` → `invalid key`
    - `MY.KEY=x` → `invalid key`
    - `KEY="unterminated` followed by `NEXT=1` → an error at line 1 with reason
      `unterminated quote`, and `NEXT` is still parsed at line 2.
    - `KEY='unterminated` → `unterminated quote`
  - **CRLF**: the same input written with `\r\n` gives the same result as with `\n`.
  - **Byte-order mark**: a leading `\xEF\xBB\xBF` is removed, so the first key is `KEY`, not
    `﻿KEY`.
  - **Empty input**: no entries and no errors.
  - **Read failure**: if the `io.Reader` returns an error, `Parse` returns a wrapped error.
  - **No leaks**: no `Reason` string contains any text from the input. Assert this for every
    error case by checking that none of the input tokens appear in the reason.
- [X] T006 [P] Write tests in `internal/check/report_test.go` for the shared types:
  - **Ordering**: `Sort(findings)` orders by Category declaration order (Unparseable,
    Missing, Misspelling, Duplicate, Empty, Unexpected). Then the main location's file, with
    `Report.Reference` before `Report.Target`. Then the main location's line. Then `Key`.
    Build an unordered slice of at least 8 findings covering every category and both files,
    and assert the exact order.
  - **Counts**: `Report.Counts()` returns a count for all 6 categories, including zeros.
  - **OK**: `Report.OK()` returns true if and only if there are no findings.
  - **Category strings**: `Category.String()` returns the JSON strings `unparseable`,
    `missing`, `likely_misspelling`, `duplicate`, `empty` and `unexpected`.
  - **Certainty**: `Category.Certainty()` returns `suggestion` for Misspelling only, and
    `definite` for every other category.

### Implementation

- [X] T007 Implement `internal/dotenv/parse.go` so that T005 passes.
  - **Types**:
    - `Entry{Key string; Line int; Empty bool}`. Add a doc comment: "Entry deliberately has
      no value field (constitution Principle I)."
    - `ParseError{Line int; Reason string}`
    - `File{Path string; Entries []Entry; Errors []ParseError}`
  - **Methods**: `File.Lines(key) []int` returns ascending lines. `File.FirstLine(key) int`.
    `File.Keys() []string` returns unique keys in order of first appearance.
  - **Reason constants**: export `ReasonMissingEquals = "missing \"=\""`,
    `ReasonInvalidKey = "invalid key"`, `ReasonUnterminatedQuote = "unterminated quote"` and
    `ReasonTrailingText = "unexpected text after closing quote"`.
  - **Key pattern**: `^[A-Za-z_][A-Za-z0-9_]*$`.
  - Use `bufio.Scanner` with a 1 MiB maximum token size, as the input is a few lines.
  - Compute `Empty` without keeping the value: track only whether any non-whitespace value
    character was seen.
  - Follow the grammar in research R3 exactly.
- [X] T008 [P] Implement `internal/check/report.go` so that T006 passes.
  - **Category**: `type Category int`, declared in this order: `Unparseable`, `Missing`,
    `Misspelling`, `Duplicate`, `Empty`, `Unexpected`. Add a `Categories()` function that
    returns all six in order. Add `String()` and `Certainty()`.
  - **Location**: `Location{File string; Line int}`.
  - **Finding**: `Finding{Category Category; Key, Suggestion, Reason string; Locations
    []Location}`.
  - **Report**: `Report{Reference, Target string; Findings []Finding}`, with
    `Counts() map[Category]int`, `OK() bool` and an unexported `sort()` that uses
    `slices.SortStableFunc` with the ordering from data-model.md.
  - **Keys**: export `Keys(f dotenv.File) []string` for use by later comparisons.

**Checkpoint**: `go test ./internal/...` passes.

---

## Phase 3: User Story 1 - Find missing keys (Priority: P1) 🎯 MVP

**Goal**: `env-checker`, run in a project directory, reports keys declared in `.env.example`
that are absent from `.env`, using the default text format and exit codes 0, 1 and 2.

**Independent Test**: In `testdata/missing/`, the reference file has `DB_HOST`, `DB_PORT` and
`API_KEY`, and the target file has `DB_HOST` and `API_KEY`. Running `env-checker` prints
`.env.example:2: missing: DB_PORT` and the summary `1 issue: 1 missing`, and exits 1.

### Tests (write first, confirm they fail)

- [X] T009 [P] [US1] Write tests in `internal/check/check_test.go` for
  `Compare(ref, target dotenv.File) Report`:
  - If all keys are present, there are no findings.
  - A single missing key gives one Missing finding with `Locations ==
    [{ref.Path, FirstLine}]`.
  - Multiple missing keys are sorted by their line in the reference file.
  - With an empty reference file and an empty target file, there are no findings.
  - `Report.Reference` and `Report.Target` equal the files' `Path`s.
- [X] T010 [P] [US1] Write tests in `internal/report/report_test.go` for
  `WriteText(w io.Writer, r check.Report) error`, using hand-built Reports:
  - **No findings**: the output is exactly
    `OK: all N keys in <ref> are present in <target>\n`. This needs a key count, so add an
    `Report.ReferenceKeys int` field set by Compare.
  - **One Missing finding**: `.env.example:3: missing: DB_PORT\n\n1 issue: 1 missing\n`
  - **Two Missing findings**: the summary is `2 issues: 2 missing`.
  - **Write failure**: a writer that returns an error makes WriteText return that error.
- [X] T011 [P] [US1] Create fixtures. Every fixture `.env` contains `SENTINEL_d0_not_leak`.
  - `testdata/clean/.env.example`: `DB_HOST=localhost`, `DB_PORT=5432`, `API_KEY=`
  - `testdata/clean/.env`: all three keys, with `API_KEY=SENTINEL_d0_not_leak`
  - `testdata/missing/.env.example`: the same three keys
  - `testdata/missing/.env`: `DB_HOST=db` and `API_KEY=SENTINEL_d0_not_leak`
  - `testdata/no_env/.env.example`: only the reference file, with no `.env`
- [X] T012 [US1] Write `cmd/env-checker/run_test.go`:
  - A `-update` flag (`flag.Bool("update", ...)`).
  - A helper `runIn(t, dir string, args ...string) (stdout, stderr string, code int)`. It
    calls `t.Chdir(dir)` and then `run(args, &outBuf, &errBuf)`.
  - A helper `golden(t, path, got string)` that compares against the golden file, or writes
    it when `-update` is set.
  - **Table test, text format**:
    - `clean`: code 0, `testdata/clean/text.golden`
    - `missing`: code 1, `testdata/missing/text.golden`
  - **Error cases**:
    - `no_env`: code 2, stdout empty, stderr starts with `env-checker: open .env:`
    - An unknown flag `--bogus`: code 2, stderr contains `env-checker:` and the usage text
    - A positional argument `foo`: code 2
  - **Version**: `--version` gives code 0 and stdout `env-checker dev\n`.
  - **`TestNoSecretLeak`**: run every scenario directory under `../../testdata`, with every
    supported `--format` value, and assert that `SENTINEL_d0_not_leak` appears in neither
    stdout nor stderr. Find the scenarios with `os.ReadDir`, so new fixtures are covered
    automatically. In this phase only `text` exists; T036 adds `json`.

### Implementation

- [X] T013 [US1] Implement `internal/check/check.go`.
  - `Compare(ref, target dotenv.File) Report` builds the R and T key sets with `dotenv.File.Keys()`.
  - It emits Missing for every key in R − T, located at `ref.FirstLine(key)`.
  - It sets `ReferenceKeys = len(R)`, then sorts.
  - Make T009 pass. Leave a clearly marked spot (`// pairing (US2), extra categories (US3)`)
    where later stories add findings.
- [X] T014 [US1] Implement `internal/report/text.go`.
  - `WriteText` follows the text format in contracts/cli.md:
    - Each finding is one line: `<file>:<lines joined by ",">: <label>: <key>`.
    - For Missing, the label is `missing`.
    - After the findings: a blank line, then `<N> issue(s): <count> <label>, ...`. The summary
      lists only non-zero categories, in Category order.
    - With no findings, print the OK line.
  - Use a single `bufio.Writer` and return the first write error.
  - Make T010 pass.
- [X] T015 [US1] Implement `cmd/env-checker/run.go`.
  - **Signature**: `run(args []string, stdout, stderr io.Writer) int`.
  - **Flags**: use a dedicated `flag.FlagSet("env-checker", flag.ContinueOnError)` with output
    sent to stderr. Flags are `--version` (bool) and `-h/--help`. Help prints usage to stdout
    and returns 0.
  - **Usage errors**: an unknown flag or any positional argument writes
    `env-checker: <msg>` and the usage text to stderr, and returns 2.
  - **Version**: a package variable `var version = "dev"` that `-ldflags "-X main.version=..."`
    can override.
  - **Loading files**: open `.env.example` and then `.env`, and parse each with `dotenv.Parse`.
    If either fails, write `env-checker: <wrapped error>` to stderr and return 2.
  - **Report**: run `check.Compare`, then `report.WriteText(stdout, ...)`. If the write fails,
    return 2. Otherwise return 0 if `OK()` is true and 1 if not.
- [X] T016 [US1] Implement `cmd/env-checker/main.go`: `func main() { os.Exit(run(os.Args[1:],
  os.Stdout, os.Stderr)) }`. Then run `go test ./cmd/env-checker -update` once, review the
  generated `text.golden` files against contracts/cli.md, and commit them. Make T012 pass.

**Checkpoint**: `go build ./cmd/env-checker` works, and quickstart Scenarios 1 and 2 behave
as documented. This is the MVP.

---

## Phase 4: User Story 2 - Catch likely misspellings (Priority: P2)

**Goal**: An unexpected key in `.env` that resembles a missing reference key is reported once,
as a likely misspelling with the suggested key. It is not reported as missing.

**Independent Test**: In `testdata/misspell/`, the reference file has `DATABASE_URL`,
`API_KEY` and `PORT`, and the target file has `DATABSE_URL`, `api_key` and `REDIS_HOST`.
- The output shows two likely misspellings: `DATABSE_URL` → `DATABASE_URL` and `api_key` →
  `API_KEY`.
- It shows `PORT` as missing.
- `REDIS_HOST` produces no misspelling finding. It is reported as unexpected only once US3 is
  done.
- The exit code is 1.

### Tests (write first, confirm they fail)

- [X] T017 [P] [US2] Write tests in `internal/check/similar_test.go` for
  `distance(a, b string) int`, which computes optimal string alignment distance on the
  strings as given:
  - `("", "")=0`, `("ABC", "ABC")=0`
  - `("ABCD", "ABDC")=1` (adjacent swap)
  - `("ABCD", "ABD")=1`, `("ABCD", "ABXD")=1`
  - `("CA", "ABC")=3` (the defining optimal-string-alignment case, where Damerau gives 2)
  - `("KITTEN", "SITTING")=3`
- [X] T018 [P] [US2] Write tests in `internal/check/similar_test.go` for
  `similar(missing, unexpected string) (dist int, ok bool)`, the FR-006 and research R5
  threshold. The upper-cased strings are compared. If they are equal, the result is
  `(0, true)`.
  - Otherwise let L be the length of the shorter string. If L is under 4, the result is false.
    If L is 4 to 7, the distance must be at most 1. If L is 8 or more, it must be at most 2.
  - **Typo corpus (SC-002)**: at least 20 pairs, of which at least 90% (≥18 of 20) MUST give
    `ok == true`. Include at least: `DATABASE_URL/DATABSE_URL`, `API_KEY/api_key`,
    `API_KEY/APIKEY`, `SECRET_KEY/SECERT_KEY`, `REDIS_HOST/REDIS_HSOT`, `PORT/PROT`,
    `AWS_REGION/AWS_REGOIN`, `SMTP_PASSWORD/SMTP_PASSWROD`, `LOG_LEVEL/LOG_LEVLE`,
    `JWT_SECRET/JWT_SECRT`, `DB_PASSWORD/DB_PASSWORDD`, `STRIPE_KEY/STRIPE_KYE`,
    `NODE_ENV/NODE_EVN`, `SENTRY_DSN/SENTRY_DNS`, `MAIL_FROM/MAIL_FORM`,
    `BUCKET_NAME/BUCKET_NMAE`, `TIMEOUT_MS/TIMEOUT_SM`, `CACHE_TTL/CAHCE_TTL`,
    `APP_URL/APP_UR`, `HOST/host`.
  - **Non-typo corpus (SC-003)**: at least 20 pairs, of which fewer than 5% (0 of 20) give
    `ok == true`. Include: `PORT/HOST`, `DB_HOST/DB_PORT`, `API_KEY/API_URL`,
    `DB/DC`, `ID/IP`, `REDIS_HOST/PORT`, `AWS_KEY/GCP_KEY`, `LOG_LEVEL/LOG_FILE`,
    `SMTP_HOST/SMTP_USER`, `CACHE_TTL/CACHE_DIR`, `APP_NAME/APP_ENV`, `MAIL_FROM/MAIL_TO`,
    `S3_BUCKET/GCS_BUCKET`, `JWT_SECRET/JWT_EXPIRY`, `NODE_ENV/NODE_PATH`,
    `DEBUG/LEVEL`, `USER/PASS`, `TZ/LANG`, `DB_NAME/DB_USER`, `HTTP_PORT/GRPC_PORT`.
  - If a listed pair contradicts the stated threshold, the threshold in FR-006 is
    authoritative. Replace the pair with a correct example and note it in a comment. Do not
    change the threshold.
- [X] T019 [P] [US2] Write tests in `internal/check/similar_test.go` for
  `pair(missing, unexpected []keyLine) []match`. Here `keyLine{Key string; Line int}` and
  `match{Missing, Unexpected keyLine}`. Cases:
  - **Closest match wins**: unexpected `DB_HOTS`, with missing `DB_HOST` (line 1) and
    `DB_HOSTS` (line 2). It pairs with the one at the smaller distance. If the distances are
    equal, it pairs with the earlier reference line (`DB_HOST`).
  - **Two unexpected keys for one missing key**: `DATABSE_URL` (line 3) and `DATABAS_URL`
    (line 5) for `DATABASE_URL`. Only the closer one is paired. If the distances are equal,
    the earlier target line (line 3) wins.
  - **No match**: dissimilar keys give no matches.
  - **Each key used once**: no key appears in more than one match.
  - **Deterministic**: shuffling the input order 50 times always gives the same result.
- [X] T020 [P] [US2] Add `Compare` tests to `internal/check/check_test.go`:
  - The `misspell` scenario gives two Misspelling findings and one Missing finding (`PORT`).
  - Misspelling `Locations == [{target, line}, {ref, line}]`, with `Key` set to the target key
    and `Suggestion` set to the reference key.
  - A key that has been paired does not also appear as Missing.
- [X] T021 [P] [US2] Add `WriteText` tests to `internal/report/report_test.go` for the
  misspelling line format: `.env:4: likely misspelling: DATABSE_URL → did you mean
  DATABASE_URL? (.env.example:2)`. Also check the summary label: `likely misspelling` for 1
  and `likely misspellings` for more than 1.
- [X] T022 [P] [US2] Create `testdata/misspell/.env.example` containing `DATABASE_URL=`,
  `API_KEY=` and `PORT=`. Create `testdata/misspell/.env` containing
  `DATABSE_URL=SENTINEL_d0_not_leak`, `api_key=x` and `REDIS_HOST=r`. Add a `misspell` row
  (code 1) to the table test in `cmd/env-checker/run_test.go`.

### Implementation

- [X] T023 [US2] Implement `internal/check/similar.go` with `distance`, `similar` and `pair`,
  following research R5.
  - `distance` uses an optimal-string-alignment table of two or three rows.
  - `similar` rejects a pair early if the lengths differ by more than the threshold, before
    calling `distance`.
  - `pair` builds every candidate pair, sorts them with `slices.SortFunc` by (distance,
    missing line, unexpected line), and accepts pairs greedily, skipping any key already used.
  - Make T017, T018 and T019 pass.
- [X] T024 [US2] Update `internal/check/check.go`:
  - Compute `unexpected₀` = T − R, ordered by `target.FirstLine`, and call `pair(missing₀,
    unexpected₀)`.
  - Emit a Misspelling finding for each match. Remove matched keys from the Missing output.
  - Keep the unmatched unexpected keys in a local variable for US3.
  - Make T020 pass.
- [X] T025 [US2] Update `internal/report/text.go` to render Misspelling findings and use the
  plural label rule. Then run `go test ./cmd/env-checker -update`, review
  `testdata/misspell/text.golden`, and confirm that T021 and T022 pass.

**Checkpoint**: Quickstart Scenario 3 behaves as documented, apart from the `REDIS_HOST`
unexpected line, which comes in US3. US1 golden files are unchanged.

---

## Phase 5: User Story 3 - Spot unexpected, duplicate, and empty keys (Priority: P3)

**Goal**: A full hygiene report covering unexpected keys, duplicates in either file, empty
values in `.env`, and unparseable lines in either file.

**Independent Test**: In `testdata/hygiene/`, the reference file has `PORT=` and `TOKEN=`. The
target file is `PORT=1`, `TOKEN=`, `this is not valid`, `PORT=2`, `LEGACY_FLAG=1`. The output
is, in order:

```text
.env:3: unparseable: missing "="
.env:1,4: duplicate: PORT
.env:2: empty: TOKEN
.env:5: unexpected: LEGACY_FLAG
```

It is followed by the summary, and the exit code is 1.

### Tests (write first, confirm they fail)

- [X] T026 [P] [US3] Add `Compare` tests to `internal/check/check_test.go`:
  - **Unexpected**: an unmatched key in T − R gives an Unexpected finding at
    `[{target, FirstLine}]`. In the `misspell` scenario, `REDIS_HOST` now becomes Unexpected.
  - **Duplicate in the target file**: `Locations` lists every line in ascending order.
  - **Duplicate in the reference file**: reported with the reference path. The key counts once
    for missing and unexpected detection.
  - **Empty**: an empty value in the target file gives an Empty finding for each empty entry's
    line. A key that is both a duplicate and empty on only one line gives one Duplicate finding
    plus one Empty finding listing only the empty line. An empty value in the reference file
    gives no finding.
  - **Empty overlaps**: an empty value on a key that is paired as a misspelling, or that is
    unexpected, still gives an Empty finding.
  - **Unparseable**: each `dotenv.ParseError` in either file gives one Unparseable finding,
    with `Reason` set and `Key` empty.
  - **Empty reference file**: if the reference file is empty and the target file has keys,
    every target key is Unexpected.
  - **Full ordering**: the `hygiene` scenario produces exactly the 4 findings above, in that
    order.
- [X] T027 [P] [US3] Add `WriteText` tests to `internal/report/report_test.go`:
  - `duplicate` lines join the line numbers with commas: `.env:1,4: duplicate: PORT`.
  - `empty: KEY`, `unexpected: KEY`, and `unparseable: <reason>` (the reason replaces the key).
  - A summary covering every category, in Category order, for example
    `7 issues: 1 unparseable, 1 missing, 2 likely misspellings, 1 duplicate, 1 empty,
    1 unexpected`.
- [X] T028 [P] [US3] Create fixtures, each with an `.env.example` and an `.env`, and each
  `.env` containing `SENTINEL_d0_not_leak`:
  - `testdata/hygiene/`: as in the Independent Test above. Put the sentinel in `PORT=2`'s
    place, as `PORT=SENTINEL_d0_not_leak`.
  - `testdata/crlf_bom/`: both files use CRLF line endings, and the `.env` starts with a
    byte-order mark. The key sets are identical, so the expected result is code 0.
  - `testdata/empty_files/`: both files are 0 bytes. Code 0. The OK line reports 0 keys.
  - `testdata/example_empty/`: an empty reference file and a target file with 2 keys. Both are
    reported as unexpected, code 1.
  - `testdata/syntax/`: the target file uses `export`, single and double quotes, a multi-line
    double-quoted value, `#` inside a value, and an inline comment. The key sets match the
    reference file. Code 0, with no unparseable findings.
  - `testdata/unterminated/`: the target file has `SECRET="SENTINEL_d0_not_leak` with no
    closing quote, followed by valid lines. Expect `unparseable: unterminated quote` at line 1,
    and the following keys are still checked.

  Write the files with `printf` or Go so the CRLF and byte-order-mark bytes are exact. Add a
  `testdata/.gitattributes` containing `* -text` so git does not normalise line endings.
- [X] T029 [US3] Add rows to the table test in `cmd/env-checker/run_test.go` for `hygiene`
  (1), `crlf_bom` (0), `empty_files` (0), `example_empty` (1), `syntax` (0) and
  `unterminated` (1).

### Implementation

- [X] T030 [US3] Update `internal/check/check.go` to emit, as specified in data-model.md
  "Comparison rules":
  - Unexpected for every unmatched key in T − R.
  - Duplicate for each file. Use the `File.Lines` method for keys with more than one line.
  - Empty for target entries with `Empty == true`. Group the lines per key into one finding,
    and don't add empty values from the reference file.
  - Unparseable for each ParseError in the reference file and then the target file.

  Make T026 pass.
- [X] T031 [US3] Update `internal/report/text.go` to render the Duplicate, Empty, Unexpected
  and Unparseable lines. Then run `go test ./cmd/env-checker -update`, review every new
  `text.golden`, and confirm that the `misspell` golden now includes
  `unexpected: REDIS_HOST`. Make T027 and T029 pass. `TestNoSecretLeak` must also pass,
  including the `unterminated` fixture.

**Checkpoint**: Quickstart Scenarios 1–4 behave as documented.

---

## Phase 6: User Story 4 - Use the checker in automation (Priority: P3)

**Goal**: Custom file paths, JSON output and guaranteed identical output for automation.

**Independent Test**: Running
`env-checker --example config/app.env.example --env config/app.env --format json` against the
hygiene inputs:
- prints a single line of valid JSON whose `summary` matches the hygiene counts
- exits with code 1
- produces byte-identical output when run twice

### Tests (write first, confirm they fail)

- [X] T032 [P] [US4] Write tests in `internal/report/report_test.go` for
  `WriteJSON(w io.Writer, r check.Report) error`, following contracts/json-report.md:
  - **Shape**: a single line ending in `\n`.
  - **Top-level fields**: `schema_version: 1`, `ok`, `reference` and `target`.
  - **Findings**: always an array, and `[]` (not `null`) when there are no findings. Each item
    has `category`, `certainty`, and `locations` as `[{file, line}]`.
  - **Optional fields**: `key` is omitted for `unparseable`. `suggestion` is present only for
    `likely_misspelling`, and `reason` only for `unparseable`.
  - **Summary**: `summary` has `total` and all 6 category counts, including zeros.
  - **Method**: unmarshal the output into a `map[string]any` and assert on that, and also
    compare one full report byte for byte with a literal string.
- [X] T033 [P] [US4] Add CLI tests to `cmd/env-checker/run_test.go`:
  - **Custom paths**: create `t.TempDir()/config/app.env.example` and
    `t.TempDir()/config/app.env` by copying `testdata/hygiene`. Call `run` with
    `--example <path> --env <path>`, without chdir. The output paths match the arguments
    exactly. Code 1.
  - **Single-dash flags**: `-env` works the same as `--env`.
  - **JSON golden files**: add `json.golden` to every scenario row in the table test, using
    `--format json`.
  - **Bad format**: `--format yaml` gives code 2 and stderr
    `env-checker: invalid --format "yaml": must be text or json`.
  - **Determinism (SC-006)**: for every scenario and both formats, running twice gives
    byte-identical stdout.
  - **Errors in JSON mode**: `--format json` against `no_env` gives code 2 with stdout empty.

### Implementation

- [X] T034 [US4] Implement `internal/report/json.go`.
  - Use private structs with `json` tags and `omitempty` on `key`, `suggestion` and `reason`.
  - Initialise `findings` with `make([]jsonFinding, 0, n)` so it is never `null`.
  - Build `summary` as a struct with fixed field order: `total`, `unparseable`, `missing`,
    `likely_misspelling`, `duplicate`, `empty`, `unexpected`.
  - Encode with `json.NewEncoder(w)` and `SetEscapeHTML(false)`.
  - Make T032 pass.
- [X] T035 [US4] Update `cmd/env-checker/run.go`:
  - **Flags**: `--example` (default `.env.example`), `--env` (default `.env`) and `--format`
    (default `text`). Check `--format` before reading any file.
  - **Output**: send the report to `WriteText` or `WriteJSON` based on `--format`.
  - **Messages**: use the paths exactly as given in findings and error messages.
- [X] T036 [US4] Extend `TestNoSecretLeak` in `cmd/env-checker/run_test.go` to cover
  `--format json`, error paths and `--help`. Run `go test ./cmd/env-checker -update`, review
  each `json.golden` against contracts/json-report.md, and confirm that T033 passes.

**Checkpoint**: Quickstart Scenario 5 and the secret-leak spot check behave as documented.
All user stories work.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T037 [P] Add `internal/check/bench_test.go`:
  - `BenchmarkCompare1000` generates two `dotenv.File`s with 1,000 keys each. 500 are shared,
    250 are only in the reference file, and 250 are only in the target file. Some of the
    target-only keys are deliberate typos.
  - `TestCompare1000UnderOneSecond` runs the same input and fails if it takes 1 second or more
    (SC-005). Skip it with `testing.Short()`.
- [X] T038 [P] Write `README.md` covering:
  - what the tool does
  - installing with `go install github.com/kyher/env-checker/cmd/env-checker@latest`
  - usage and flags, copied from contracts/cli.md
  - the finding categories and the misspelling rule (FR-006 in plain words)
  - the exit-code table
  - a JSON example
  - a CI snippet: a GitHub Actions step running `env-checker --format json`
  - the statement "values are never printed"
- [X] T039 [P] Add a `Makefile` with these targets:
  - `build`: `CGO_ENABLED=0 go build -ldflags "-X main.version=$(VERSION)" -o
    bin/env-checker ./cmd/env-checker`
  - `test`: `go test -race ./...`
  - `lint`: `gofmt -l . && go vet ./... && go tool staticcheck ./...`
  - `golden`: `go test ./cmd/env-checker -update`
- [X] T040 Run every quality gate locally: `gofmt -l .` (no output), `go vet ./...`,
  `go tool staticcheck ./...` and `go test -race ./...`. Fix everything they report.
- [X] T041 Run [quickstart.md](./quickstart.md) Scenarios 1–5 and the secret-leak spot check
  against `bin/env-checker`. Record any mismatch as a bug with a regression test first
  (Principle III), then fix it.

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (T001–T004)**: needs Go installed. T003 and T004 can run in parallel after T001.
- **Foundational (T005–T008)**: needs Setup. Blocks every user story.
- **US1 (T009–T016)**: needs Foundational. This is the MVP.
- **US2 (T017–T025)**: needs US1, because it extends `Compare`, `WriteText` and the
  golden-file test helpers.
- **US3 (T026–T031)**: needs US1. It can be developed alongside US2, but T030 must merge with
  T024 in `check.go`, so sequential order (US2 → US3) is recommended.
- **US4 (T032–T036)**: needs US1. T032 and T034 (JSON) only need the Foundational types and
  can start right after Phase 2. T033 and T036 golden files are best generated after US3 so
  every scenario exists.
- **Polish (T037–T041)**: needs all stories.

### Within each story

Tests come before implementation, and must be confirmed failing. Order within a story:
`internal/check` → `internal/report` → `cmd/env-checker`, then regenerate and review the
golden files.

### Task-level dependencies of note

- T007 → T005; T008 → T006; T013 → T009; T014 → T010; T015 → T013 and T014; T016 → T011,
  T012 and T015.
- T023 → T017–T019; T024 → T020 and T023; T025 → T021, T022 and T024.
- T030 → T026 and T024; T031 → T027–T030.
- T034 → T032; T035 → T034; T036 → T033 and T035.

---

## Parallel Examples

### Phase 2

```text
Task: "T005 parser tests in internal/dotenv/parse_test.go"
Task: "T006 report type tests in internal/check/report_test.go"
# then
Task: "T007 parser in internal/dotenv/parse.go"
Task: "T008 report types in internal/check/report.go"
```

### User Story 1

```text
Task: "T009 Compare missing tests in internal/check/check_test.go"
Task: "T010 WriteText tests in internal/report/report_test.go"
Task: "T011 fixtures in testdata/{clean,missing,no_env}/"
```

### User Story 2

```text
Task: "T017–T019 similar/pair tests in internal/check/similar_test.go"  # one file, one worker
Task: "T021 misspelling text tests in internal/report/report_test.go"
Task: "T022 testdata/misspell fixture + run_test row"
```

### User Story 4 (JSON can start early)

```text
Task: "T032 WriteJSON tests in internal/report/report_test.go"   # after Phase 2
Task: "T034 WriteJSON in internal/report/json.go"
```

---

## Implementation Strategy

### MVP first (User Story 1 only)

1. Phase 1 Setup, after installing Go.
2. Phase 2 Foundational.
3. Phase 3 US1. Then stop and check quickstart Scenarios 1 and 2. The tool is now useful for
   catching missing keys.

### Incremental delivery

1. **US2**: misspellings. This is the feature's distinguishing value.
2. **US3**: the hygiene categories. This completes the spec's list of finding categories.
3. **US4**: JSON output and custom paths, for CI adoption.
4. **Polish**: the benchmark, README, Makefile and full quickstart run.

Each checkpoint leaves `go test -race ./...` passing and the tool releasable.

---

## Notes

- Commit after each task or each test-and-implementation pair.
- Never update golden files without reading the diff. Golden diffs are review material.
- If a test reveals that the spec, contracts and data model disagree, stop and fix the
  documents first (`/speckit-analyze` can help) rather than letting the code decide.
