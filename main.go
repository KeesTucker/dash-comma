// dash-comma is a server dashboard: dashdot's API and config, with its own
// stats collection and UI.
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/KeesTucker/dash-comma/internal/config"
	"github.com/KeesTucker/dash-comma/internal/server"
	"github.com/KeesTucker/dash-comma/internal/stats"
	"github.com/KeesTucker/dash-comma/web"
)

func main() {
	cfg := config.Load()
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck(cfg))
	}
	if cfg.RunningInDocker {
		stats.HostRoot = "/mnt/host"
	}

	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		log.Fatal(err)
	}
	srv := server.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := fmt.Sprintf(":%d", int(cfg.Port))
	// No read or write timeout: they would cut off websockets and polling.
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(dist),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    64 << 10,
	}
	go func() {
		log.Printf("listening on *:%d", int(cfg.Port))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	go srv.RunSpeedTests(ctx)

	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdown)
}

// healthcheck asks the running server for its UI, for Docker's HEALTHCHECK.
func healthcheck(cfg *config.Config) int {
	path := "/" + strings.Trim(cfg.RoutingPath, "/")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d%s", int(cfg.Port), path), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, resp.Status)
		return 1
	}
	return 0
}
