package settings

import "testing"

func TestBearer(t *testing.T) {
	cases := map[string]string{
		"Bearer abc":   "abc",
		"bearer abc":   "abc", // scheme is case-insensitive
		"Bearer  abc ": "abc", // trimmed
		"Basic abc":    "",
		"abc":          "",
		"":             "",
		"Bearer":       "",
	}
	for in, want := range cases {
		if got := Bearer(in); got != want {
			t.Errorf("Bearer(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTokenOK(t *testing.T) {
	for _, c := range []struct {
		name, provided, configured string
		want                       bool
	}{
		{"auth off admits anything", "", "", true},
		{"auth off admits a stray token", "whatever", "", true},
		{"correct", "secret", "secret", true},
		{"wrong", "nope", "secret", false},
		{"missing", "", "secret", false},
		{"prefix is not a match", "secre", "secret", false},
	} {
		if got := TokenOK(c.provided, c.configured); got != c.want {
			t.Errorf("%s: TokenOK(%q,%q)=%v want %v", c.name, c.provided, c.configured, got, c.want)
		}
	}
}
