package main

import (
	"context"
	"github.com/example/digitalocean-ddns/internal/config"
	"github.com/example/digitalocean-ddns/internal/digitalocean"
	"github.com/example/digitalocean-ddns/internal/observability"
	"github.com/example/digitalocean-ddns/internal/publicip"
	"github.com/example/digitalocean-ddns/internal/updater"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var version = "dev"

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		log.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	httpClient := &http.Client{Timeout: config.LookupTimeout()}
	metrics := &observability.Metrics{}
	u := updater.Updater{IP: publicip.Client{HTTP: httpClient, Providers: cfg.Providers}, DNS: digitalocean.Client{HTTP: httpClient, BaseURL: cfg.DOAPIBaseURL, Token: cfg.Token}, Records: cfg.Records, Metrics: metrics, Log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !metrics.IsReady() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/metrics", metrics)
	server := &http.Server{Addr: cfg.ListenAddress, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Info("observability server starting", "address", cfg.ListenAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", "error", err)
			stop()
		}
	}()
	log.Info("DDNS updater starting", "version", version, "records", len(cfg.Records), "interval", cfg.Interval.String())
	u.Run(ctx, cfg.Interval)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP shutdown failed", "error", err)
	}
	log.Info("shutdown complete")
}
