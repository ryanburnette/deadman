package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ryanburnette/deadman/internal/handlers"
	"github.com/ryanburnette/deadman/internal/mailer"
	"github.com/ryanburnette/deadman/internal/monitor"
)

func serveCmd(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", getEnv("ADDR", ":3080"), "listen address")
	configPath := fs.String("config", getEnv("CONFIG_PATH", defaultConfigPath()), "services config path")
	statePath := fs.String("state", getEnv("STATE_PATH", defaultStatePath()), "state file path")
	checkInterval := fs.Duration("check-interval", 15*time.Second, "how often to check for missed heartbeats and config changes")

	for _, a := range args {
		if a == "-h" || a == "-help" || a == "--help" {
			fmt.Println("Usage: deadman serve [options]")
			fmt.Println()
			fmt.Println("Options:")
			fs.SetOutput(os.Stdout)
			fs.PrintDefaults()
			os.Exit(0)
		}
	}
	fs.Parse(args)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(slog.LevelError) // log.Fatal et al. show as ERROR, not INFO

	smtpCfg, err := mailer.ConfigFromEnv()
	if err != nil {
		log.Fatal("smtp config: ", err)
	}
	m, err := monitor.New(*configPath, *statePath, mailer.New(smtpCfg))
	if err != nil {
		log.Fatal("starting monitor: ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go m.Run(ctx, *checkInterval)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	handlers.Register(r, m)

	srv := &http.Server{
		Addr:    *addr,
		Handler: r,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-done
		slog.Info("shutting down")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown error", "error", err)
		}
	}()

	slog.Info("starting server", "addr", *addr, "config", *configPath, "state", *statePath)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("server error: ", err)
	}
}
