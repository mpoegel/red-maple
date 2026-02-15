package api

import "context"

type Importer interface {
	QueryLast24Hours(ctx context.Context, table string) ([]map[string]any, error)
	QueryLast7Days(ctx context.Context, table string) ([]map[string]any, error)
	QueryLast30Days(ctx context.Context, table string) ([]map[string]any, error)
}
