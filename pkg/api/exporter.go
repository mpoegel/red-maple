package api

import (
	"context"
	"time"
)

type DataTag string

const (
	LocationTag DataTag = "location"
)

type DataPoint struct {
	Table  string
	Tags   map[DataTag]string
	Fields map[string]any
	Stamp  time.Time
}

type DataExporter interface {
	Export(ctx context.Context, dataPoints []*DataPoint) error
}

type ProviderFunc func(ctx context.Context) (*DataPoint, error)
