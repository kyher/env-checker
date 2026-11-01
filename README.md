# env-checker

Compare a `.env` file against `.env.example` and find missing, misspelled, unexpected,
duplicate, and empty keys.

```console
$ env-checker
.env.example:3: missing: DB_PORT
.env:4: likely misspelling: DATABSE_URL → did you mean DATABASE_URL? (.env.example:2)
.env:2,9: duplicate: PORT
.env:11: empty: SECRET_TOKEN
.env:12: unexpected: LEGACY_FLAG

5 issues: 1 missing, 1 likely misspelling, 1 duplicate, 1 empty, 1 unexpected
```

**Values are never printed.** env-checker only looks at key names, and whether a value is
empty. It never stores, logs, or outputs any value from either file, so it's safe to run in CI
logs.

## Install

```sh
go install github.com/kyher/env-checker/cmd/env-checker@latest
```

Or build from source: `make build` (produces `bin/env-checker`).

## Usage

```text
env-checker [--example PATH] [--env PATH] [--format text|json]
env-checker --version
```

| Flag | Default | Meaning |
|---|---|---|
| `--example PATH` | `.env.example` | Reference file: the keys that are expected |
| `--env PATH` | `.env` | Target file: the keys that are actually set |
| `--format FORMAT` | `text` | `text` for people, `json` for tools |

### Exit codes

| Code | Meaning |
|---|---|
| 0 | No issues |
| 1 | Issues found |
| 2 | Usage error, or a file couldn't be read |

## What it reports

| Category | Meaning |
|---|---|
| `unparseable` | A line in either file that isn't a valid `KEY=VALUE` entry. The rest of the file is still checked. |
| `missing` | A key declared in `.env.example` but not set in `.env` |
| `likely_misspelling` | A key in `.env` that looks like a typo of a missing key. Shown with the suggested key. This is a *suggestion*, not a certainty. |
| `duplicate` | A key set more than once in the same file |
| `empty` | A key in `.env` with an empty value (`KEY=`, `KEY=""`) |
| `unexpected` | A key in `.env` that `.env.example` doesn't declare |

### How misspellings are detected

A key in `.env` is treated as a likely misspelling of a missing key when:

- the two keys are identical apart from upper and lower case (`api_key` vs `API_KEY`), or
- both keys are at least 4 characters long and differ by at most one typo. A typo is one
  added, removed, or changed character, or two neighbouring characters swapped. Keys of 8 or
  more characters may differ by up to two typos.

Each key is paired at most once, closest match first.

### Supported syntax

The checker handles:

- comments, both on their own line and after a value (`KEY=value # comment`)
- blank lines
- an `export` prefix
- single- and double-quoted values, including double-quoted values over several lines
- Windows (CRLF) line endings
- a UTF-8 byte-order mark at the start of the file

`${VAR}` references are not expanded.

## JSON output

`--format json` prints one JSON object on a single line:

```json
{"schema_version":1,"ok":false,"reference":".env.example","target":".env",
 "findings":[{"category":"missing","certainty":"definite","key":"DB_PORT",
 "locations":[{"file":".env.example","line":3}]}],
 "summary":{"total":1,"unparseable":0,"missing":1,"likely_misspelling":0,
 "duplicate":0,"empty":0,"unexpected":0}}
```

See [the JSON contract](specs/001-env-key-check/contracts/json-report.md) for every field.

## In CI

```yaml
- name: Check env keys
  run: |
    go install github.com/kyher/env-checker/cmd/env-checker@latest
    env-checker --example .env.example --env .env.ci
```

## Development

```sh
make test     # go test -race ./...
make lint     # gofmt, go vet, staticcheck
make golden   # regenerate testdata/*/*.golden; review the diff before committing
```
