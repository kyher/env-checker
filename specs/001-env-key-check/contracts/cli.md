# Contract: Command-Line Interface

**Binary**: `env-checker`

## Synopsis

```text
env-checker [--example PATH] [--env PATH] [--format text|json]
env-checker --version
env-checker --help
```

| Flag | Default | Meaning |
|---|---|---|
| `--example PATH` | `.env.example` | Reference file: the keys that are expected |
| `--env PATH` | `.env` | Target file: the keys that are actually set |
| `--format FORMAT` | `text` | `text` for people, `json` for tools ([json-report.md](./json-report.md)) |
| `--version` | | Prints `env-checker <version>` to stdout and exits 0 |
| `--help`, `-h` | | Prints usage to stdout and exits 0 |

- Relative paths are resolved from the current directory.
- Paths in the output appear exactly as the user typed them.
- Single-dash forms such as `-env` are also accepted.
- Positional arguments are a usage error.
- The tool never prompts for input.

## Streams

- **stdout**: only the report (or the version or help text).
- **stderr**: only diagnostics.
- On exit code 2, stdout is empty.
- Neither stream ever contains a value from either file (Principle I).

## Exit codes

Exit codes are a stable public contract.

| Code | Meaning |
|---|---|
| 0 | Both files were read and there are no findings |
| 1 | Both files were read and there is at least one finding |
| 2 | Usage error (unknown flag, bad `--format`, positional argument), a file missing or unreadable, or an output write failure |

The error message on stderr for code 2 has this form:

```text
env-checker: <message>
```

Examples:

- `env-checker: open .env: no such file or directory`
- `env-checker: invalid --format "yaml": must be text or json`

Usage errors are followed by the usage text.

## Text format

Findings are grouped by category in the order of FR-014. Empty groups are not printed. Each
finding is on one line, with the main location first. The format is
`<file>:<line>: <message>`.

```text
.env:14: unparseable: missing "="
.env.example:3: missing: DB_PORT
.env:4: likely misspelling: DATABSE_URL → did you mean DATABASE_URL? (.env.example:2)
.env:7: likely misspelling: api_key → did you mean API_KEY? (.env.example:5)
.env:2,9: duplicate: PORT
.env:11: empty: SECRET_TOKEN
.env:12: unexpected: LEGACY_FLAG

7 issues: 1 unparseable, 1 missing, 2 likely misspellings, 1 duplicate, 1 empty, 1 unexpected
```

Rules:

- Within a group, findings are listed by file (reference first) and then by line.
- **Duplicate**: all line numbers are joined with commas.
- **Summary line**: preceded by a blank line. It lists only the non-zero categories, in the
  same order as the groups. Plurals follow normal English rules ("1 issue" and "3 issues").
  Category names don't change in the plural, except "likely misspelling(s)".
- **No findings**: the whole output is a single line:

```text
OK: all 12 keys in .env.example are present in .env
```

## Compatibility

- Adding a flag or a new finding category is a MINOR version change.
- Changing an existing exit code, flag name or text output line format is a MAJOR version
  change.
