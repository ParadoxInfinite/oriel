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
		{"OPENAI_API_KEY", "sk-proj-abc", true},                          // name
		{"DATABASE_PASSWORD", "hunter2", true},                           // name
		{"GITHUB_TOKEN", "x", true},                                      // name
		{"AWS_SECRET_ACCESS_KEY", "x", true},                             // name
		{"FOO", "ghp_0123456789abcdef", true},                            // value prefix
		{"FOO", "AKIAIOSFODNN7EXAMPLE", true},                            // value prefix
		{"FOO", "eyJhbGciOiJIUzI1NiJ9.payload", true},                    // jwt-ish
		{"RANDOM", "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6", true},             // high-entropy
		{"DB_PWD", "short", true},                                        // name: PWD
		{"WEBHOOK_URL", "https://hooks.example.com/x", true},             // name: WEBHOOK
		{"FOO", "abcdefabcdefabcdefabcdefabcdef", true},                  // 30-char all-hex token (no digit)
		{"FOO", "aGVsbG8vd29ybGQvc2VjcmV0L3Rva2Vu", true},                // base64 with internal slash
		{"DATABASE_URL", "postgres://user:pass@localhost:5432/db", true}, // url userinfo creds
		{"REDIS_URL", "redis://:secret@redis:6379", true},                // url password-only userinfo
		{"MONGO_URI", "mongodb+srv://u:p@cluster0.example/db", true},     // url userinfo creds
		{"NODE_ENV", "production", false},
		{"PUBLIC_URL", "https://example.com/path", false},             // plain url, no creds
		{"CALLBACK", "postgres://localhost/db", false},                // dsn, no creds
		{"FOO", "https://host/long/path@version/segment", false},      // '@' in path, not userinfo
		{"PATH", "/usr/local/bin:/usr/bin:/bin", false},               // path (has ':')
		{"HOME", "/Users/apple/some/long/path/to/a/file/here", false}, // path (leading '/')
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

func TestMaskCommand(t *testing.T) {
	cases := []struct {
		in   string
		mode Mode
		want string
	}{
		{"server --token=sk-abc --port 8080", MaskSensitive, "server --token=•••••••• --port 8080"},
		{"app --password=hunter2 run", MaskAll, "app --password=•••••••• run"},
		{"loader ghp_0123456789abcdef --v", MaskSensitive, "loader •••••••• --v"},
		{"server --token=sk-abc", MaskOff, "server --token=sk-abc"},
		{"plain --port 8080 --name web", MaskAll, "plain --port 8080 --name web"},
		{"", MaskAll, ""},
	}
	for _, c := range cases {
		if got := MaskCommand(c.in, c.mode); got != c.want {
			t.Errorf("MaskCommand(%q,%v)=%q want %q", c.in, c.mode, got, c.want)
		}
	}
}

func TestMaskLabels(t *testing.T) {
	in := map[string]string{"com.docker.compose.project": "myapp", "api_token": "sk-secret", "empty": ""}
	got := MaskLabels(in, MaskAll)
	if got["com.docker.compose.project"] != "myapp" {
		t.Errorf("metadata label masked: %q", got["com.docker.compose.project"])
	}
	if got["api_token"] != masked {
		t.Errorf("sensitive label not masked: %q", got["api_token"])
	}
	if got["empty"] != "" {
		t.Errorf("empty label changed: %q", got["empty"])
	}
	// Input must not be mutated.
	if in["api_token"] != "sk-secret" {
		t.Errorf("MaskLabels mutated input")
	}
	// Off is a pass-through.
	if off := MaskLabels(in, MaskOff); off["api_token"] != "sk-secret" {
		t.Errorf("MaskLabels(off) masked: %q", off["api_token"])
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
