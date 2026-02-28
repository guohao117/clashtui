//go:build ignore

// Debug is a standalone tool for testing Clash log streaming.
// Run with: go run cmd/debug.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/guohao117/clashtui/internal/api"
	"github.com/guohao117/clashtui/internal/config"
)

func main() {
	fmt.Println("Starting debug log reader...")

	cfg := config.LoadFromEnv()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	client := api.NewClient(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println("\nShutting down...")
		cancel()
	}()

	ch := make(chan []api.ClashLog, 16)
	go func() {
		if err := client.StreamLogs(ctx, ch); err != nil && ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "stream error: %v\n", err)
		}
		close(ch)
	}()

	count := 0
	for batch := range ch {
		for _, log := range batch {
			count++
			fmt.Printf("[%d] %-7s %s  %s\n", count, log.Type, log.Time, log.Payload)
		}
	}

	fmt.Printf("Total logs: %d\n", count)
}
