package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
)

// The `config` subcommand edits settings.json config that can't hot-reload, via
// the running instance over loopback (so the server persists it in its own
// context). Today that's the reverse-proxy base path; setting it restarts a
// managed service to apply, since the base is baked into the served assets.

func runConfig(args []string) error {
	if len(args) == 0 {
		return configUsage()
	}
	switch args[0] {
	case "base-path":
		return runConfigBasePath(args[1:])
	case "auth-token":
		return runConfigAuthToken(args[1:])
	default:
		return configUsage()
	}
}

func configUsage() error {
	fmt.Println(`Usage: oriel config <command>

  base-path [<path>] [--clear] [--port N]
        Show, set, or clear the reverse-proxy sub-path (e.g. /oriel).
        Setting it restarts a managed service to apply.

  auth-token [--generate | --clear | <token>] [--port N]
        Show, generate, set, or clear the bearer token that gates non-loopback
        /api access (MCP-over-HTTP, reverse-proxied access). Off by default;
        loopback (the local UI) never needs it.`)
	return nil
}

func runConfigBasePath(args []string) error {
	fs := flag.NewFlagSet("config base-path", flag.ContinueOnError)
	port := fs.Int("port", 4321, "port the running Oriel instance listens on")
	clear := fs.Bool("clear", false, "serve at the host root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()

	// No value and not --clear → show the current base path.
	if len(rest) == 0 && !*clear {
		self, err := fetchSelf(*port)
		if err != nil {
			return err
		}
		if self.BasePath == "" || self.BasePath == "/" {
			fmt.Println("base path: (host root)")
		} else {
			fmt.Printf("base path: %s\n", self.BasePath)
		}
		return nil
	}

	val := ""
	if !*clear {
		val = rest[0]
	}
	return setBasePath(*port, val)
}

func setBasePath(port int, val string) error {
	payload, _ := json.Marshal(map[string]string{"basePath": val})
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://127.0.0.1:%d/api/config", port), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := remoteClient.Do(req)
	if err != nil {
		return notRunning(port, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response from Oriel: %s", resp.Status)
	}
	var out struct {
		BasePath   string `json:"basePath"`
		Restarting bool   `json:"restarting"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if out.BasePath == "/" {
		fmt.Println("base path cleared, serving at the host root")
	} else {
		fmt.Printf("base path set to %s\n", out.BasePath)
	}
	if out.Restarting {
		fmt.Println("restarting Oriel to apply…")
	} else {
		fmt.Println("restart Oriel to apply (not a managed service).")
	}
	return nil
}

func runConfigAuthToken(args []string) error {
	fs := flag.NewFlagSet("config auth-token", flag.ContinueOnError)
	port := fs.Int("port", 4321, "port the running Oriel instance listens on")
	generate := fs.Bool("generate", false, "generate a random token and print it once")
	clear := fs.Bool("clear", false, "disable the token gate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()

	// No flags/value → show whether the gate is on.
	if !*generate && !*clear && len(rest) == 0 {
		enabled, err := fetchAuth(*port)
		if err != nil {
			return err
		}
		if enabled {
			fmt.Println("auth: ON, non-loopback /api requires a bearer token")
		} else {
			fmt.Println("auth: off, loopback only")
		}
		return nil
	}

	body := map[string]any{}
	switch {
	case *clear:
		body["clear"] = true
	case *generate:
		body["generate"] = true
	default:
		body["token"] = rest[0]
	}
	return setAuth(*port, body)
}

func fetchAuth(port int) (bool, error) {
	resp, err := remoteClient.Get(fmt.Sprintf("http://127.0.0.1:%d/api/auth", port))
	if err != nil {
		return false, notRunning(port, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected response from Oriel: %s", resp.Status)
	}
	var out struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, err
	}
	return out.Enabled, nil
}

func setAuth(port int, body map[string]any) error {
	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://127.0.0.1:%d/api/auth", port), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := remoteClient.Do(req)
	if err != nil {
		return notRunning(port, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response from Oriel: %s", resp.Status)
	}
	var out struct {
		Enabled bool   `json:"enabled"`
		Token   string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	switch {
	case out.Token != "":
		fmt.Println("auth token generated, set it as a bearer token on your MCP/HTTP client:")
		fmt.Printf("\n  %s\n\n", out.Token)
		fmt.Println("Shown once, store it now. Non-loopback /api now requires it.")
	case out.Enabled:
		fmt.Println("auth token set, non-loopback /api now requires it.")
	default:
		fmt.Println("auth disabled, loopback only.")
	}
	return nil
}
