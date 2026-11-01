package check

import (
	"math/rand/v2"
	"slices"
	"testing"
)

func TestDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"ABC", "ABC", 0},
		{"", "ABC", 3},
		{"ABCD", "ABDC", 1}, // adjacent transposition
		{"ABCD", "ABD", 1},
		{"ABCD", "ABXD", 1},
		{"CA", "ABC", 3}, // OSA, not unrestricted Damerau (which gives 2)
		{"KITTEN", "SITTING", 3},
	}
	for _, tc := range tests {
		if got := distance(tc.a, tc.b); got != tc.want {
			t.Errorf("distance(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
		if got := distance(tc.b, tc.a); got != tc.want {
			t.Errorf("distance(%q, %q) = %d, want %d (symmetry)", tc.b, tc.a, got, tc.want)
		}
	}
}

// Realistic typos: at least 90% must be detected (SC-002).
var typoCorpus = [][2]string{
	{"DATABASE_URL", "DATABSE_URL"},
	{"API_KEY", "api_key"},
	{"API_KEY", "APIKEY"},
	{"SECRET_KEY", "SECERT_KEY"},
	{"REDIS_HOST", "REDIS_HSOT"},
	{"PORT", "PROT"},
	{"AWS_REGION", "AWS_REGOIN"},
	{"SMTP_PASSWORD", "SMTP_PASSWROD"},
	{"LOG_LEVEL", "LOG_LEVLE"},
	{"JWT_SECRET", "JWT_SECRT"},
	{"DB_PASSWORD", "DB_PASSWORDD"},
	{"STRIPE_KEY", "STRIPE_KYE"},
	{"NODE_ENV", "NODE_EVN"},
	{"SENTRY_DSN", "SENTRY_DNS"},
	{"MAIL_FROM", "MAIL_FORM"},
	{"BUCKET_NAME", "BUCKET_NMAE"},
	{"TIMEOUT_MS", "TIMEOUT_SM"},
	{"CACHE_TTL", "CAHCE_TTL"},
	{"APP_URL", "APP_UR"},
	{"HOST", "host"},
}

// Genuinely different keys: fewer than 5% may be flagged (SC-003).
var nonTypoCorpus = [][2]string{
	{"PORT", "HOST"},
	{"DB_HOST", "DB_PORT"},
	{"API_KEY", "API_URL"},
	{"DB", "DC"},
	{"ID", "IP"},
	{"REDIS_HOST", "PORT"},
	{"AWS_KEY", "GCP_KEY"},
	{"LOG_LEVEL", "LOG_FILE"},
	{"SMTP_HOST", "SMTP_USER"},
	{"CACHE_TTL", "CACHE_DIR"},
	{"APP_NAME", "APP_ENV"},
	{"MAIL_FROM", "MAIL_TO"},
	{"S3_BUCKET", "GCS_BUCKET"},
	{"JWT_SECRET", "JWT_EXPIRY"},
	{"NODE_ENV", "NODE_PATH"},
	{"DEBUG", "LEVEL"},
	{"USER", "PASS"},
	{"TZ", "LANG"},
	{"DB_NAME", "DB_USER"},
	{"HTTP_PORT", "GRPC_PORT"},
}

func TestSimilarCorpora(t *testing.T) {
	if len(typoCorpus) < 20 || len(nonTypoCorpus) < 20 {
		t.Fatal("corpora must each have at least 20 pairs")
	}
	hits := 0
	for _, p := range typoCorpus {
		if _, ok := similar(p[0], p[1]); ok {
			hits++
		} else {
			t.Logf("typo not detected: %s / %s", p[0], p[1])
		}
	}
	if rate := float64(hits) / float64(len(typoCorpus)); rate < 0.90 {
		t.Errorf("typo detection rate %.0f%%, want >= 90%%", rate*100)
	}
	falsePos := 0
	for _, p := range nonTypoCorpus {
		if d, ok := similar(p[0], p[1]); ok {
			falsePos++
			t.Logf("false positive: %s / %s (distance %d)", p[0], p[1], d)
		}
	}
	if rate := float64(falsePos) / float64(len(nonTypoCorpus)); rate >= 0.05 {
		t.Errorf("false-positive rate %.0f%%, want < 5%%", rate*100)
	}
}

func TestSimilarThresholds(t *testing.T) {
	tests := []struct {
		a, b     string
		wantDist int
		wantOK   bool
	}{
		{"API_KEY", "api_key", 0, true}, // case-only
		{"DB", "db", 0, true},           // case-only applies regardless of length
		{"ABC", "ABD", 1, false},        // shorter < 4
		{"ABCD", "ABCE", 1, true},       // 4..7 allows 1
		{"ABCD", "ABEF", 2, false},
		{"ABCDEFG", "ABCDEXY", 2, false},  // 7 chars, 2 edits
		{"ABCDEFGH", "ABCDEFXY", 2, true}, // 8 chars allows 2
		{"ABCDEFGH", "ABCDEXYZ", 3, false},
		{"ABCDEFGH", "ABCDEFGHIJK", 3, false}, // length gap alone exceeds threshold
	}
	for _, tc := range tests {
		d, ok := similar(tc.a, tc.b)
		if ok != tc.wantOK || (ok && d != tc.wantDist) {
			t.Errorf("similar(%q, %q) = (%d, %v), want (%d, %v)", tc.a, tc.b, d, ok, tc.wantDist, tc.wantOK)
		}
	}
}

func kl(key string, line int) keyLine { return keyLine{Key: key, Line: line} }

func TestPair(t *testing.T) {
	tests := []struct {
		name       string
		missing    []keyLine
		unexpected []keyLine
		want       []match
	}{
		{
			name:       "closest wins",
			missing:    []keyLine{kl("DB_HOST", 1), kl("DB_HOSTS", 2)},
			unexpected: []keyLine{kl("DB_HOSTSS", 1)}, // dist 2 to DB_HOST, 1 to DB_HOSTS
			want:       []match{{kl("DB_HOSTS", 2), kl("DB_HOSTSS", 1)}},
		},
		{
			name:       "tie broken by earlier reference line",
			missing:    []keyLine{kl("DB_HOST", 1), kl("DB_HOSTS", 2)},
			unexpected: []keyLine{kl("DB_HOTS", 1)}, // dist 1 to both
			want:       []match{{kl("DB_HOST", 1), kl("DB_HOTS", 1)}},
		},
		{
			name:       "two candidates for one missing key, earlier target line wins tie",
			missing:    []keyLine{kl("DATABASE_URL", 1)},
			unexpected: []keyLine{kl("DATABAS_URL", 5), kl("DATABSE_URL", 3)},
			want:       []match{{kl("DATABASE_URL", 1), kl("DATABSE_URL", 3)}},
		},
		{
			name:       "closer candidate beats earlier line",
			missing:    []keyLine{kl("DATABASE_URL", 1)},
			unexpected: []keyLine{kl("DATABSE_UR", 2), kl("DATABSE_URL", 3)},
			want:       []match{{kl("DATABASE_URL", 1), kl("DATABSE_URL", 3)}},
		},
		{
			name:       "no match",
			missing:    []keyLine{kl("PORT", 1)},
			unexpected: []keyLine{kl("REDIS_HOST", 1)},
			want:       nil,
		},
		{
			name:       "each key used once",
			missing:    []keyLine{kl("API_KEY", 1), kl("API_KEYS", 2)},
			unexpected: []keyLine{kl("API_KEX", 1), kl("API_KEYX", 2)},
			want:       []match{{kl("API_KEY", 1), kl("API_KEX", 1)}, {kl("API_KEYS", 2), kl("API_KEYX", 2)}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pair(tc.missing, tc.unexpected)
			if !slices.Equal(got, tc.want) {
				t.Errorf("pair = %+v, want %+v", got, tc.want)
			}
			seen := map[string]bool{}
			for _, m := range got {
				if seen["m"+m.Missing.Key] || seen["u"+m.Unexpected.Key] {
					t.Errorf("key used twice in %+v", got)
				}
				seen["m"+m.Missing.Key], seen["u"+m.Unexpected.Key] = true, true
			}
		})
	}
}

func TestPairDeterministic(t *testing.T) {
	missing := []keyLine{kl("DB_HOST", 1), kl("DB_HOSTS", 2), kl("API_KEY", 3), kl("SECRET_KEY", 4)}
	unexpected := []keyLine{kl("DB_HOTS", 1), kl("api_key", 2), kl("SECERT_KEY", 3), kl("DB_HSOTS", 4)}
	want := pair(missing, unexpected)
	rng := rand.New(rand.NewPCG(1, 2))
	for range 50 {
		m, u := slices.Clone(missing), slices.Clone(unexpected)
		rng.Shuffle(len(m), func(i, j int) { m[i], m[j] = m[j], m[i] })
		rng.Shuffle(len(u), func(i, j int) { u[i], u[j] = u[j], u[i] })
		if got := pair(m, u); !slices.Equal(got, want) {
			t.Fatalf("pair not deterministic: got %+v, want %+v", got, want)
		}
	}
}
