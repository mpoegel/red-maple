package redmaple

import (
	"context"
	"log/slog"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
)

type ExportHub struct {
	interval  time.Duration
	exporters []api.DataExporter
	providers []api.ProviderFunc
}

func NewExportHub(interval time.Duration) *ExportHub {
	return &ExportHub{
		interval:  interval,
		exporters: []api.DataExporter{},
		providers: []api.ProviderFunc{},
	}
}

func (e *ExportHub) AddExporter(exporter api.DataExporter) {
	e.exporters = append(e.exporters, exporter)
}

func (e *ExportHub) AddProvider(provider api.ProviderFunc) {
	e.providers = append(e.providers, provider)
}

func (e *ExportHub) Run(ctx context.Context) {
	timer := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			points := []*api.DataPoint{}
			for _, provider := range e.providers {
				data, err := provider(ctx)
				if err != nil {
					slog.Warn("data provider failed", "err", err)
					continue
				}
				points = append(points, data)
			}
			for _, exporter := range e.exporters {
				if err := exporter.Export(ctx, points); err != nil {
					slog.Warn("data export failed", "err", err)
				}
			}
			timer = time.NewTimer(e.interval)
		}
	}
}
