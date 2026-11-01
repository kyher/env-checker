package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

const sentinel = "SENTINEL_d0_not_leak"

// formats lists every supported --format value; each scenario has a
// <format>.golden file per entry.
var formats = []string{"text", "json"}

func testdata(name string) string {
	p, err := filepath.Abs(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		panic(err)
	}
	return p
}

// runIn runs the command with dir as the working directory.
func runIn(t *testing.T, dir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	t.Chdir(dir)
	var out, errb bytes.Buffer
	code = run(args, &out, &errb)
	return out.String(), errb.String(), code
}

func golden(t *testing.T, path, got string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	if got != string(want) {
		t.Errorf("output mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}

func formatArgs(format string) []string {
	if format == "text" {
		return nil
	}
	return []string{"--format", format}
}

// scenarios are testdata directories with both files and an expected exit code.
var scenarios = []struct {
	dir  string
	code int
}{
	{"clean", 0},
	{"missing", 1},
	{"misspell", 1},
	{"hygiene", 1},
	{"crlf_bom", 0},
	{"empty_files", 0},
	{"example_empty", 1},
	{"syntax", 0},
	{"unterminated", 1},
}

func TestScenarios(t *testing.T) {
	for _, sc := range scenarios {
		for _, format := range formats {
			t.Run(sc.dir+"/"+format, func(t *testing.T) {
				dir := testdata(sc.dir)
				stdout, stderr, code := runIn(t, dir, formatArgs(format)...)
				if code != sc.code {
					t.Errorf("exit code = %d, want %d (stderr: %q)", code, sc.code, stderr)
				}
				if stderr != "" {
					t.Errorf("unexpected stderr: %q", stderr)
				}
				golden(t, filepath.Join(dir, format+".golden"), stdout)
			})
		}
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name       string
		dir        string
		args       []string
		wantStderr []string
	}{
		{name: "missing env file", dir: "no_env", wantStderr: []string{"env-checker: open .env:"}},
		{name: "unknown flag", dir: "clean", args: []string{"--bogus"}, wantStderr: []string{"env-checker:", "Usage"}},
		{name: "positional argument", dir: "clean", args: []string{"foo"}, wantStderr: []string{"env-checker:", "Usage"}},
		{name: "bad format", dir: "clean", args: []string{"--format", "yaml"},
			wantStderr: []string{`env-checker: invalid --format "yaml": must be text or json`}},
		{name: "missing env file json", dir: "no_env", args: []string{"--format", "json"},
			wantStderr: []string{"env-checker: open .env:"}},
		{name: "missing example file", dir: "clean", args: []string{"--example", "nope.example"},
			wantStderr: []string{"env-checker: open nope.example:"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, code := runIn(t, testdata(tc.dir), tc.args...)
			if code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}
			if stdout != "" {
				t.Errorf("stdout should be empty, got %q", stdout)
			}
			for _, w := range tc.wantStderr {
				if !strings.Contains(stderr, w) {
					t.Errorf("stderr %q does not contain %q", stderr, w)
				}
			}
		})
	}
}

func TestCustomPaths(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")
	if err := os.Mkdir(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	example, env := filepath.Join(cfg, "app.env.example"), filepath.Join(cfg, "app.env")
	copyFile(t, filepath.Join(testdata("hygiene"), ".env.example"), example)
	copyFile(t, filepath.Join(testdata("hygiene"), ".env"), env)

	for _, dash := range []string{"--", "-"} {
		t.Run(dash, func(t *testing.T) {
			var out, errb bytes.Buffer
			code := run([]string{dash + "example", example, dash + "env", env}, &out, &errb)
			if code != 1 {
				t.Fatalf("exit code = %d, want 1 (stderr %q)", code, errb.String())
			}
			if !strings.Contains(out.String(), env+":3: unparseable") {
				t.Errorf("output should use the given path %q:\n%s", env, out.String())
			}
		})
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDeterministic(t *testing.T) {
	for _, sc := range scenarios {
		for _, format := range formats {
			t.Run(sc.dir+"/"+format, func(t *testing.T) {
				a, _, _ := runIn(t, testdata(sc.dir), formatArgs(format)...)
				b, _, _ := runIn(t, testdata(sc.dir), formatArgs(format)...)
				if a != b {
					t.Errorf("outputs differ:\n%s\n---\n%s", a, b)
				}
			})
		}
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, os.ErrClosed }

func TestWriteFailure(t *testing.T) {
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			t.Chdir(testdata("missing"))
			var errb bytes.Buffer
			code := run(formatArgs(format), failWriter{}, &errb)
			if code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}
			if !strings.HasPrefix(errb.String(), "env-checker: write report:") {
				t.Errorf("stderr = %q", errb.String())
			}
		})
	}
}

func TestVersionAndHelp(t *testing.T) {
	stdout, _, code := runIn(t, testdata("clean"), "--version")
	if code != 0 || stdout != "env-checker dev\n" {
		t.Errorf("--version: code %d, stdout %q", code, stdout)
	}
	stdout, _, code = runIn(t, testdata("clean"), "--help")
	if code != 0 || !strings.Contains(stdout, "Usage") {
		t.Errorf("--help: code %d, stdout %q", code, stdout)
	}
}

// TestNoSecretLeak runs every testdata scenario in every format, plus error
// paths, and asserts the sentinel value never appears (constitution Principle I).
func TestNoSecretLeak(t *testing.T) {
	entries, err := os.ReadDir(testdata(""))
	if err != nil {
		t.Fatal(err)
	}
	extra := [][]string{nil, {"--help"}, {"--bogus"}}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		for _, format := range formats {
			for _, x := range extra {
				args := append(formatArgs(format), x...)
				t.Run(ent.Name()+"/"+format+"/"+strings.Join(x, ""), func(t *testing.T) {
					stdout, stderr, _ := runIn(t, testdata(ent.Name()), args...)
					if strings.Contains(stdout+stderr, sentinel) {
						t.Errorf("sentinel leaked:\nstdout: %s\nstderr: %s", stdout, stderr)
					}
				})
			}
		}
	}
}
