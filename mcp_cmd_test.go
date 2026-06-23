package main

import (
	"testing"

	"github.com/ParadoxInfinite/oriel/internal/tools"
)

func TestExposedAddr(t *testing.T) {
	for addr, exposed := range map[string]bool{
		"127.0.0.1:8080":   false,
		"localhost:8080":   false,
		"[::1]:8080":       false,
		":8080":            true,
		"0.0.0.0:8080":     true,
		"192.168.1.5:8080": true,
		"example.com:8080": true,
	} {
		if got := exposedAddr(addr); got != exposed {
			t.Errorf("exposedAddr(%q) = %v, want %v", addr, got, exposed)
		}
	}
}

func TestToolFilter(t *testing.T) {
	read := &tools.Tool{Name: "container.list", ReadOnly: true}
	mut := &tools.Tool{Name: "container.stop"}
	dgr := &tools.Tool{Name: "container.remove", Destructive: true}

	cases := []struct {
		name              string
		filter            func(*tools.Tool) bool
		read, mut, danger bool // expected admission
	}{
		{"no flags admits all", toolFilter(false, "", ""), true, true, true},
		{"read-only keeps only reads", toolFilter(true, "", ""), true, false, false},
		{"allow-list is exclusive", toolFilter(false, "container.stop", ""), false, true, false},
		{"deny-list removes named", toolFilter(false, "", "container.remove"), true, true, false},
		{"read-only + allow intersect", toolFilter(true, "container.list,container.stop", ""), true, false, false},
	}
	for _, c := range cases {
		if got := c.filter(read); got != c.read {
			t.Errorf("%s: read admitted=%v, want %v", c.name, got, c.read)
		}
		if got := c.filter(mut); got != c.mut {
			t.Errorf("%s: mutation admitted=%v, want %v", c.name, got, c.mut)
		}
		if got := c.filter(dgr); got != c.danger {
			t.Errorf("%s: destructive admitted=%v, want %v", c.name, got, c.danger)
		}
	}
}
