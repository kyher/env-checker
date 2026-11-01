package dotenv

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func e(key string, line int, empty bool) Entry { return Entry{Key: key, Line: line, Empty: empty} }

func perr(line int, reason string) ParseError { return ParseError{Line: line, Reason: reason} }

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		entries []Entry
		errs    []ParseError
	}{
		{name: "empty input", in: ""},
		{name: "blank and comment lines", in: "\n# comment\n   # indented comment\n\t\n"},
		{name: "simple", in: "KEY=value\n", entries: []Entry{e("KEY", 1, false)}},
		{name: "spaces around equals", in: "KEY = value", entries: []Entry{e("KEY", 1, false)}},
		{name: "export prefix", in: "export KEY=value", entries: []Entry{e("KEY", 1, false)}},
		{name: "export tab prefix", in: "export\tKEY=value", entries: []Entry{e("KEY", 1, false)}},
		{name: "key named export", in: "export=1", entries: []Entry{e("export", 1, false)}},
		{name: "key starting with export", in: "exported=1", entries: []Entry{e("exported", 1, false)}},
		{name: "empty unquoted", in: "KEY=", entries: []Entry{e("KEY", 1, true)}},
		{name: "empty double quoted", in: `KEY=""`, entries: []Entry{e("KEY", 1, true)}},
		{name: "empty single quoted", in: `KEY=''`, entries: []Entry{e("KEY", 1, true)}},
		{name: "space in quotes is not empty", in: `KEY=" "`, entries: []Entry{e("KEY", 1, false)}},
		{name: "inline comment", in: "KEY=abc # note", entries: []Entry{e("KEY", 1, false)}},
		{name: "hash inside value", in: "KEY=a#b", entries: []Entry{e("KEY", 1, false)}},
		{name: "only comment after equals", in: "KEY= # only comment", entries: []Entry{e("KEY", 1, true)}},
		{name: "whitespace-only unquoted value", in: "KEY=   ", entries: []Entry{e("KEY", 1, true)}},
		{name: "equals in value", in: "KEY=a=b=c", entries: []Entry{e("KEY", 1, false)}},
		{name: "escaped double quote", in: `KEY="a\"b"`, entries: []Entry{e("KEY", 1, false)}},
		{name: "quoted with trailing comment", in: `KEY="v" # c`, entries: []Entry{e("KEY", 1, false)}},
		{name: "single quoted literal backslash", in: `KEY='a\'`, entries: []Entry{e("KEY", 1, false)}},
		{
			name:    "multi-line double quoted",
			in:      "KEY=\"line1\nline2\nline3\"\nNEXT=1\n",
			entries: []Entry{e("KEY", 1, false), e("NEXT", 4, false)},
		},
		{
			name:    "multi-line single quoted",
			in:      "A='multi\nline'\nB=1\n",
			entries: []Entry{e("A", 1, false), e("B", 3, false)},
		},
		{
			name:    "multi-line single quoted is literal",
			in:      "A='x\\\nline' # c\nB=\n",
			entries: []Entry{e("A", 1, false), e("B", 3, true)},
		},
		{
			name:    "single quote unterminated across lines",
			in:      "A='never\nB=1\nC=2\n",
			entries: []Entry{e("B", 2, false), e("C", 3, false)},
			errs:    []ParseError{perr(1, ReasonUnterminatedQuote)},
		},
		{
			name:    "duplicates kept",
			in:      "A=1\nA=2\n",
			entries: []Entry{e("A", 1, false), e("A", 2, false)},
		},
		{name: "missing equals", in: "this is not valid", errs: []ParseError{perr(1, ReasonMissingEquals)}},
		{name: "bare key", in: "KEY", errs: []ParseError{perr(1, ReasonMissingEquals)}},
		{name: "key starts with digit", in: "1KEY=x", errs: []ParseError{perr(1, ReasonInvalidKey)}},
		{name: "key with dash", in: "MY-KEY=x", errs: []ParseError{perr(1, ReasonInvalidKey)}},
		{name: "key with dot", in: "MY.KEY=x", errs: []ParseError{perr(1, ReasonInvalidKey)}},
		{name: "empty key", in: "=x", errs: []ParseError{perr(1, ReasonInvalidKey)}},
		{name: "trailing text after single quote", in: "KEY='it''s'", errs: []ParseError{perr(1, ReasonTrailingText)}},
		{name: "trailing text after double quote", in: `KEY="a"b`, errs: []ParseError{perr(1, ReasonTrailingText)}},
		{
			name:    "unterminated double quote resumes next line",
			in:      "KEY=\"unterminated\nNEXT=1\n",
			entries: []Entry{e("NEXT", 2, false)},
			errs:    []ParseError{perr(1, ReasonUnterminatedQuote)},
		},
		{
			name:    "unterminated single quote",
			in:      "KEY='unterminated\nNEXT=1",
			entries: []Entry{e("NEXT", 2, false)},
			errs:    []ParseError{perr(1, ReasonUnterminatedQuote)},
		},
		{
			name:    "errors do not stop parsing",
			in:      "A=1\nbad line\nB=2\n",
			entries: []Entry{e("A", 1, false), e("B", 3, false)},
			errs:    []ParseError{perr(2, ReasonMissingEquals)},
		},
		{
			name:    "BOM stripped",
			in:      "\xEF\xBB\xBFKEY=1\n",
			entries: []Entry{e("KEY", 1, false)},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			check := func(t *testing.T, in string) {
				t.Helper()
				f, err := Parse("x.env", strings.NewReader(in))
				if err != nil {
					t.Fatalf("Parse: unexpected error %v", err)
				}
				if f.Path != "x.env" {
					t.Errorf("Path = %q", f.Path)
				}
				if !slices.Equal(f.Entries, tc.entries) {
					t.Errorf("Entries = %+v, want %+v", f.Entries, tc.entries)
				}
				if !slices.Equal(f.Errors, tc.errs) {
					t.Errorf("Errors = %+v, want %+v", f.Errors, tc.errs)
				}
			}
			check(t, tc.in)
			t.Run("CRLF", func(t *testing.T) {
				check(t, strings.ReplaceAll(tc.in, "\n", "\r\n"))
			})
		})
	}
}

// Reasons must come from the fixed set so no line content (possibly a secret) can leak.
func TestParseReasonsAreFixed(t *testing.T) {
	allowed := []string{ReasonMissingEquals, ReasonInvalidKey, ReasonUnterminatedQuote, ReasonTrailingText}
	in := "SECRETVALUE\n1SECRET=SECRETVALUE\nK='SECRETVALUE'SECRETVALUE\nK=\"SECRETVALUE\n"
	f, err := Parse("x", strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Errors) != 4 {
		t.Fatalf("got %d errors, want 4: %+v", len(f.Errors), f.Errors)
	}
	for _, pe := range f.Errors {
		if !slices.Contains(allowed, pe.Reason) {
			t.Errorf("reason %q not in fixed set", pe.Reason)
		}
		if strings.Contains(pe.Reason, "SECRET") {
			t.Errorf("reason %q contains input text", pe.Reason)
		}
	}
}

// A very long value (e.g. an inlined certificate) is a valid entry and must
// not abort parsing.
func TestParseLongLine(t *testing.T) {
	in := "BIG=" + strings.Repeat("x", 2<<20) + "\nNEXT=1\n"
	f, err := Parse("x", strings.NewReader(in))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []Entry{e("BIG", 1, false), e("NEXT", 2, false)}
	if !slices.Equal(f.Entries, want) || len(f.Errors) != 0 {
		t.Errorf("Entries = %+v, Errors = %+v", f.Entries, f.Errors)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestParseReadError(t *testing.T) {
	_, err := Parse("x.env", errReader{})
	if err == nil || !strings.Contains(err.Error(), "boom") || !strings.Contains(err.Error(), "x.env") {
		t.Fatalf("err = %v, want wrapped read error naming file", err)
	}
}

func TestFileHelpers(t *testing.T) {
	f, err := Parse("x", strings.NewReader("B=1\nA=1\nB=2\nC=\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := f.Keys(); !slices.Equal(got, []string{"B", "A", "C"}) {
		t.Errorf("Keys = %v", got)
	}
	if got := f.Lines("B"); !slices.Equal(got, []int{1, 3}) {
		t.Errorf("Lines(B) = %v", got)
	}
	if got := f.FirstLine("B"); got != 1 {
		t.Errorf("FirstLine(B) = %d", got)
	}
	if got := f.FirstLine("Z"); got != 0 {
		t.Errorf("FirstLine(Z) = %d, want 0", got)
	}
}
