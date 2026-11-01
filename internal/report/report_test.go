package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/kyher/env-checker/internal/check"
)

const (
	ref = ".env.example"
	tgt = ".env"
)

func at(file string, lines ...int) []check.Location {
	locs := make([]check.Location, len(lines))
	for i, l := range lines {
		locs[i] = check.Location{File: file, Line: l}
	}
	return locs
}

func rep(findings ...check.Finding) check.Report {
	return check.Report{Reference: ref, Target: tgt, ReferenceKeys: 12, Findings: findings}
}

func TestWriteText(t *testing.T) {
	tests := []struct {
		name string
		r    check.Report
		want string
	}{
		{
			name: "ok",
			r:    rep(),
			want: "OK: all 12 keys in .env.example are present in .env\n",
		},
		{
			name: "one missing",
			r:    rep(check.Finding{Category: check.Missing, Key: "DB_PORT", Locations: at(ref, 3)}),
			want: ".env.example:3: missing: DB_PORT\n\n1 issue: 1 missing\n",
		},
		{
			name: "two missing",
			r: rep(
				check.Finding{Category: check.Missing, Key: "A", Locations: at(ref, 1)},
				check.Finding{Category: check.Missing, Key: "B", Locations: at(ref, 2)},
			),
			want: ".env.example:1: missing: A\n.env.example:2: missing: B\n\n2 issues: 2 missing\n",
		},
		{
			name: "misspellings",
			r: rep(
				check.Finding{Category: check.Misspelling, Key: "DATABSE_URL", Suggestion: "DATABASE_URL",
					Locations: []check.Location{{File: tgt, Line: 4}, {File: ref, Line: 2}}},
				check.Finding{Category: check.Misspelling, Key: "api_key", Suggestion: "API_KEY",
					Locations: []check.Location{{File: tgt, Line: 7}, {File: ref, Line: 5}}},
			),
			want: ".env:4: likely misspelling: DATABSE_URL → did you mean DATABASE_URL? (.env.example:2)\n" +
				".env:7: likely misspelling: api_key → did you mean API_KEY? (.env.example:5)\n" +
				"\n2 issues: 2 likely misspellings\n",
		},
		{
			name: "one misspelling singular label",
			r: rep(check.Finding{Category: check.Misspelling, Key: "PROT", Suggestion: "PORT",
				Locations: []check.Location{{File: tgt, Line: 1}, {File: ref, Line: 1}}}),
			want: ".env:1: likely misspelling: PROT → did you mean PORT? (.env.example:1)\n\n1 issue: 1 likely misspelling\n",
		},
		{
			name: "all categories",
			r: rep(
				check.Finding{Category: check.Unparseable, Reason: `missing "="`, Locations: at(tgt, 14)},
				check.Finding{Category: check.Missing, Key: "DB_PORT", Locations: at(ref, 3)},
				check.Finding{Category: check.Misspelling, Key: "DATABSE_URL", Suggestion: "DATABASE_URL",
					Locations: []check.Location{{File: tgt, Line: 4}, {File: ref, Line: 2}}},
				check.Finding{Category: check.Misspelling, Key: "api_key", Suggestion: "API_KEY",
					Locations: []check.Location{{File: tgt, Line: 7}, {File: ref, Line: 5}}},
				check.Finding{Category: check.Duplicate, Key: "PORT", Locations: at(tgt, 2, 9)},
				check.Finding{Category: check.Empty, Key: "SECRET_TOKEN", Locations: at(tgt, 11)},
				check.Finding{Category: check.Unexpected, Key: "LEGACY_FLAG", Locations: at(tgt, 12)},
			),
			want: `.env:14: unparseable: missing "="` + "\n" +
				".env.example:3: missing: DB_PORT\n" +
				".env:4: likely misspelling: DATABSE_URL → did you mean DATABASE_URL? (.env.example:2)\n" +
				".env:7: likely misspelling: api_key → did you mean API_KEY? (.env.example:5)\n" +
				".env:2,9: duplicate: PORT\n" +
				".env:11: empty: SECRET_TOKEN\n" +
				".env:12: unexpected: LEGACY_FLAG\n" +
				"\n7 issues: 1 unparseable, 1 missing, 2 likely misspellings, 1 duplicate, 1 empty, 1 unexpected\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteText(&buf, tc.r); err != nil {
				t.Fatal(err)
			}
			if got := buf.String(); got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

type failWriter struct{}

var errWrite = errors.New("write failed")

func (failWriter) Write([]byte) (int, error) { return 0, errWrite }

func TestWriteTextError(t *testing.T) {
	r := rep(check.Finding{Category: check.Missing, Key: "A", Locations: at(ref, 1)})
	if err := WriteText(failWriter{}, r); !errors.Is(err, errWrite) {
		t.Fatalf("err = %v, want %v", err, errWrite)
	}
}

func TestWriteJSONEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, rep()); err != nil {
		t.Fatal(err)
	}
	want := `{"schema_version":1,"ok":true,"reference":".env.example","target":".env","findings":[],` +
		`"summary":{"total":0,"unparseable":0,"missing":0,"likely_misspelling":0,"duplicate":0,"empty":0,"unexpected":0}}` + "\n"
	if got := buf.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestWriteJSONFindings(t *testing.T) {
	r := rep(
		check.Finding{Category: check.Unparseable, Reason: `missing "="`, Locations: at(tgt, 14)},
		check.Finding{Category: check.Missing, Key: "DB_PORT", Locations: at(ref, 3)},
		check.Finding{Category: check.Misspelling, Key: "DATABSE_URL", Suggestion: "DATABASE_URL",
			Locations: []check.Location{{File: tgt, Line: 4}, {File: ref, Line: 2}}},
		check.Finding{Category: check.Duplicate, Key: "PORT", Locations: at(tgt, 2, 9)},
	)
	var buf bytes.Buffer
	if err := WriteJSON(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Count(out, "\n") != 1 || !strings.HasSuffix(out, "\n") {
		t.Errorf("output must be a single line ending in newline: %q", out)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["schema_version"] != 1.0 || doc["ok"] != false || doc["reference"] != ref || doc["target"] != tgt {
		t.Errorf("top-level fields wrong: %v", doc)
	}
	findings := doc["findings"].([]any)
	if len(findings) != 4 {
		t.Fatalf("findings len = %d", len(findings))
	}
	f0 := findings[0].(map[string]any)
	if f0["category"] != "unparseable" || f0["certainty"] != "definite" || f0["reason"] != `missing "="` {
		t.Errorf("unparseable finding = %v", f0)
	}
	if _, ok := f0["key"]; ok {
		t.Errorf("unparseable finding must omit key: %v", f0)
	}
	f1 := findings[1].(map[string]any)
	if _, ok := f1["suggestion"]; ok {
		t.Errorf("missing finding must omit suggestion: %v", f1)
	}
	if _, ok := f1["reason"]; ok {
		t.Errorf("missing finding must omit reason: %v", f1)
	}
	f2 := findings[2].(map[string]any)
	if f2["category"] != "likely_misspelling" || f2["certainty"] != "suggestion" || f2["suggestion"] != "DATABASE_URL" {
		t.Errorf("misspelling finding = %v", f2)
	}
	locs := f2["locations"].([]any)
	if len(locs) != 2 || locs[0].(map[string]any)["file"] != tgt || locs[1].(map[string]any)["line"] != 2.0 {
		t.Errorf("misspelling locations = %v", locs)
	}
	sum := doc["summary"].(map[string]any)
	if sum["total"] != 4.0 || sum["missing"] != 1.0 || sum["unexpected"] != 0.0 || len(sum) != 7 {
		t.Errorf("summary = %v", sum)
	}
}

func TestWriteJSONError(t *testing.T) {
	if err := WriteJSON(failWriter{}, rep()); !errors.Is(err, errWrite) {
		t.Fatalf("err = %v, want %v", err, errWrite)
	}
}
