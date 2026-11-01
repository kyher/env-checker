# Feature Specification: Env Key Check

**Feature Branch**: `001-env-key-check` (spec directory; work is currently on `main`)

**Created**: 2026-11-01

**Status**: Draft

**Input**: User description: "Compare .env.example against .env, flag missing and misspelled keys"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find missing keys (Priority: P1)

A developer has cloned a project, copied or hand-written a `.env` file, and wants to know
whether every variable the project expects (as declared in `.env.example`) is present. They
run the checker in the project directory and get a list of keys that are declared in the
example file but absent from their `.env`, each with the line in `.env.example` where it is
declared.

**Why this priority**: Missing variables are the most common cause of "works on my machine"
startup failures. Detecting them alone makes the tool useful.

**Independent Test**: Run the checker against a `.env.example` with keys A, B, C and a `.env`
with keys A, C; it reports B as missing and signals that issues were found.

**Acceptance Scenarios**:

1. **Given** `.env.example` declares `DB_HOST`, `DB_PORT`, `API_KEY` and `.env` defines all
   three, **When** the user runs the checker, **Then** it reports no issues and signals success.
2. **Given** `.env.example` declares `DB_HOST`, `DB_PORT`, `API_KEY` and `.env` defines only
   `DB_HOST` and `API_KEY`, **When** the user runs the checker, **Then** it reports `DB_PORT`
   as missing, with its line number in `.env.example`, and signals that issues were found.
3. **Given** `.env` does not exist, **When** the user runs the checker, **Then** it reports a
   clear error naming the missing file and signals an error distinct from "issues found".

---

### User Story 2 - Catch likely misspellings (Priority: P2)

A developer's `.env` contains `DATABSE_URL` while `.env.example` declares `DATABASE_URL`.
Rather than seeing two unrelated findings ("`DATABASE_URL` missing" and "`DATABSE_URL`
unexpected"), they see one finding: "`DATABSE_URL` (line 4 of `.env`) looks like a misspelling
of `DATABASE_URL`", clearly labelled as a suggestion rather than a certainty.

**Why this priority**: Typos are the second most common failure and are hard to spot by eye.
Pairing them with the intended key makes the fix obvious.

**Independent Test**: Run the checker with `.env.example` declaring `DATABASE_URL` and `.env`
defining `DATABSE_URL`; it reports exactly one likely-misspelling finding with the suggested
key and does not separately report the key as missing or unexpected.

**Acceptance Scenarios**:

1. **Given** `.env.example` declares `DATABASE_URL` and `.env` defines `DATABSE_URL`,
   **When** the user runs the checker, **Then** one finding reports `DATABSE_URL` as a likely
   misspelling of `DATABASE_URL`, with line numbers in both files.
2. **Given** `.env.example` declares `API_KEY` and `.env` defines `api_key`, **When** the user
   runs the checker, **Then** it reports `api_key` as a likely misspelling (case mismatch) of
   `API_KEY`.
3. **Given** `.env.example` declares `PORT` and `.env` defines `REDIS_HOST`, **When** the user
   runs the checker, **Then** `PORT` is reported as missing and `REDIS_HOST` as unexpected;
   no misspelling is suggested because the names are not similar.

---

### User Story 3 - Spot unexpected, duplicate, and empty keys (Priority: P3)

A developer wants a complete hygiene report: keys in `.env` that the example does not
declare (stale or unknown variables), keys defined more than once in `.env` (where only one
value takes effect), keys present but with an empty value, and lines the checker could not
understand.

**Why this priority**: These are real but less frequent problems; the tool is valuable
without them.

**Independent Test**: Run the checker with a `.env` that contains an undeclared key, a key
defined twice, a key with an empty value, and a malformed line; each is reported once under
its own category with its line number(s).

**Acceptance Scenarios**:

1. **Given** `.env` defines `LEGACY_FLAG` which is not declared in `.env.example` and is not
   similar to any missing key, **When** the user runs the checker, **Then** `LEGACY_FLAG` is
   reported as unexpected with its line number.
2. **Given** `.env` defines `PORT` on lines 2 and 9, **When** the user runs the checker,
   **Then** `PORT` is reported as a duplicate, listing both line numbers.
3. **Given** `.env` contains `API_KEY=` (no value), **When** the user runs the checker,
   **Then** `API_KEY` is reported as empty, without being reported as missing.
4. **Given** `.env` contains a line `this is not valid`, **When** the user runs the checker,
   **Then** that line is reported as unparseable with its line number, and the rest of the
   file is still checked.

---

### User Story 4 - Use the checker in automation (Priority: P3)

A team adds the checker to CI or a pre-commit hook. They need a pass/fail signal, the option
to point at files other than the defaults, and a machine-readable report they can feed into
other tools.

**Why this priority**: Extends reach to teams, but depends on the detection in Stories 1–3.

**Independent Test**: Run the checker with explicit file paths and the machine-readable
option against inputs with known issues; the outcome signal indicates issues and the report
parses and lists every finding.

**Acceptance Scenarios**:

1. **Given** files at `config/app.env.example` and `config/app.env`, **When** the user runs the
   checker with those paths, **Then** it compares those files instead of the defaults.
2. **Given** inputs with two findings, **When** the user requests machine-readable output,
   **Then** the output contains exactly those two findings with category, key, suggestion
   (where applicable), file, and line number(s), and nothing else on standard output.
3. **Given** the same inputs, **When** the checker is run twice, **Then** both outputs are
   identical.

### Edge Cases

- `.env.example` is missing or unreadable → error naming the file; distinct error outcome.
- Both files are empty → no findings; success.
- `.env.example` is empty but `.env` has keys → every `.env` key reported as unexpected.
- A key is duplicated in `.env.example` → reported as a duplicate in that file; the key is
  still treated as declared once for comparison.
- One unexpected key is similar to two missing keys (e.g. `DB_HOTS` vs `DB_HOST` and
  `DB_HOSTS`) → paired with the single closest match; ties broken by order of appearance in
  `.env.example`. The other missing key is reported as missing.
- Two unexpected keys are similar to one missing key → only the closest is paired as a
  misspelling; the other is reported as unexpected.
- Lines with comments, blank lines, an `export ` prefix, quoted values, values containing `=`
  or `#`, Windows line endings, or a leading byte-order mark → handled without false findings.
- A value that spans multiple lines inside quotes → treated as a single entry; its
  continuation lines are not reported as unparseable.
- Very short keys (e.g. `DB` vs `DC`) → not suggested as misspellings, to avoid noise.

## Requirements *(mandatory)*

### Functional Requirements

**Inputs**

- **FR-001**: The checker MUST compare a reference file (default `.env.example`) against a
  target file (default `.env`), both resolved relative to the current directory.
- **FR-002**: Users MUST be able to specify alternative paths for either file.
- **FR-003**: The checker MUST read entries of the form `KEY=VALUE`, ignoring blank lines and
  full-line comments, accepting an optional `export ` prefix, and accepting single- or
  double-quoted values (including quoted values spanning multiple lines).

**Findings**

- **FR-004**: The checker MUST report every key declared in the reference file but absent from
  the target file as **missing**, unless it is paired as a misspelling (FR-006).
- **FR-005**: The checker MUST report every key in the target file not declared in the
  reference file as **unexpected**, unless it is paired as a misspelling (FR-006).
- **FR-006**: The checker MUST pair an unexpected key with a missing key and report it as a
  **likely misspelling** when either:
  (a) the two keys are identical ignoring letter case; or
  (b) both keys are at least 4 characters long and one can be turned into the other with at
  most 2 single-character insertions, deletions, substitutions, or adjacent swaps (at most 1
  such edit when the shorter key is under 8 characters).
  Each key MUST be paired at most once, choosing the closest match first; ties are broken by
  order of appearance in the reference file, then in the target file.
- **FR-007**: Likely-misspelling findings MUST name the suggested intended key and MUST be
  labelled as suggestions, distinguishable from definite findings.
- **FR-008**: The checker MUST report a key defined more than once within the same file as a
  **duplicate**, listing all line numbers.
- **FR-009**: The checker MUST report a key present in the target file with an empty value as
  **empty**.
- **FR-010**: The checker MUST report any line it cannot interpret as **unparseable**, with its
  line number, and MUST continue checking the remainder of the file.
- **FR-011**: Every finding MUST include its category, the key (where applicable), the file,
  and the line number(s).

**Output and outcome**

- **FR-012**: The checker MUST NOT display, log, or include in any output any value from the
  target file, in any output mode or error message.
- **FR-013**: The checker MUST produce a human-readable report by default and a
  machine-readable report on request.
- **FR-014**: Findings MUST be listed in a stable order: grouped by category (unparseable,
  missing, likely misspelling, duplicate, empty, unexpected), then by line number.
- **FR-015**: The checker MUST end with a distinct outcome for each case: no findings
  (success), one or more findings (issues found), and usage or file-access problems (error).
- **FR-016**: When there are no findings, the human-readable report MUST say so explicitly.
- **FR-017**: The report MUST end with a summary count of findings per category.

### Key Entities

- **Reference file**: the declaration of expected keys (normally `.env.example`); its values
  are placeholders and are not compared.
- **Target file**: the developer's actual configuration (normally `.env`); its values are
  secret and are never shown.
- **Entry**: a key, its line number(s), and whether its value is empty.
- **Finding**: a category (missing, likely misspelling, unexpected, duplicate, empty,
  unparseable), the key, an optional suggested key, the file, and line number(s).
- **Report**: the ordered list of findings, per-category counts, and the overall outcome.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can identify every missing or misspelled variable in their `.env`
  with a single command, in under 10 seconds from starting to read the report.
- **SC-002**: On a reference set of at least 20 realistic typo cases (character swaps,
  dropped or doubled letters, case mismatches), at least 90% are reported as likely
  misspellings of the correct key.
- **SC-003**: On a reference set of at least 20 pairs of genuinely different keys, fewer than
  5% are wrongly suggested as misspellings.
- **SC-004**: No value from the target file appears in any output, verified across all output
  modes with planted sentinel values (0 occurrences).
- **SC-005**: Files with up to 1,000 entries each are checked in under 1 second on a typical
  developer laptop.
- **SC-006**: Running the checker twice on the same inputs produces identical output 100% of
  the time.

## Assumptions

- Values in the reference file are placeholders; only keys are compared. Value-format checks
  (e.g. "must be a number") are out of scope.
- Unexpected keys count as findings and cause an "issues found" outcome. Teams that keep
  local-only variables can declare them in the example file; an ignore mechanism is out of
  scope for this feature.
- Empty values in the reference file are normal and are not reported; only empty values in
  the target file are reported.
- Variable interpolation (`${OTHER_VAR}`) is not expanded; values are not inspected beyond
  emptiness.
- Comparing more than two files, multiple environments, or fixing files automatically is out
  of scope.
- Keys are case-sensitive for matching; a case-only difference is reported as a likely
  misspelling, not as a match.
