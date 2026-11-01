package check

import "github.com/kyher/env-checker/internal/dotenv"

// Compare checks target against ref and returns a sorted report.
func Compare(ref, target dotenv.File) Report {
	r := Report{Reference: ref.Path, Target: target.Path}

	refKeys, targetKeys := ref.Keys(), target.Keys()
	r.ReferenceKeys = len(refKeys)
	inRef, inTarget := keySet(refKeys), keySet(targetKeys)

	// Set differences, each ordered by first line in its file.
	var missing, unexpected []keyLine
	for _, k := range refKeys {
		if !inTarget[k] {
			missing = append(missing, keyLine{k, ref.FirstLine(k)})
		}
	}
	for _, k := range targetKeys {
		if !inRef[k] {
			unexpected = append(unexpected, keyLine{k, target.FirstLine(k)})
		}
	}

	paired := map[string]bool{}
	for _, m := range pair(missing, unexpected) {
		paired[m.Missing.Key], paired[m.Unexpected.Key] = true, true
		r.Findings = append(r.Findings, Finding{
			Category:   Misspelling,
			Key:        m.Unexpected.Key,
			Suggestion: m.Missing.Key,
			Locations:  []Location{{target.Path, m.Unexpected.Line}, {ref.Path, m.Missing.Line}},
		})
	}
	for _, m := range missing {
		if !paired[m.Key] {
			r.Findings = append(r.Findings, Finding{
				Category:  Missing,
				Key:       m.Key,
				Locations: []Location{{ref.Path, m.Line}},
			})
		}
	}

	for _, u := range unexpected {
		if !paired[u.Key] {
			r.Findings = append(r.Findings, Finding{
				Category:  Unexpected,
				Key:       u.Key,
				Locations: []Location{{target.Path, u.Line}},
			})
		}
	}

	for _, f := range []dotenv.File{ref, target} {
		r.Findings = append(r.Findings, duplicates(f)...)
		for _, pe := range f.Errors {
			r.Findings = append(r.Findings, Finding{
				Category:  Unparseable,
				Reason:    pe.Reason,
				Locations: []Location{{f.Path, pe.Line}},
			})
		}
	}
	r.Findings = append(r.Findings, empties(target)...)

	r.sort()
	return r
}

// duplicates reports keys assigned on more than one line of f.
func duplicates(f dotenv.File) []Finding {
	var out []Finding
	for _, k := range f.Keys() {
		if lines := f.Lines(k); len(lines) > 1 {
			out = append(out, Finding{Category: Duplicate, Key: k, Locations: locations(f.Path, lines)})
		}
	}
	return out
}

// empties reports, per key, the lines of f where the key has an empty value.
func empties(f dotenv.File) []Finding {
	var out []Finding
	for _, k := range f.Keys() {
		var lines []int
		for _, e := range f.Entries {
			if e.Key == k && e.Empty {
				lines = append(lines, e.Line)
			}
		}
		if len(lines) > 0 {
			out = append(out, Finding{Category: Empty, Key: k, Locations: locations(f.Path, lines)})
		}
	}
	return out
}

func locations(file string, lines []int) []Location {
	locs := make([]Location, len(lines))
	for i, l := range lines {
		locs[i] = Location{file, l}
	}
	return locs
}

func keySet(keys []string) map[string]bool {
	s := make(map[string]bool, len(keys))
	for _, k := range keys {
		s[k] = true
	}
	return s
}
