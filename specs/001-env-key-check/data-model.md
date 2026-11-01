# Data Model: Env Key Check

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

All types are in-memory. Nothing is persisted.

## dotenv.Entry

One `KEY=VALUE` assignment read from a file.

| Field | Type | Notes |
|---|---|---|
| Key | string | Matches `[A-Za-z_][A-Za-z0-9_]*` |
| Line | int | 1-based line where the entry starts |
| Empty | bool | True if the value is empty after quotes are removed (`KEY=`, `KEY=""`, `KEY=''`) |

**Never stores the value** (Principle I, research R4).

## dotenv.ParseError

| Field | Type | Notes |
|---|---|---|
| Line | int | 1-based |
| Reason | string | One of a fixed set: `missing "="`, `invalid key`, `unterminated quote`, `unexpected text after closing quote`. Never includes text from the line. |

## dotenv.File

The result of parsing one file.

| Field | Type | Notes |
|---|---|---|
| Path | string | Exactly as the user gave it; used in findings |
| Entries | []Entry | In file order. A duplicate key appears more than once. |
| Errors | []ParseError | In file order |

**Derived**: `FirstLine(key)` is the line of the first entry with that key. `Lines(key)` is
every line that has the key.

## check.Category

An enum. Its declaration order is the report order (FR-014):

| Order | Value | JSON | Certainty | Possible in |
|---|---|---|---|---|
| 1 | Unparseable | `unparseable` | definite | reference, target |
| 2 | Missing | `missing` | definite | reference |
| 3 | Misspelling | `likely_misspelling` | suggestion | target (also refers to a reference line) |
| 4 | Duplicate | `duplicate` | definite | reference, target |
| 5 | Empty | `empty` | definite | target |
| 6 | Unexpected | `unexpected` | definite | target |

## check.Location

| Field | Type | Notes |
|---|---|---|
| File | string | `File.Path` of the file that contains the line |
| Line | int | 1-based |

## check.Finding

| Field | Type | Notes |
|---|---|---|
| Category | Category | |
| Key | string | Empty for `unparseable` |
| Suggestion | string | Only set for `likely_misspelling`: the intended key from the reference file |
| Reason | string | Only set for `unparseable`: the ParseError reason |
| Locations | []Location | At least 1. The first location is the main one. |

What each category's Locations contain:

- **unparseable**: `[{file, line}]`
- **missing**: `[{reference, firstLine}]`
- **likely_misspelling**: `[{target, firstLine(key)}, {reference, firstLine(suggestion)}]`
- **duplicate**: every line of the key in that file, in ascending order
- **empty**: `[{target, line}]` for each empty entry of the key. If the key is duplicated,
  only the entries that are empty are listed.
- **unexpected**: `[{target, firstLine}]`

## check.Report

| Field | Type | Notes |
|---|---|---|
| Reference | string | Path |
| Target | string | Path |
| Findings | []Finding | Sorted. See ordering below. |

**Derived**: `Counts()` returns the number of findings per category. `OK()` returns
`len(Findings) == 0`.

### Ordering (FR-014, Principle IV)

Findings are sorted by these fields, in this order:

1. Category, in declaration order.
2. The main location's file: reference file first, then target file.
3. The main location's line.
4. Key.

### Comparison rules

Let **R** be the set of keys in the reference file and **T** the set of keys in the target
file. The `−` below means set difference.

1. `missing₀` = R − T and `unexpected₀` = T − R. Both are ordered by first line.
2. Misspelling pairing (research R5) runs over `missing₀ × unexpected₀`. A key that ends up
   in a pair is removed from both sets.
3. Each pair gives one `likely_misspelling` finding. Each remaining key gives a `missing` or
   `unexpected` finding.
4. **duplicate**: a key that appears on more than one line in the same file. This applies to
   both files independently. The key still counts once in R or T.
5. **empty**: a key in T with any entry where `Empty` is true. This includes keys that are
   paired as misspellings or reported as unexpected. Empty values in the reference file are
   never reported.
6. **unparseable**: one finding for each ParseError, in each file.

## Exit status

| Condition | Code |
|---|---|
| Both files read and `Report.OK()` | 0 |
| Both files read and there is at least one finding | 1 |
| Bad flags, a file missing or unreadable, or an output write failure | 2 |
