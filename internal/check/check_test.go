package check

import (
	"slices"
	"strings"
	"testing"

	"github.com/kyher/env-checker/internal/dotenv"
)

const (
	refPath = ".env.example"
	tgtPath = ".env"
)

func parse(t *testing.T, path, content string) dotenv.File {
	t.Helper()
	f, err := dotenv.Parse(path, strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func compare(t *testing.T, ref, tgt string) Report {
	t.Helper()
	return Compare(parse(t, refPath, ref), parse(t, tgtPath, tgt))
}

// summary renders findings compactly for table comparisons.
func summary(r Report) []string {
	out := make([]string, 0, len(r.Findings))
	for _, f := range r.Findings {
		var b strings.Builder
		b.WriteString(f.Category.String())
		b.WriteString(" ")
		if f.Key != "" {
			b.WriteString(f.Key)
		} else {
			b.WriteString("[" + f.Reason + "]")
		}
		if f.Suggestion != "" {
			b.WriteString("->" + f.Suggestion)
		}
		for _, l := range f.Locations {
			b.WriteString(" " + l.File + ":" + itoa(l.Line))
		}
		out = append(out, b.String())
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for ; n > 0; n /= 10 {
		d = append([]byte{byte('0' + n%10)}, d...)
	}
	return string(d)
}

func TestCompareMissing(t *testing.T) {
	tests := []struct {
		name     string
		ref, tgt string
		want     []string
	}{
		{name: "all present", ref: "A=\nB=\n", tgt: "B=1\nA=1\n", want: []string{}},
		{name: "both empty", ref: "", tgt: "", want: []string{}},
		{
			name: "single missing",
			ref:  "DB_HOST=\nDB_PORT=\nAPI_KEY=\n", tgt: "DB_HOST=x\nAPI_KEY=y\n",
			want: []string{"missing DB_PORT .env.example:2"},
		},
		{
			name: "multiple missing sorted by reference line",
			ref:  "# c\nZETA=\nALPHA=\nMID=\n", tgt: "",
			want: []string{"missing ZETA .env.example:2", "missing ALPHA .env.example:3", "missing MID .env.example:4"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := compare(t, tc.ref, tc.tgt)
			if got := summary(r); !slices.Equal(got, tc.want) {
				t.Errorf("findings:\n got  %q\n want %q", got, tc.want)
			}
		})
	}
}

func TestCompareReportMetadata(t *testing.T) {
	r := compare(t, "A=\nB=\nA=\n", "A=1\n")
	if r.Reference != refPath || r.Target != tgtPath {
		t.Errorf("paths = %q, %q", r.Reference, r.Target)
	}
	if r.ReferenceKeys != 2 {
		t.Errorf("ReferenceKeys = %d, want 2", r.ReferenceKeys)
	}
}

func TestCompareMisspelling(t *testing.T) {
	tests := []struct {
		name     string
		ref, tgt string
		want     []string
	}{
		{
			name: "misspell scenario",
			ref:  "DATABASE_URL=\nAPI_KEY=\nPORT=\n",
			tgt:  "DATABSE_URL=x\napi_key=x\nREDIS_HOST=r\n",
			want: []string{
				"missing PORT .env.example:3",
				"likely_misspelling DATABSE_URL->DATABASE_URL .env:1 .env.example:1",
				"likely_misspelling api_key->API_KEY .env:2 .env.example:2",
			},
		},
		{
			name: "paired key is not also missing",
			ref:  "SECRET_KEY=\n", tgt: "SECERT_KEY=1\n",
			want: []string{"likely_misspelling SECERT_KEY->SECRET_KEY .env:1 .env.example:1"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := compare(t, tc.ref, tc.tgt)
			got := summary(r)
			// US3 adds unexpected findings; only compare the categories under test here.
			got = slices.DeleteFunc(got, func(s string) bool { return strings.HasPrefix(s, "unexpected ") })
			if !slices.Equal(got, tc.want) {
				t.Errorf("findings:\n got  %q\n want %q", got, tc.want)
			}
		})
	}
}

func TestCompareHygiene(t *testing.T) {
	tests := []struct {
		name     string
		ref, tgt string
		want     []string
	}{
		{
			name: "hygiene scenario full ordering",
			ref:  "PORT=\nTOKEN=\n",
			tgt:  "PORT=1\nTOKEN=\nthis is not valid\nPORT=2\nLEGACY_FLAG=1\n",
			want: []string{
				`unparseable [missing "="] .env:3`,
				"duplicate PORT .env:1 .env:4",
				"empty TOKEN .env:2",
				"unexpected LEGACY_FLAG .env:5",
			},
		},
		{
			name: "unexpected after pairing",
			ref:  "DATABASE_URL=\nAPI_KEY=\nPORT=\n",
			tgt:  "DATABSE_URL=x\napi_key=x\nREDIS_HOST=r\n",
			want: []string{
				"missing PORT .env.example:3",
				"likely_misspelling DATABSE_URL->DATABASE_URL .env:1 .env.example:1",
				"likely_misspelling api_key->API_KEY .env:2 .env.example:2",
				"unexpected REDIS_HOST .env:3",
			},
		},
		{
			name: "duplicate in reference counts once",
			ref:  "A=\nB=\nA=\n", tgt: "A=1\nB=1\n",
			want: []string{"duplicate A .env.example:1 .env.example:3"},
		},
		{
			name: "duplicate with only one empty entry",
			ref:  "K=\n", tgt: "K=1\nK=\n",
			want: []string{"duplicate K .env:1 .env:2", "empty K .env:2"},
		},
		{
			name: "duplicate both empty",
			ref:  "K=\n", tgt: "K=\nK=\n",
			want: []string{"duplicate K .env:1 .env:2", "empty K .env:1 .env:2"},
		},
		{
			name: "empty reference values are not reported",
			ref:  "A=\nB=''\n", tgt: "A=1\nB=2\n",
			want: []string{},
		},
		{
			name: "empty on misspelled and unexpected keys",
			ref:  "SECRET_KEY=\n", tgt: "SECERT_KEY=\nOTHER=\n",
			want: []string{
				"likely_misspelling SECERT_KEY->SECRET_KEY .env:1 .env.example:1",
				"empty SECERT_KEY .env:1",
				"empty OTHER .env:2",
				"unexpected OTHER .env:2",
			},
		},
		{
			name: "unparseable in both files, reference first",
			ref:  "A=\n1BAD=x\n", tgt: "A=1\nbad\n",
			want: []string{
				"unparseable [invalid key] .env.example:2",
				`unparseable [missing "="] .env:2`,
			},
		},
		{
			name: "empty reference, keys in target",
			ref:  "", tgt: "X=1\nY=2\n",
			want: []string{"unexpected X .env:1", "unexpected Y .env:2"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := compare(t, tc.ref, tc.tgt)
			if got := summary(r); !slices.Equal(got, tc.want) {
				t.Errorf("findings:\n got  %q\n want %q", got, tc.want)
			}
		})
	}
}
