package api

import (
	"context"
	"time"
)

type Importer interface {
	QueryRange(ctx context.Context, table string, duration time.Duration) ([]*DataPoint, error)
}
