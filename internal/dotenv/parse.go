// Package dotenv parses .env-style files into keys and line numbers.
//
// Values are inspected only to decide whether they are empty and are then
// discarded; they are never stored (constitution Principle I).
package dotenv

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Fixed reasons for unparseable lines. Reasons never include line content.
const (
	ReasonMissingEquals     = `missing "="`
	ReasonInvalidKey        = "invalid key"
	ReasonUnterminatedQuote = "unterminated quote"
	ReasonTrailingText      = "unexpected text after closing quote"
)

const maxLineSize = 1 << 20

// Entry is one KEY=VALUE assignment.
//
// Entry deliberately has no value field (constitution Principle I).
type Entry struct {
	Key   string
	Line  int  // 1-based line where the entry starts
	Empty bool // value is empty after unquoting
}

// ParseError describes a line that could not be interpreted.
type ParseError struct {
	Line   int
	Reason string // one of the Reason* constants
}

// File is the parsed form of one dotenv file.
type File struct {
	Path    string
	Entries []Entry
	Errors  []ParseError
}

// Keys returns the unique keys in order of first appearance.
func (f File) Keys() []string {
	seen := make(map[string]bool, len(f.Entries))
	keys := make([]string, 0, len(f.Entries))
	for _, e := range f.Entries {
		if !seen[e.Key] {
			seen[e.Key] = true
			keys = append(keys, e.Key)
		}
	}
	return keys
}

// Lines returns every line on which key is assigned, ascending.
func (f File) Lines(key string) []int {
	var lines []int
	for _, e := range f.Entries {
		if e.Key == key {
			lines = append(lines, e.Line)
		}
	}
	return lines
}

// FirstLine returns the first line on which key is assigned, or 0 if absent.
func (f File) FirstLine(key string) int {
	for _, e := range f.Entries {
		if e.Key == key {
			return e.Line
		}
	}
	return 0
}

// Parse reads dotenv content from r. path is recorded on the result and used
// in error messages. Unparseable lines are collected in File.Errors; only
// read failures produce a non-nil error.
func Parse(path string, r io.Reader) (File, error) {
	lines, err := readLines(r)
	if err != nil {
		return File{}, fmt.Errorf("read %s: %w", path, err)
	}
	f := File{Path: path}
	for i := 0; i < len(lines); i++ {
		next, entry, reason := parseEntry(lines, i)
		if reason != "" {
			f.Errors = append(f.Errors, ParseError{Line: i + 1, Reason: reason})
		} else if entry != nil {
			f.Entries = append(f.Entries, *entry)
		}
		i = next
	}
	return f, nil
}

func readLines(r io.Reader) ([]string, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), maxLineSize)
	var lines []string
	for sc.Scan() {
		line := strings.TrimSuffix(sc.Text(), "\r")
		if len(lines) == 0 {
			line = strings.TrimPrefix(line, "\xEF\xBB\xBF")
		}
		lines = append(lines, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

const blanks = " \t"

// parseEntry parses the logical entry starting at lines[i]. It returns the
// index of the last physical line consumed, and either an entry, a failure
// reason, or neither (blank/comment line).
func parseEntry(lines []string, i int) (last int, entry *Entry, reason string) {
	start := i
	s := strings.TrimLeft(lines[i], blanks)
	if s == "" || s[0] == '#' {
		return i, nil, ""
	}
	if rest, ok := strings.CutPrefix(s, "export"); ok && rest != "" && strings.ContainsRune(blanks, rune(rest[0])) {
		s = strings.TrimLeft(rest, blanks)
	}
	eq := strings.IndexByte(s, '=')
	if eq < 0 {
		return i, nil, ReasonMissingEquals
	}
	key := strings.TrimRight(s[:eq], blanks)
	if !validKey(key) {
		return i, nil, ReasonInvalidKey
	}
	rest := strings.TrimLeft(s[eq+1:], blanks)

	var empty bool
	switch {
	case rest == "":
		empty = true
	case rest[0] == '\'':
		end := strings.IndexByte(rest[1:], '\'')
		if end < 0 {
			return i, nil, ReasonUnterminatedQuote
		}
		empty = end == 0
		if !onlyComment(rest[end+2:]) {
			return i, nil, ReasonTrailingText
		}
	case rest[0] == '"':
		var ok bool
		last, empty, ok = scanDoubleQuoted(lines, i, rest[1:])
		if !ok {
			return i, nil, ReasonUnterminatedQuote
		}
		if last < 0 {
			return i, nil, ReasonTrailingText
		}
		i = last
	default:
		empty = unquotedEmpty(rest)
	}
	return i, &Entry{Key: key, Line: start + 1, Empty: empty}, ""
}

// scanDoubleQuoted scans a double-quoted value beginning just after the
// opening quote in s (which is on lines[i]). It returns the index of the line
// holding the closing quote, whether the value is empty, and ok=false if the
// quote is never closed. If text other than a comment follows the closing
// quote, last is -1.
func scanDoubleQuoted(lines []string, i int, s string) (last int, empty bool, ok bool) {
	empty = true
	for {
		for p := 0; p < len(s); p++ {
			switch s[p] {
			case '\\':
				empty = false
				p++
			case '"':
				if !onlyComment(s[p+1:]) {
					return -1, empty, true
				}
				return i, empty, true
			default:
				empty = false
			}
		}
		i++
		if i >= len(lines) {
			return 0, false, false
		}
		empty = false // the newline itself is content
		s = lines[i]
	}
}

// onlyComment reports whether s is empty, whitespace, or a whitespace-led comment.
func onlyComment(s string) bool {
	s = strings.TrimLeft(s, blanks)
	return s == "" || s[0] == '#'
}

// unquotedEmpty reports whether an unquoted value (already left-trimmed) is
// empty once an inline comment (a '#' at the start or preceded by whitespace)
// and trailing whitespace are removed.
func unquotedEmpty(s string) bool {
	for p := 0; p < len(s); p++ {
		if s[p] == '#' && (p == 0 || s[p-1] == ' ' || s[p-1] == '\t') {
			s = s[:p]
			break
		}
	}
	return strings.TrimRight(s, blanks) == ""
}

func validKey(k string) bool {
	if k == "" {
		return false
	}
	for i := 0; i < len(k); i++ {
		c := k[i]
		switch {
		case c == '_', c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}
