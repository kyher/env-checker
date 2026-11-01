package check

import (
	"cmp"
	"slices"
	"strings"
)

// keyLine is a key and the first line it appears on in its file.
type keyLine struct {
	Key  string
	Line int
}

// match pairs a missing reference key with the unexpected target key that is
// probably a misspelling of it.
type match struct {
	Missing    keyLine
	Unexpected keyLine
}

// distance returns the optimal string alignment distance between a and b:
// the number of single-byte insertions, deletions, substitutions, and
// adjacent transpositions needed, with no substring edited more than once.
// Keys are ASCII (enforced by the parser), so bytes are characters.
func distance(a, b string) int {
	if len(a) < len(b) {
		a, b = b, a
	}
	prev2 := make([]int, len(b)+1) // row i-2
	prev := make([]int, len(b)+1)  // row i-1
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				cur[j] = min(cur[j], prev2[j-2]+1)
			}
		}
		prev2, prev, cur = prev, cur, prev2
	}
	return prev[len(b)]
}

// similar reports whether unexpected is likely a misspelling of missing
// (FR-006): identical ignoring case, or — when both keys are at least 4
// characters — within 1 edit (shorter key under 8 characters) or 2 edits.
func similar(missing, unexpected string) (dist int, ok bool) {
	a, b := strings.ToUpper(missing), strings.ToUpper(unexpected)
	if a == b {
		return 0, true
	}
	shorter := min(len(a), len(b))
	if shorter < 4 {
		return 0, false
	}
	limit := 2
	if shorter < 8 {
		limit = 1
	}
	if gap := len(a) - len(b); gap > limit || -gap > limit {
		return 0, false
	}
	d := distance(a, b)
	return d, d <= limit
}

// pair greedily matches missing and unexpected keys, closest first. Ties are
// broken by reference line, then target line, so the result is deterministic
// and independent of input order. Each key is used at most once.
func pair(missing, unexpected []keyLine) []match {
	type candidate struct {
		match
		dist int
	}
	var cands []candidate
	for _, m := range missing {
		for _, u := range unexpected {
			if d, ok := similar(m.Key, u.Key); ok {
				cands = append(cands, candidate{match{m, u}, d})
			}
		}
	}
	slices.SortFunc(cands, func(x, y candidate) int {
		return cmp.Or(
			cmp.Compare(x.dist, y.dist),
			cmp.Compare(x.Missing.Line, y.Missing.Line),
			cmp.Compare(x.Unexpected.Line, y.Unexpected.Line),
		)
	})
	usedM, usedU := map[string]bool{}, map[string]bool{}
	var out []match
	for _, c := range cands {
		if usedM[c.Missing.Key] || usedU[c.Unexpected.Key] {
			continue
		}
		usedM[c.Missing.Key], usedU[c.Unexpected.Key] = true, true
		out = append(out, c.match)
	}
	return out
}
