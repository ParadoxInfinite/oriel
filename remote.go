package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// The `remote` subcommand manages the host allow-list of a *running* Oriel over
// loopback (127.0.0.1, which the API guard always trusts). The server persists
// the list and hot-reloads its guard, so changes take effect with no restart —
// and, run on the box itself, this is the way out of the bootstrap deadlock
// where you can't reach Settings → Remote access because the proxy host is 403'd.

type remoteBody struct {
	Hosts []string `json:"hosts"`
}

var remoteClient = &http.Client{Timeout: 5 * time.Second}

func runRemote(args []string) error {
	if len(args) == 0 {
		return remoteUsage()
	}
	sub := args[0]
	fs := flag.NewFlagSet("remote", flag.ContinueOnError)
	port := fs.Int("port", 4321, "port the running Oriel instance listens on")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	hosts := fs.Args()

	switch sub {
	case "list":
		cur, err := remoteGet(*port)
		if err != nil {
			return err
		}
		printHosts(cur)
		return nil
	case "allow", "deny":
		if len(hosts) == 0 {
			return fmt.Errorf("usage: oriel remote %s <host>...", sub)
		}
		cur, err := remoteGet(*port)
		if err != nil {
			return err
		}
		updated, err := remotePut(*port, applyHosts(cur, hosts, sub == "allow"))
		if err != nil {
			return err
		}
		printHosts(updated)
		fmt.Println("Applied to the running instance — no restart needed.")
		return nil
	default:
		return remoteUsage()
	}
}

func remoteUsage() error {
	fmt.Println(`Usage: oriel remote <command> [--port N] [host...]

Manage the hosts allowed to reach /api over the network. Changes apply to the
running instance immediately (no restart) and persist across restarts.

Commands:
  list                 show the currently allowed hosts
  allow <host>...      allow one or more hostnames, e.g. oriel.example.com
  deny  <host>...      remove one or more hostnames

  --port N   port the running Oriel listens on (default 4321)

Oriel has no authentication — only allow hosts on a network you trust.`)
	return nil
}

// applyHosts adds or removes hosts (lowercased, trimmed, deduped) from cur.
func applyHosts(cur, change []string, add bool) []string {
	set := map[string]bool{}
	for _, h := range cur {
		if n := normCLIHost(h); n != "" {
			set[n] = true
		}
	}
	for _, h := range change {
		n := normCLIHost(h)
		if n == "" {
			continue
		}
		if add {
			set[n] = true
		} else {
			delete(set, n)
		}
	}
	out := make([]string, 0, len(set))
	for h := range set {
		out = append(out, h)
	}
	return out
}

func normCLIHost(h string) string { return strings.ToLower(strings.TrimSpace(h)) }

func printHosts(hosts []string) {
	if len(hosts) == 0 {
		fmt.Println("Allowed hosts: (none — loopback only)")
		return
	}
	fmt.Println("Allowed hosts:")
	for _, h := range hosts {
		fmt.Printf("  - %s\n", h)
	}
}

func remoteURL(port int) string { return fmt.Sprintf("http://127.0.0.1:%d/api/remote", port) }

func remoteGet(port int) ([]string, error) {
	resp, err := remoteClient.Get(remoteURL(port))
	if err != nil {
		return nil, notRunning(port, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response from Oriel: %s", resp.Status)
	}
	var b remoteBody
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		return nil, err
	}
	return b.Hosts, nil
}

func remotePut(port int, hosts []string) ([]string, error) {
	payload, _ := json.Marshal(remoteBody{Hosts: hosts})
	req, err := http.NewRequest(http.MethodPut, remoteURL(port), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := remoteClient.Do(req)
	if err != nil {
		return nil, notRunning(port, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response from Oriel: %s", resp.Status)
	}
	var b remoteBody
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		return nil, err
	}
	return b.Hosts, nil
}

func notRunning(port int, err error) error {
	return fmt.Errorf("could not reach Oriel on 127.0.0.1:%d (%v)\n"+
		"Is it running? Check `oriel service status`, or pass --port if it listens elsewhere", port, err)
}
