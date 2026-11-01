# Quickstart: Validate Env Key Check

This guide checks that the feature works from end to end. For exact output formats, see
[contracts/cli.md](./contracts/cli.md) and [contracts/json-report.md](./contracts/json-report.md).

## Prerequisites

- Go 1.25 or newer (`go version`). It is **not currently installed** on the dev machine.
- Commands are run from the repository root.

## Build and run the quality gates

```sh
go build -o bin/env-checker ./cmd/env-checker
gofmt -l .                    # expect no output
go vet ./...
go tool staticcheck ./...
go test -race ./...           # includes the golden-file and secret-leak tests
```

## Scenario 1: Clean (US1)

```sh
mkdir -p /tmp/ec && cd /tmp/ec
printf 'DB_HOST=localhost\nDB_PORT=5432\nAPI_KEY=\n' > .env.example
printf 'DB_HOST=db\nDB_PORT=5432\nAPI_KEY=SENTINEL_d0_not_leak\n' > .env
env-checker; echo "exit=$?"
```

Expected: `OK: all 3 keys in .env.example are present in .env` and `exit=0`.

## Scenario 2: Missing key (US1)

```sh
printf 'DB_HOST=db\nAPI_KEY=SENTINEL_d0_not_leak\n' > .env
env-checker; echo "exit=$?"
```

Expected: `.env.example:2: missing: DB_PORT`, a summary line `1 issue: 1 missing`, and
`exit=1`.

## Scenario 3: Misspellings (US2)

```sh
printf 'DATABASE_URL=\nAPI_KEY=\nPORT=\n' > .env.example
printf 'DATABSE_URL=SENTINEL_d0_not_leak\napi_key=x\nREDIS_HOST=r\n' > .env
env-checker; echo "exit=$?"
```

Expected:

- `DATABSE_URL` is reported as a likely misspelling of `DATABASE_URL`.
- `api_key` is reported as a likely misspelling of `API_KEY`.
- `PORT` is reported as missing.
- `REDIS_HOST` is reported as unexpected. It is not paired with `PORT` because the two names
  aren't similar.
- `exit=1`.

## Scenario 4: Hygiene (US3)

```sh
printf 'PORT=\nTOKEN=\n' > .env.example
printf 'PORT=1\nTOKEN=\nthis is not valid\nPORT=2\nLEGACY_FLAG=1\n' > .env
env-checker; echo "exit=$?"
```

Expected:

- `.env:3: unparseable: missing "="` is listed first.
- `.env:1,4: duplicate: PORT`
- `.env:2: empty: TOKEN`
- `.env:5: unexpected: LEGACY_FLAG`
- `exit=1`.

## Scenario 5: Automation (US4)

```sh
mkdir -p config && cp .env.example config/app.env.example && cp .env config/app.env
env-checker --example config/app.env.example --env config/app.env --format json | jq .summary
env-checker --format json > a.json; env-checker --format json > b.json; cmp a.json b.json && echo identical
env-checker --env nope.env; echo "exit=$?"
```

Expected:

- The first command prints the summary with counts that match Scenario 4.
- The second command prints `identical`.
- The third command prints `env-checker: open nope.env: no such file or directory` on stderr,
  nothing on stdout, and `exit=2`.

## Secret-leak spot check (Principle I)

```sh
for f in text json; do env-checker --format $f 2>&1; done | grep -c SENTINEL_d0_not_leak
```

Expected: `0`. Also run this after Scenarios 1–5. The automated version of this check is part
of `go test`.
