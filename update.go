package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"
)

// The `update` subcommand drives the running instance's self-update over
// loopback, check, download+verify, restart, so a headless box can upgrade
// from the terminal without the (possibly 403'd) UI. It reuses the same
// checksum-verified machinery as the in-app updater.

type updateStatus struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"` // without the leading "v"
	UpdateAvailable bool   `json:"updateAvailable"`
	Managed         bool   `json:"managed"`
	PackageManager  string `json:"packageManager"`
	Error           string `json:"error"`
}

type applyResult struct {
	Updated bool   `json:"updated"`
	Message string `json:"message"`
	Latest  string `json:"latest"`
}

// The check makes a GitHub call server-side (up to ~10s); apply downloads a
// ~12MB binary server-side before replying, so give it room.
var (
	updateCheckClient = &http.Client{Timeout: 30 * time.Second}
	updateApplyClient = &http.Client{Timeout: 6 * time.Minute}
)

func runUpdate(args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	port := fs.Int("port", 4321, "port the running Oriel instance listens on")
	checkOnly := fs.Bool("check", false, "only check for an update; don't install")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st, err := getUpdateStatus(*port)
	if err != nil {
		return err
	}
	if st.Error != "" {
		return fmt.Errorf("update check failed: %s", st.Error)
	}
	fmt.Printf("current: %s\n", st.Current)
	fmt.Printf("latest:  v%s\n", st.Latest)
	if !st.UpdateAvailable {
		fmt.Println("You're on the latest version.")
		return nil
	}
	if st.PackageManager == "homebrew" {
		fmt.Printf("v%s is available. Oriel was installed with Homebrew, update it with: brew upgrade oriel\n", st.Latest)
		return nil
	}
	if *checkOnly {
		fmt.Printf("Update available, run `oriel update` to install v%s.\n", st.Latest)
		return nil
	}
	if !st.Managed {
		return fmt.Errorf("self-update needs a service install, run `oriel service install` (or re-run install.sh)")
	}

	fmt.Printf("Updating to v%s, downloading and verifying…\n", st.Latest)
	res, err := postJSON[applyResult](updateApplyClient, *port, "/api/update/apply")
	if err != nil {
		return err
	}
	if !res.Updated {
		fmt.Println(res.Message)
		return nil
	}
	fmt.Println("verified. Restarting the service…")
	// Best-effort: the server restarts and drops this connection.
	_, _ = postJSON[map[string]any](updateCheckClient, *port, "/api/update/restart")
	if err := waitForVersion(*port, st.Current); err != nil {
		return err
	}
	// New checks ride along with a new binary. Surface a stale shell env now, in
	// the same breath as the upgrade, rather than waiting for the user to run
	// `oriel doctor` and wonder what changed.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if hint := dockerHostHint(ctx); hint != "" {
		fmt.Printf("\nHeads up: %s\n", hint)
	}
	return nil
}

func getUpdateStatus(port int) (updateStatus, error) {
	var st updateStatus
	resp, err := updateCheckClient.Get(fmt.Sprintf("http://127.0.0.1:%d/api/update?force=1", port))
	if err != nil {
		return st, notRunning(port, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return st, fmt.Errorf("unexpected response from Oriel: %s", resp.Status)
	}
	return st, json.NewDecoder(resp.Body).Decode(&st)
}

func postJSON[T any](client *http.Client, port int, path string) (T, error) {
	var out T
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil)
	if err != nil {
		return out, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return out, notRunning(port, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error != "" {
			return out, fmt.Errorf("%s", e.Error)
		}
		return out, fmt.Errorf("unexpected response from Oriel: %s", resp.Status)
	}
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

// waitForVersion polls /api/self until the running version differs from prev
// (the service has restarted onto the new binary).
func waitForVersion(port int, prev string) error {
	for i := 0; i < 40; i++ {
		time.Sleep(500 * time.Millisecond)
		if self, err := fetchSelf(port); err == nil && self.Version != "" && self.Version != prev {
			fmt.Printf("Now on %s.\n", self.Version)
			return nil
		}
	}
	fmt.Println("Update installed; the service is restarting.")
	return nil
}
