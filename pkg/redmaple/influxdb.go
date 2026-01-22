package redmaple

import (
	"context"

	influxdb3 "github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	lineprotocol "github.com/influxdata/line-protocol/v2/lineprotocol"
	api "github.com/mpoegel/red-maple/pkg/api"
)

type InfluxDBExporter struct {
	client *influxdb3.Client
}

var _ api.DataExporter = (*InfluxDBExporter)(nil)

func NewInfluxDBExporter(cfg *InfluxDBConfig) (*InfluxDBExporter, error) {
	client, err := influxdb3.New(influxdb3.ClientConfig{
		Host:     cfg.Endpoint,
		Token:    cfg.Token,
		Database: cfg.Database,
	})
	if err != nil {
		return nil, err
	}
	return &InfluxDBExporter{
		client: client,
	}, nil
}

func (e *InfluxDBExporter) Close() error {
	return e.client.Close()
}

func (e *InfluxDBExporter) Export(ctx context.Context, dataPoints []*api.DataPoint) error {
	if len(dataPoints) == 0 {
		return nil
	}
	points := make([]*influxdb3.Point, len(dataPoints))
	for i, data := range dataPoints {
		tags := map[string]string{}
		for k, v := range data.Tags {
			tags[string(k)] = v
		}
		point := influxdb3.NewPoint(data.Table, tags, data.Fields, data.Stamp)
		points[i] = point
	}
	return e.client.WritePoints(ctx,
		points,
		influxdb3.WithPrecision(lineprotocol.Second))
}
