package redmaple

import (
	"context"

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

func (e *InfluxDBClient) QueryLast24Hours(ctx context.Context, table string) ([]map[string]any, error) {
	return e.queryTable(ctx, table, "24 hours")
}

func (e *InfluxDBClient) QueryLast7Days(ctx context.Context, table string) ([]map[string]any, error) {
	return e.queryTable(ctx, table, "7 days")
}

func (e *InfluxDBClient) QueryLast30Days(ctx context.Context, table string) ([]map[string]any, error) {
	return e.queryTable(ctx, table, "30 days")
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
