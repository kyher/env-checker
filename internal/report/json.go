package report

import (
	"encoding/json"
	"io"

	"github.com/kyher/env-checker/internal/check"
)

// SchemaVersion is the JSON report schema version (contracts/json-report.md).
const SchemaVersion = 1

type jsonReport struct {
	SchemaVersion int           `json:"schema_version"`
	OK            bool          `json:"ok"`
	Reference     string        `json:"reference"`
	Target        string        `json:"target"`
	Findings      []jsonFinding `json:"findings"`
	Summary       jsonSummary   `json:"summary"`
}

type jsonFinding struct {
	Category   string         `json:"category"`
	Certainty  string         `json:"certainty"`
	Key        string         `json:"key,omitempty"`
	Suggestion string         `json:"suggestion,omitempty"`
	Reason     string         `json:"reason,omitempty"`
	Locations  []jsonLocation `json:"locations"`
}

type jsonLocation struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

type jsonSummary struct {
	Total             int `json:"total"`
	Unparseable       int `json:"unparseable"`
	Missing           int `json:"missing"`
	LikelyMisspelling int `json:"likely_misspelling"`
	Duplicate         int `json:"duplicate"`
	Empty             int `json:"empty"`
	Unexpected        int `json:"unexpected"`
}

// WriteJSON writes the machine-readable report as a single line.
func WriteJSON(w io.Writer, r check.Report) error {
	c := r.Counts()
	out := jsonReport{
		SchemaVersion: SchemaVersion,
		OK:            r.OK(),
		Reference:     r.Reference,
		Target:        r.Target,
		Findings:      make([]jsonFinding, 0, len(r.Findings)),
		Summary: jsonSummary{
			Total:             len(r.Findings),
			Unparseable:       c[check.Unparseable],
			Missing:           c[check.Missing],
			LikelyMisspelling: c[check.Misspelling],
			Duplicate:         c[check.Duplicate],
			Empty:             c[check.Empty],
			Unexpected:        c[check.Unexpected],
		},
	}
	for _, f := range r.Findings {
		locs := make([]jsonLocation, len(f.Locations))
		for i, l := range f.Locations {
			locs[i] = jsonLocation{File: l.File, Line: l.Line}
		}
		out.Findings = append(out.Findings, jsonFinding{
			Category:   f.Category.String(),
			Certainty:  f.Category.Certainty(),
			Key:        f.Key,
			Suggestion: f.Suggestion,
			Reason:     f.Reason,
			Locations:  locs,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(out)
}
