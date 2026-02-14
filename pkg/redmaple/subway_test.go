package redmaple_test

import (
	"testing"
	"time"

	redmaple "github.com/mpoegel/red-maple/pkg/redmaple"
)

func TestMinutesUntilArrival(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	now := time.Now().In(loc)

	tests := []struct {
		name        string
		arrival     int64
		minExpected int
		maxExpected int
	}{
		{
			name:        "future arrival",
			arrival:     now.Add(10 * time.Minute).Unix(),
			minExpected: 9,
			maxExpected: 10,
		},
		{
			name:        "arrival in 5 minutes",
			arrival:     now.Add(5 * time.Minute).Unix(),
			minExpected: 4,
			maxExpected: 5,
		},
		{
			name:        "arrival in 1 minute",
			arrival:     now.Add(1 * time.Minute).Unix(),
			minExpected: 0,
			maxExpected: 1,
		},
		{
			name:        "arrival now",
			arrival:     now.Unix(),
			minExpected: 0,
			maxExpected: 0,
		},
		{
			name:        "past arrival",
			arrival:     now.Add(-5 * time.Minute).Unix(),
			minExpected: -5,
			maxExpected: -4,
		},
		{
			name:        "far future",
			arrival:     now.Add(60 * time.Minute).Unix(),
			minExpected: 59,
			maxExpected: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redmaple.MinutesUntilArrival(tt.arrival, loc)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("MinutesUntilArrival(%d, _) = %d, want between %d and %d", tt.arrival, result, tt.minExpected, tt.maxExpected)
			}
		})
	}
}
