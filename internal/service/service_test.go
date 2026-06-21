package service

import "testing"

func TestNormalizeManagedExe(t *testing.T) {
	// Use a path guaranteed not to exist so EvalSymlinks is a no-op and only the
	// suffix normalization is exercised.
	base := "/nonexistent-oriel-test/bin/oriel"
	cases := map[string]string{
		base:                base,
		base + ".bak":       base,
		base + " (deleted)": base,
	}
	for in, want := range cases {
		if got := normalizeManagedExe(in); got != want {
			t.Errorf("normalizeManagedExe(%q) = %q, want %q", in, got, want)
		}
	}
}
