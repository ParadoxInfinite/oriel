package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/grant"
)

// The `ai` subcommand manages the destructive-grant window directly on disk, so
// it works whether or not the server is running and is seen by every Oriel
// process (server, `oriel mcp`). Destructive tools stay locked for MCP / the
// assistant until a window is open.
func runAI(args []string) error {
	if len(args) == 0 {
		return aiUsage()
	}
	store := grant.New()

	switch args[0] {
	case "allow-destructive":
		fs := flag.NewFlagSet("allow-destructive", flag.ContinueOnError)
		forStr := fs.String("for", "6h", "window length, e.g. 6h, 90m, 6d")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		d, err := parseFor(*forStr)
		if err != nil {
			return err
		}
		exp, err := store.Open(d)
		if err != nil {
			return err
		}
		fmt.Printf("Destructive actions unlocked for %s, until %s.\n", *forStr, exp.Local().Format(time.RFC1123))
		fmt.Println("Lock early with: oriel ai lock")
		return nil

	case "status":
		active, exp := store.Status()
		if !active {
			fmt.Println("Destructive actions: LOCKED (no open window).")
			return nil
		}
		fmt.Printf("Destructive actions: UNLOCKED until %s (%s left).\n",
			exp.Local().Format(time.RFC1123), time.Until(exp).Round(time.Minute))
		return nil

	case "lock":
		if err := store.Lock(); err != nil {
			return err
		}
		fmt.Println("Destructive actions locked.")
		return nil

	default:
		return aiUsage()
	}
}

// parseFor accepts Go durations (6h, 90m) plus a trailing-d days form (6d),
// which time.ParseDuration doesn't handle.
func parseFor(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		days, err := strconv.ParseFloat(strings.TrimSuffix(s, "d"), 64)
		if err != nil || days <= 0 || days > maxGrantDays {
			return 0, fmt.Errorf("invalid window %q (1m to %dd)", s, maxGrantDays)
		}
		return time.Duration(days * 24 * float64(time.Hour)), nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 || d > maxGrantDays*24*time.Hour {
		return 0, fmt.Errorf("invalid window %q (try 6h, 90m, or 6d; max %dd)", s, maxGrantDays)
	}
	return d, nil
}

// maxGrantDays caps the destructive window. Long enough for a trusted box, short
// enough that "forever" is a deliberate, repeated choice.
const maxGrantDays = 30

func aiUsage() error {
	fmt.Println("usage: oriel ai <command>")
	fmt.Println("  allow-destructive [--for 6h]  open the destructive-grant window (MCP / assistant)")
	fmt.Println("  status                        show whether destructive actions are unlocked")
	fmt.Println("  lock                          close the window now")
	return nil
}
