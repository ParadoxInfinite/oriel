package server

import "testing"

func TestCompareSemverPrerelease(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.9.0", "0.8.0", 1},           // higher core
		{"0.8.0", "0.9.0", -1},          //
		{"0.9.0", "0.9.0", 0},           // equal
		{"0.9.0-rc.1", "0.8.0", 1},      // pre-release of a higher core still wins
		{"0.9.0-rc.1", "0.9.0", -1},     // a pre-release ranks below its final release
		{"0.9.0", "0.9.0-rc.1", 1},      //
		{"0.9.0-rc.2", "0.9.0-rc.1", 1}, // numeric identifier ordering
		{"0.9.0-rc.1", "0.9.0-rc.1", 0},
		{"1.0.0-alpha", "1.0.0-alpha.1", -1}, // fewer identifiers ranks lower
		{"1.0.0-alpha.1", "1.0.0-beta", -1},  // alpha < beta lexically
		{"v0.9.0-rc.1", "0.9.0", -1},         // leading v tolerated
	}
	for _, c := range cases {
		if got := compareSemver(c.a, c.b); got != c.want {
			t.Errorf("compareSemver(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestIsNewerWithChannels(t *testing.T) {
	// Edge tester on stable, an rc appears → update offered.
	if !isNewer("0.8.0", "0.9.0-rc.1") {
		t.Error("an rc with a higher core should be newer")
	}
	// Edge tester on the rc, the final ships → move onto it.
	if !isNewer("0.9.0-rc.1", "0.9.0") {
		t.Error("the final release should be newer than its rc")
	}
	// Don't offer to 'downgrade' a final onto its own rc.
	if isNewer("0.9.0", "0.9.0-rc.1") {
		t.Error("an rc must not look newer than its final release")
	}
	if isNewer("0.9.0-rc.2", "0.9.0-rc.1") {
		t.Error("an older rc must not look newer")
	}
}

func TestNormChannel(t *testing.T) {
	for in, want := range map[string]string{
		"edge": "edge", "stable": "stable", "": "stable", "bogus": "stable", "EDGE": "stable",
	} {
		if got := normChannel(in); got != want {
			t.Errorf("normChannel(%q)=%q want %q", in, got, want)
		}
	}
}

func TestPrerelease(t *testing.T) {
	for in, want := range map[string]string{
		"0.9.0": "", "v0.9.0": "", "0.9.0-rc.1": "rc.1", "v0.9.0-rc.1+build.5": "rc.1", "0.9.0+meta": "",
	} {
		if got := prerelease(in); got != want {
			t.Errorf("prerelease(%q)=%q want %q", in, got, want)
		}
	}
}
