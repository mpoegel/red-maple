package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	redmaple "github.com/mpoegel/red-maple/pkg/redmaple"
)

func main() {
	if run() != nil {
		os.Exit(1)
	}
}

func run() error {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	config := redmaple.LoadConfig()

	server, err := redmaple.NewServer(config)
	if err != nil {
		slog.Error("could not create server", "error", err)
		return err
	}

	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		server.Stop(ctx)
	}()

	return server.Start(ctx)
}
