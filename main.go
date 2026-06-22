package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
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
			runSub("service", service.Run) // install | uninstall | status
		case "remote":
			runSub("remote", runRemote) // list | allow | deny — host allow-list over loopback
		case "doctor":
			runSub("doctor", runDoctor) // config/connectivity health check
		case "config":
			runSub("config", runConfig) // edit settings.json over loopback
		case "update":
			runSub("update", runUpdate) // checksum-verified self-update
		case "ai":
			runSub("ai", runAI) // allow-destructive | status | lock — the grant window
		case "mcp":
			runSub("mcp", runMCP) // serve the tool registry to an MCP client over stdio
		case "version", "--version", "-v":
			fmt.Println("oriel", version)
			return
		default:
			// A non-flag first arg is a mistyped subcommand; flags fall through to
			// the server below.
			if !strings.HasPrefix(os.Args[1], "-") {
				fmt.Fprintf(os.Stderr, "oriel: unknown command %q\nrun one of: service, remote, doctor, config, update, ai, mcp, version\n", os.Args[1])
				os.Exit(2)
			}
		}
	}

	port := flag.Int("port", 4321, "port to listen on (bound to 127.0.0.1 only)")
	noOpen := flag.Bool("no-open", false, "do not open a browser window on start")
	flag.Parse()

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	srv := server.New(webFS(), version)

	httpServer := &http.Server{
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Bind synchronously so a port-in-use error is reported up front, before we
	// open a browser to a dead port or detach the serve loop.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("cannot listen on %s: %v", addr, err)
	}

	// Graceful shutdown on Ctrl-C / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	url := fmt.Sprintf("http://%s", addr)
	srv.LogStartup(url)
	if !*noOpen {
		go openBrowser(url) // the listener is already up
	}

	serveErr := make(chan error, 1)
	go func() {
		if err := httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("shutting down...")
	case err := <-serveErr:
		log.Printf("server error: %v", err)
	}
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

// runSub runs a subcommand handler and exits. A -h request (flag.ErrHelp) is a
// success — the flag package already printed usage — so it exits 0 cleanly
// rather than re-printing the error and exiting non-zero.
func runSub(name string, fn func([]string) error) {
	if err := fn(os.Args[2:]); err != nil && !errors.Is(err, flag.ErrHelp) {
		log.Fatalf("%s: %v", name, err)
	}
	os.Exit(0)
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
