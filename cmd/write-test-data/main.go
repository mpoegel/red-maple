package main

// This tool generates and writes historical test data to S3 for testing purposes.
// Currently generates random Citibike station data with classics and ebikes counts.

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
	redmaple "github.com/mpoegel/red-maple/pkg/redmaple"
	s3 "github.com/mpoegel/red-maple/pkg/s3"
)

type WriteOpts struct {
	Days         int
	Stations     []string
	Interval     time.Duration
	StartTime    time.Time
	BaseClassics int
	BaseEbikes   int
}

type DataWriter interface {
	GenerateHistoricalData(ctx context.Context, opts WriteOpts) ([]*api.DataPoint, error)
	Name() string
}

type CitibikeWriter struct{}

func (w *CitibikeWriter) Name() string {
	return "citibike"
}

func (w *CitibikeWriter) GenerateHistoricalData(ctx context.Context, opts WriteOpts) ([]*api.DataPoint, error) {
	var points []*api.DataPoint

	for _, station := range opts.Stations {
		classics := min(50, max(0, opts.BaseClassics))
		ebikes := min(50, max(0, opts.BaseEbikes))

		startTime := opts.StartTime
		if startTime.IsZero() {
			startTime = time.Now()
		}

		endTime := startTime.Add(-time.Duration(opts.Days) * 24 * time.Hour)

		for t := startTime; t.After(endTime); t = t.Add(-opts.Interval) {
			delta := rand.Intn(6) - 3
			classics += delta
			classics = min(50, max(0, classics))

			delta = rand.Intn(6) - 3
			ebikes += delta
			ebikes = min(50, max(0, ebikes))

			point := &api.DataPoint{
				Table: "citibike",
				Tags: map[api.DataTag]string{
					api.LocationTag: station,
				},
				Fields: map[string]any{
					"classics": classics,
					"ebikes":   ebikes,
				},
				Stamp: t,
			}
			points = append(points, point)
		}
	}

	return points, nil
}

var writers = map[string]DataWriter{
	"citibike": &CitibikeWriter{},
}

func main() {
	if run() != nil {
		os.Exit(1)
	}
}

func run() error {
	days := flag.Int("days", 7, "number of days of history to write (1-30)")
	stations := flag.String("stations", "", "comma-separated citibike stations (default: from config)")
	interval := flag.Int("interval", 5, "minutes between readings")
	baseClassics := flag.Int("base-classics", 25, "starting classics count (0-50)")
	baseEbikes := flag.Int("base-ebikes", 25, "starting ebikes count (0-50)")
	dryRun := flag.Bool("dry-run", false, "print what would be written without writing")
	verbose := flag.Bool("verbose", false, "enable debug logging")

	flag.Parse()

	if *verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	if *days < 1 || *days > 30 {
		slog.Error("days must be between 1 and 30")
		return fmt.Errorf("days must be between 1 and 30")
	}

	cfg := redmaple.LoadConfig()

	if !cfg.S3.Enabled {
		slog.Error("S3 is not enabled. Set S3_ENABLED=true and configure S3_ENDPOINT, S3_BUCKET, S3_ACCESS_KEY, S3_SECRET_KEY")
		return fmt.Errorf("S3 is not enabled")
	}

	stationList := *stations
	if stationList == "" {
		if len(cfg.CitibikeStations) > 0 {
			stationList = cfg.CitibikeStations[0]
		} else {
			stationList = "Test Station"
		}
	}

	stationsSlice := parseStations(stationList)

	opts := WriteOpts{
		Days:         *days,
		Stations:     stationsSlice,
		Interval:     time.Duration(*interval) * time.Minute,
		BaseClassics: *baseClassics,
		BaseEbikes:   *baseEbikes,
	}

	slog.Info("generating test data",
		"days", opts.Days,
		"stations", opts.Stations,
		"interval", opts.Interval,
		"baseClassics", opts.BaseClassics,
		"baseEbikes", opts.BaseEbikes)

	writer := writers["citibike"]
	points, err := writer.GenerateHistoricalData(context.Background(), opts)
	if err != nil {
		slog.Error("failed to generate data", "err", err)
		return err
	}

	slog.Info("generated points", "count", len(points))

	if *dryRun {
		slog.Info("dry-run: would write points", "count", len(points))
		for i, p := range points {
			if i < 10 {
				slog.Info("dry-run point",
					"table", p.Table,
					"location", p.Tags[api.LocationTag],
					"classics", p.Fields["classics"],
					"ebikes", p.Fields["ebikes"],
					"time", p.Stamp)
			}
		}
		if len(points) > 10 {
			slog.Info("dry-run: ... and more points", "remaining", len(points)-10)
		}
		return nil
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

	slog.Info("writing to S3", "bucket", cfg.S3.Bucket)

	const batchSize = 1000
	for i := 0; i < len(points); i += batchSize {
		end := i + batchSize
		if end > len(points) {
			end = len(points)
		}
		batch := points[i:end]
		for _, p := range batch {
			slog.Debug("exporting data point",
				"table", p.Table,
				"location", p.Tags[api.LocationTag],
				"classics", p.Fields["classics"],
				"ebikes", p.Fields["ebikes"],
				"time", p.Stamp)
		}
		if err := s3Client.Export(context.Background(), batch); err != nil {
			slog.Error("failed to write batch", "err", err, "batchSize", len(batch))
			return err
		}
		slog.Info("wrote batch", "batch", i/batchSize+1, "size", len(batch))
	}

	slog.Info("successfully wrote test data", "totalPoints", len(points))
	return nil
}

func parseStations(s string) []string {
	if s == "" {
		return []string{}
	}
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
