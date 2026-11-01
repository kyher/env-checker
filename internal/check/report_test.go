package check

import (
	"slices"
	"testing"
)

func loc(file string, line int) []Location { return []Location{{File: file, Line: line}} }

func TestReportSortOrder(t *testing.T) {
	r := Report{Reference: ".env.example", Target: ".env"}
	want := []Finding{
		{Category: Unparseable, Reason: "x", Locations: loc(".env.example", 9)},
		{Category: Unparseable, Reason: "x", Locations: loc(".env", 1)},
		{Category: Missing, Key: "B", Locations: loc(".env.example", 2)},
		{Category: Missing, Key: "A", Locations: loc(".env.example", 5)},
		{Category: Misspelling, Key: "Y", Suggestion: "X", Locations: []Location{{".env", 3}, {".env.example", 1}}},
		{Category: Duplicate, Key: "D", Locations: []Location{{".env.example", 4}, {".env.example", 6}}},
		{Category: Duplicate, Key: "D", Locations: []Location{{".env", 1}, {".env", 7}}},
		{Category: Empty, Key: "E", Locations: loc(".env", 2)},
		{Category: Unexpected, Key: "M", Locations: loc(".env", 8)},
		{Category: Unexpected, Key: "N", Locations: loc(".env", 8)},
	}
	// Reverse to get a scrambled input.
	r.Findings = slices.Clone(want)
	slices.Reverse(r.Findings)
	r.sort()
	for i := range want {
		if r.Findings[i].Category != want[i].Category || r.Findings[i].Key != want[i].Key ||
			r.Findings[i].Locations[0] != want[i].Locations[0] {
			t.Fatalf("position %d: got %+v, want %+v", i, r.Findings[i], want[i])
		}
	}
}

func TestReportCountsAndOK(t *testing.T) {
	var r Report
	if !r.OK() {
		t.Error("empty report should be OK")
	}
	c := r.Counts()
	if len(c) != 6 {
		t.Errorf("Counts has %d categories, want 6", len(c))
	}
	r.Findings = []Finding{
		{Category: Missing, Key: "A", Locations: loc("r", 1)},
		{Category: Missing, Key: "B", Locations: loc("r", 2)},
		{Category: Empty, Key: "C", Locations: loc("t", 1)},
	}
	if r.OK() {
		t.Error("report with findings should not be OK")
	}
	c = r.Counts()
	if c[Missing] != 2 || c[Empty] != 1 || c[Unexpected] != 0 || len(c) != 6 {
		t.Errorf("Counts = %v", c)
	}
}

func TestCategoryStrings(t *testing.T) {
	want := []string{"unparseable", "missing", "likely_misspelling", "duplicate", "empty", "unexpected"}
	cats := Categories()
	if len(cats) != len(want) {
		t.Fatalf("Categories() len = %d", len(cats))
	}
	for i, c := range cats {
		if c.String() != want[i] {
			t.Errorf("%d: String() = %q, want %q", i, c.String(), want[i])
		}
		wantCert := "definite"
		if c == Misspelling {
			wantCert = "suggestion"
		}
		if c.Certainty() != wantCert {
			t.Errorf("%s: Certainty() = %q, want %q", c, c.Certainty(), wantCert)
		}
	}
}
