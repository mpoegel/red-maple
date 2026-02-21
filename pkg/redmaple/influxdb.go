package redmaple

import (
	"context"
	"fmt"
	"time"

	influxdb3 "github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	lineprotocol "github.com/influxdata/line-protocol/v2/lineprotocol"
	api "github.com/mpoegel/red-maple/pkg/api"
)

type InfluxDBClient struct {
	client *influxdb3.Client
}

var _ api.DataExporter = (*InfluxDBClient)(nil)
var _ api.Importer = (*InfluxDBClient)(nil)

func NewInfluxDBClient(cfg *InfluxDBConfig) (*InfluxDBClient, error) {
	client, err := influxdb3.New(influxdb3.ClientConfig{
		Host:     cfg.Endpoint,
		Token:    cfg.Token,
		Database: cfg.Database,
	})
	if err != nil {
		return nil, err
	}
	return &InfluxDBClient{
		client: client,
	}, nil
}

func (e *InfluxDBClient) Close() error {
	return e.client.Close()
}

func (e *InfluxDBClient) Export(ctx context.Context, dataPoints []*api.DataPoint) error {
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

func (e *InfluxDBClient) QueryRange(ctx context.Context, table string, duration time.Duration) ([]*api.DataPoint, error) {
	d := fmt.Sprintf("%.0f seconds", duration.Seconds())
	query := "SELECT * FROM " + table + " WHERE time >= now() - interval '" + d + "'"
	iterator, err := e.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	var results []*api.DataPoint
	for iterator.Next() {
		row := iterator.Value()

		point := &api.DataPoint{
			Table:  table,
			Tags:   make(map[api.DataTag]string),
			Fields: make(map[string]any),
		}

		if v, ok := row["time"]; ok {
			if t, ok := v.(time.Time); ok {
				point.Stamp = t
			}
		}

		for k, v := range row {
			if k == "time" {
				continue
			}
			point.Fields[k] = v
		}

		results = append(results, point)
	}

	return results, iterator.Err()
}

func (e *InfluxDBClient) queryTable(ctx context.Context, table, duration string) ([]map[string]any, error) {
	query := "SELECT * FROM " + table + " WHERE time >= now() - interval '" + duration + "'"
	iterator, err := e.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	for iterator.Next() {
		row := iterator.Value()
		results = append(results, row)
	}
	return results, iterator.Err()
}
