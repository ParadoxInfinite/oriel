package secrets

import (
	"slices"
	"testing"
)

func TestIsSensitive(t *testing.T) {
	cases := []struct {
		key, val string
		want     bool
	}{
		{"OPENAI_API_KEY", "sk-proj-abc", true},     // name
		{"DATABASE_PASSWORD", "hunter2", true},      // name
		{"GITHUB_TOKEN", "x", true},                 // name
		{"AWS_SECRET_ACCESS_KEY", "x", true},        // name
		{"FOO", "ghp_0123456789abcdef", true},       // value prefix
		{"FOO", "AKIAIOSFODNN7EXAMPLE", true},       // value prefix
		{"FOO", "eyJhbGciOiJIUzI1NiJ9.payload", true}, // jwt-ish
		{"RANDOM", "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6", true}, // high-entropy
		{"NODE_ENV", "production", false},
		{"PATH", "/usr/local/bin:/usr/bin:/bin", false}, // path, not masked
		{"PORT", "3000", false},
		{"GREETING", "hello world this is a long but spaced sentence", false},
	}
	for _, c := range cases {
		if got := IsSensitive(c.key, c.val); got != c.want {
			t.Errorf("IsSensitive(%q,%q)=%v want %v", c.key, c.val, got, c.want)
		}
	}
}

func TestMaskEnvAll(t *testing.T) {
	in := []string{"PATH=/usr/bin", "API_KEY=sk-secret", "EMPTY=", "NOEQUALS"}
	got := MaskEnv(in, MaskAll)
	want := []string{"PATH=••••••••", "API_KEY=••••••••", "EMPTY=", "NOEQUALS"}
	if !slices.Equal(got, want) {
		t.Errorf("MaskEnv(all) = %v, want %v", got, want)
	}
	// Input must not be mutated.
	if in[0] != "PATH=/usr/bin" {
		t.Errorf("MaskEnv mutated input: %q", in[0])
	}
}

func TestMaskEnvSensitive(t *testing.T) {
	in := []string{"PATH=/usr/bin", "API_KEY=sk-secret", "NODE_ENV=production"}
	got := MaskEnv(in, MaskSensitive)
	want := []string{"PATH=/usr/bin", "API_KEY=••••••••", "NODE_ENV=production"}
	if !slices.Equal(got, want) {
		t.Errorf("MaskEnv(sensitive) = %v, want %v", got, want)
	}
}

func TestMaskEnvOff(t *testing.T) {
	in := []string{"API_KEY=sk-secret"}
	if got := MaskEnv(in, MaskOff); !slices.Equal(got, in) {
		t.Errorf("MaskEnv(off) = %v, want unchanged", got)
	}
}

func TestParseMode(t *testing.T) {
	for in, want := range map[string]Mode{
		"all": MaskAll, "sensitive": MaskSensitive, "off": MaskOff, "": MaskAll, "bogus": MaskAll,
	} {
		if got := ParseMode(in); got != want {
			t.Errorf("ParseMode(%q)=%v want %v", in, got, want)
		}
	}
}
