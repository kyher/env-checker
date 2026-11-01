# Contract: JSON Report (`--format json`)

The report is a single JSON object written on one line and followed by a newline. It uses
the same exit codes as text mode. On exit code 2, nothing is written to stdout.

## Example

```json
{
  "schema_version": 1,
  "ok": false,
  "reference": ".env.example",
  "target": ".env",
  "findings": [
    {
      "category": "unparseable",
      "certainty": "definite",
      "reason": "missing \"=\"",
      "locations": [{"file": ".env", "line": 14}]
    },
    {
      "category": "missing",
      "certainty": "definite",
      "key": "DB_PORT",
      "locations": [{"file": ".env.example", "line": 3}]
    },
    {
      "category": "likely_misspelling",
      "certainty": "suggestion",
      "key": "DATABSE_URL",
      "suggestion": "DATABASE_URL",
      "locations": [{"file": ".env", "line": 4}, {"file": ".env.example", "line": 2}]
    },
    {
      "category": "duplicate",
      "certainty": "definite",
      "key": "PORT",
      "locations": [{"file": ".env", "line": 2}, {"file": ".env", "line": 9}]
    }
  ],
  "summary": {
    "total": 4,
    "unparseable": 1,
    "missing": 1,
    "likely_misspelling": 1,
    "duplicate": 1,
    "empty": 0,
    "unexpected": 0
  }
}
```

The example is pretty-printed for readability. The real output is on one line.

## Fields

| Field | Type | Always present | Notes |
|---|---|---|---|
| `schema_version` | integer | yes | Currently `1` |
| `ok` | boolean | yes | `true` if and only if `findings` is empty |
| `reference`, `target` | string | yes | Paths exactly as given |
| `findings` | array | yes | `[]` when there are no findings, never `null`. Ordered as in [data-model.md](../data-model.md#ordering-fr-014-principle-iv) |
| `findings[].category` | string | yes | `unparseable` \| `missing` \| `likely_misspelling` \| `duplicate` \| `empty` \| `unexpected` |
| `findings[].certainty` | string | yes | `suggestion` for `likely_misspelling`, otherwise `definite` |
| `findings[].key` | string | not for `unparseable` | |
| `findings[].suggestion` | string | only `likely_misspelling` | The intended reference key |
| `findings[].reason` | string | only `unparseable` | Taken from a fixed set (see research R3) |
| `findings[].locations` | array | yes | At least 1 item, each `{file, line}`. The first item is the main location. |
| `summary` | object | yes | `total` plus a count for every category. Every category is always listed, including those at 0. |

## Guarantees

- No field ever contains a value from either file.
- The output is identical for identical inputs.

## Versioning

- Adding an optional field or a new category is backwards-compatible, and `schema_version`
  stays the same.
- Removing or renaming a field, or changing what it means, increases `schema_version`.
