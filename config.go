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
	default:
		return configUsage()
	}
}

func configUsage() error {
	fmt.Println(`Usage: oriel config <command>

  base-path [<path>] [--clear] [--port N]
        Show, set, or clear the reverse-proxy sub-path (e.g. /oriel).
        Setting it restarts a managed service to apply.`)
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
		fmt.Println("base path cleared — serving at the host root")
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
