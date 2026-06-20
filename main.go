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

func main() {
	// `oriel service <install|uninstall|status>` manages the background
	// service and exits; everything else runs the server.
	if len(os.Args) > 1 && os.Args[1] == "service" {
		if err := service.Run(os.Args[2:]); err != nil {
			log.Fatalf("service: %v", err)
		}
		return
	}

	port := flag.Int("port", 4321, "port to listen on (bound to 127.0.0.1 only)")
	noOpen := flag.Bool("no-open", false, "do not open a browser window on start")
	flag.Parse()

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	srv := server.New(webFS())

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
