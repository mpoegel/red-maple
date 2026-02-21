package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	redmaple "github.com/mpoegel/red-maple/pkg/redmaple"
	s3 "github.com/mpoegel/red-maple/pkg/s3"
)

func main() {
	if run() != nil {
		os.Exit(1)
	}
}

func run() error {
	duration := flag.Duration("duration", 24*time.Hour, "duration to query (e.g., 24h, 6h, 7d)")
	table := flag.String("table", "", "table to query (required)")
	verbose := flag.Bool("verbose", false, "enable debug logging")

	flag.Parse()

	if *verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	if *table == "" {
		slog.Error("--table is required")
		return fmt.Errorf("--table is required")
	}

	cfg := redmaple.LoadConfig()

	if !cfg.S3.Enabled {
		slog.Error("S3 is not enabled. Set S3_ENABLED=true and configure S3_ENDPOINT, S3_BUCKET, S3_ACCESS_KEY, S3_SECRET_KEY")
		return fmt.Errorf("S3 is not enabled")
	}

	s3Client, err := s3.NewClient(
		s3.WithBucket(cfg.S3.Bucket),
		s3.WithCredentials(cfg.S3.AccessKey, cfg.S3.SecretKey),
		s3.WithEndpoint(cfg.S3.Endpoint),
		s3.WithScheme(cfg.S3.Scheme),
		s3.WithFlushInterval(cfg.S3.FlushInterval),
		s3.WithRegion(cfg.S3.Region),
		s3.WithRetentionDays(cfg.S3.RetentionDays),
	)
	if err != nil {
		slog.Error("failed to create S3 client", "err", err)
		return err
	}
	defer s3Client.Close()

	slog.Info("querying S3", "table", *table, "duration", *duration)

	results, err := s3Client.QueryRange(context.Background(), *table, *duration)
	if err != nil {
		slog.Error("failed to query S3", "err", err)
		return err
	}

	slog.Info("retrieved data points", "count", len(results))

	encoder := json.NewEncoder(os.Stdout)
	for _, result := range results {
		if err := encoder.Encode(result); err != nil {
			slog.Warn("failed to encode result", "err", err)
			continue
		}
	}

	return nil
}
