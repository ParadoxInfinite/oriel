package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/server"
	"github.com/ParadoxInfinite/oriel/internal/service"
)

// version is the build version. It stays "dev" for local builds and is set to
// the release tag via -ldflags "-X main.version=v1.2.3" in the release workflow.
var version = "dev"

func main() {
	// Subcommands that run and exit before the server starts.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "service":
			// `oriel service <install|uninstall|status>` manages the background service.
			if err := service.Run(os.Args[2:]); err != nil {
				log.Fatalf("service: %v", err)
			}
			return
		case "remote":
			// `oriel remote <list|allow|deny>` manages the running instance's host
			// allow-list over loopback.
			if err := runRemote(os.Args[2:]); err != nil {
				log.Fatalf("remote: %v", err)
			}
			return
		case "doctor":
			// `oriel doctor` reports config/connectivity health and exits.
			if err := runDoctor(os.Args[2:]); err != nil {
				log.Fatalf("doctor: %v", err)
			}
			return
		case "config":
			// `oriel config base-path …` edits settings.json config over loopback.
			if err := runConfig(os.Args[2:]); err != nil {
				log.Fatalf("config: %v", err)
			}
			return
		case "update":
			// `oriel update` drives the running instance's checksum-verified self-update.
			if err := runUpdate(os.Args[2:]); err != nil {
				log.Fatalf("update: %v", err)
			}
			return
		case "version", "--version", "-v":
			fmt.Println("oriel", version)
			return
		}
	}

	port := flag.Int("port", 4321, "port to listen on (bound to 127.0.0.1 only)")
	noOpen := flag.Bool("no-open", false, "do not open a browser window on start")
	flag.Parse()

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	srv := server.New(webFS(), version)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown on Ctrl-C / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		url := fmt.Sprintf("http://%s", addr)
		log.Printf("oriel listening on %s", url)
		if !*noOpen {
			// Give the listener a beat to come up before opening the browser.
			time.Sleep(300 * time.Millisecond)
			openBrowser(url)
		}
	}()

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	// Short grace period: SSE streams never close on their own, so a long
	// timeout just stalls every restart. 2s is plenty for real in-flight calls.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	// Persist the metrics history so a restart resumes where it left off.
	srv.Close()
}

// openBrowser opens the default browser at url. Best-effort; errors are ignored.
func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	_ = exec.Command(cmd, args...).Start()
}
