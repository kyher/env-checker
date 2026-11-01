package check

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kyher/env-checker/internal/dotenv"
)

// largeFiles builds two 1,000-key files: 500 shared, 250 reference-only,
// 250 target-only (half of which are near-miss typos of reference-only keys).
func largeFiles(tb testing.TB) (ref, target dotenv.File) {
	tb.Helper()
	var r, t strings.Builder
	for i := range 500 {
		fmt.Fprintf(&r, "SHARED_KEY_%04d=\n", i)
		fmt.Fprintf(&t, "SHARED_KEY_%04d=v\n", i)
	}
	for i := range 250 {
		fmt.Fprintf(&r, "REFERENCE_ONLY_%04d=\n", i)
		if i%2 == 0 {
			fmt.Fprintf(&t, "REFERENCE_ONLX_%04d=v\n", i) // typo
		} else {
			fmt.Fprintf(&t, "TARGET_ONLY_%04d=v\n", i)
		}
	}
	var err error
	if ref, err = dotenv.Parse("ref", strings.NewReader(r.String())); err != nil {
		tb.Fatal(err)
	}
	if target, err = dotenv.Parse("target", strings.NewReader(t.String())); err != nil {
		tb.Fatal(err)
	}
	return ref, target
}

func BenchmarkCompare1000(b *testing.B) {
	ref, target := largeFiles(b)
	for b.Loop() {
		Compare(ref, target)
	}
}

// SC-005: 1,000 entries per file checked in under 1 second.
func TestCompare1000UnderOneSecond(t *testing.T) {
	if testing.Short() {
		t.Skip("timing test")
	}
	ref, target := largeFiles(t)
	start := time.Now()
	r := Compare(ref, target)
	if d := time.Since(start); d >= time.Second {
		t.Errorf("Compare took %v, want < 1s", d)
	}
	if got := r.Counts()[Misspelling]; got != 125 {
		t.Errorf("misspellings = %d, want 125", got)
	}
}
