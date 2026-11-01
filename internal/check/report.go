// Package check compares a reference dotenv file against a target file and
// produces an ordered report of findings. It performs no I/O.
package check

import (
	"cmp"
	"slices"
)

// Category classifies a finding. Declaration order is report order (FR-014).
type Category int

const (
	Unparseable Category = iota
	Missing
	Misspelling
	Duplicate
	Empty
	Unexpected
)

var categoryNames = [...]string{
	Unparseable: "unparseable",
	Missing:     "missing",
	Misspelling: "likely_misspelling",
	Duplicate:   "duplicate",
	Empty:       "empty",
	Unexpected:  "unexpected",
}

// Categories returns every category in report order.
func Categories() []Category {
	return []Category{Unparseable, Missing, Misspelling, Duplicate, Empty, Unexpected}
}

// String returns the stable machine name used in JSON output.
func (c Category) String() string { return categoryNames[c] }

// Certainty is "suggestion" for heuristic findings and "definite" otherwise.
func (c Category) Certainty() string {
	if c == Misspelling {
		return "suggestion"
	}
	return "definite"
}

// Location is a line in one of the two files.
type Location struct {
	File string
	Line int
}

// Finding is one reported problem. Locations[0] is the primary location.
type Finding struct {
	Category   Category
	Key        string // empty for Unparseable
	Suggestion string // Misspelling only: the intended reference key
	Reason     string // Unparseable only
	Locations  []Location
}

// Report is the result of a comparison.
type Report struct {
	Reference     string
	Target        string
	ReferenceKeys int // number of unique keys in the reference file
	Findings      []Finding
}

// OK reports whether there are no findings.
func (r Report) OK() bool { return len(r.Findings) == 0 }

// Counts returns the number of findings per category, including zeros.
func (r Report) Counts() map[Category]int {
	c := make(map[Category]int, len(categoryNames))
	for _, cat := range Categories() {
		c[cat] = 0
	}
	for _, f := range r.Findings {
		c[f.Category]++
	}
	return c
}

// sort orders findings by category, primary file (reference first), primary
// line, then key.
func (r *Report) sort() {
	fileRank := func(f Finding) int {
		if f.Locations[0].File == r.Reference {
			return 0
		}
		return 1
	}
	slices.SortStableFunc(r.Findings, func(a, b Finding) int {
		return cmp.Or(
			cmp.Compare(a.Category, b.Category),
			cmp.Compare(fileRank(a), fileRank(b)),
			cmp.Compare(a.Locations[0].Line, b.Locations[0].Line),
			cmp.Compare(a.Key, b.Key),
		)
	})
}
