// Package report renders a check.Report for people (text) or tools (JSON).
package report

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/kyher/env-checker/internal/check"
)

// WriteText writes the human-readable report described in contracts/cli.md.
func WriteText(w io.Writer, r check.Report) error {
	bw := bufio.NewWriter(w)
	if r.OK() {
		fmt.Fprintf(bw, "OK: all %d keys in %s are present in %s\n", r.ReferenceKeys, r.Reference, r.Target)
		return bw.Flush()
	}
	for _, f := range r.Findings {
		fmt.Fprintf(bw, "%s: %s: %s\n", position(f.Locations), label(f.Category, 1), subject(f))
	}
	fmt.Fprintf(bw, "\n%s\n", summaryLine(r))
	return bw.Flush()
}

// position renders "<file>:<line>[,<line>...]" using the primary file.
func position(locs []check.Location) string {
	file := locs[0].File
	lines := make([]string, 0, len(locs))
	for _, l := range locs {
		if l.File == file {
			lines = append(lines, strconv.Itoa(l.Line))
		}
	}
	return file + ":" + strings.Join(lines, ",")
}

func subject(f check.Finding) string {
	switch f.Category {
	case check.Unparseable:
		return f.Reason
	case check.Misspelling:
		hint := f.Locations[len(f.Locations)-1]
		return fmt.Sprintf("%s → did you mean %s? (%s:%d)", f.Key, f.Suggestion, hint.File, hint.Line)
	}
	return f.Key
}

// label is the human name of a category; n selects singular or plural.
func label(c check.Category, n int) string {
	switch c {
	case check.Misspelling:
		if n == 1 {
			return "likely misspelling"
		}
		return "likely misspellings"
	}
	return c.String()
}

func summaryLine(r check.Report) string {
	counts := r.Counts()
	total := len(r.Findings)
	parts := make([]string, 0, len(counts))
	for _, c := range check.Categories() {
		if n := counts[c]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, label(c, n)))
		}
	}
	noun := "issues"
	if total == 1 {
		noun = "issue"
	}
	return fmt.Sprintf("%d %s: %s", total, noun, strings.Join(parts, ", "))
}
