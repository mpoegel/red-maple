package redmaple_test

import (
	"testing"
	"time"

	citibike "github.com/mpoegel/red-maple/pkg/citibike"
	"github.com/mpoegel/red-maple/pkg/redmaple"
)

func TestCompactToBuckets_EmptyHistory(t *testing.T) {
	result := redmaple.CompactToBuckets(nil, 1, "all")
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestCompactToBuckets_EmptySlice(t *testing.T) {
	result := redmaple.CompactToBuckets([]citibike.HistoricalBikeCount{}, 1, "all")
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestCompactToBuckets_24Buckets_Day1(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	history := make([]citibike.HistoricalBikeCount, 24)
	for i := 0; i < 24; i++ {
		history[i] = citibike.HistoricalBikeCount{
			Classics: i + 1,
			Ebikes:   i + 10,
			Stamp:    baseTime.Add(time.Duration(i) * time.Hour),
		}
	}

	result := redmaple.CompactToBuckets(history, 1, "all")

	if len(result) != 24 {
		t.Errorf("expected 24 buckets, got %d", len(result))
	}

	if result[0].Min != 11 {
		t.Errorf("expected first bucket min=11 (1+10), got %d", result[0].Min)
	}
	if result[0].Max != 11 {
		t.Errorf("expected first bucket max=11 (1+10), got %d", result[0].Max)
	}
}

func TestCompactToBuckets_21Buckets_Day7(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	history := make([]citibike.HistoricalBikeCount, 21)
	for i := 0; i < 21; i++ {
		history[i] = citibike.HistoricalBikeCount{
			Classics: i * 2,
			Ebikes:   i * 3,
			Stamp:    baseTime.Add(time.Duration(i) * 8 * time.Hour),
		}
	}

	result := redmaple.CompactToBuckets(history, 7, "all")

	if len(result) != 21 {
		t.Errorf("expected 21 buckets, got %d", len(result))
	}

	if result[0].Min != 0 {
		t.Errorf("expected first bucket min=0, got %d", result[0].Min)
	}
	if result[0].Max != 0 {
		t.Errorf("expected first bucket max=0 (0+0), got %d", result[0].Max)
	}
}

func TestCompactToBuckets_30Buckets_Day30(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	history := make([]citibike.HistoricalBikeCount, 30)
	for i := 0; i < 30; i++ {
		history[i] = citibike.HistoricalBikeCount{
			Classics: i + 5,
			Ebikes:   i + 15,
			Stamp:    baseTime.Add(time.Duration(i) * 24 * time.Hour),
		}
	}

	result := redmaple.CompactToBuckets(history, 30, "all")

	if len(result) != 30 {
		t.Errorf("expected 30 buckets, got %d", len(result))
	}

	if result[0].Min != 20 {
		t.Errorf("expected first bucket min=20 (5+15), got %d", result[0].Min)
	}
	if result[0].Max != 20 {
		t.Errorf("expected first bucket max=20 (5+15), got %d", result[0].Max)
	}
}

func TestCompactToBuckets_ClassicOnly(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	history := []citibike.HistoricalBikeCount{
		{Classics: 10, Ebikes: 50, Stamp: baseTime},
		{Classics: 15, Ebikes: 60, Stamp: baseTime.Add(time.Hour)},
	}

	result := redmaple.CompactToBuckets(history, 1, "classic")

	if len(result) != 2 {
		t.Errorf("expected 2 buckets, got %d", len(result))
	}

	if result[0].Min != 10 || result[0].Max != 10 {
		t.Errorf("expected first bucket min=10 max=10, got min=%d max=%d", result[0].Min, result[0].Max)
	}
	if result[1].Min != 15 || result[1].Max != 15 {
		t.Errorf("expected second bucket min=15 max=15, got min=%d max=%d", result[1].Min, result[1].Max)
	}
}

func TestCompactToBuckets_ElectricOnly(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	history := []citibike.HistoricalBikeCount{
		{Classics: 10, Ebikes: 5, Stamp: baseTime},
		{Classics: 20, Ebikes: 8, Stamp: baseTime.Add(time.Hour)},
	}

	result := redmaple.CompactToBuckets(history, 1, "electric")

	if len(result) != 2 {
		t.Errorf("expected 2 buckets, got %d", len(result))
	}

	if result[0].Min != 5 || result[0].Max != 5 {
		t.Errorf("expected first bucket min=5 max=5, got min=%d max=%d", result[0].Min, result[0].Max)
	}
	if result[1].Min != 8 || result[1].Max != 8 {
		t.Errorf("expected second bucket min=8 max=8, got min=%d max=%d", result[1].Min, result[1].Max)
	}
}

func TestCompactToBuckets_AllBikes(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	history := []citibike.HistoricalBikeCount{
		{Classics: 10, Ebikes: 5, Stamp: baseTime},
		{Classics: 20, Ebikes: 8, Stamp: baseTime.Add(time.Hour)},
	}

	result := redmaple.CompactToBuckets(history, 1, "all")

	if result[0].Min != 15 || result[0].Max != 15 {
		t.Errorf("expected first bucket min=15 max=15 (10+5), got min=%d max=%d", result[0].Min, result[0].Max)
	}
	if result[1].Min != 28 || result[1].Max != 28 {
		t.Errorf("expected second bucket min=28 max=28 (20+8), got min=%d max=%d", result[1].Min, result[1].Max)
	}
}

func TestCompactToBuckets_MinMaxAggregation(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	history := []citibike.HistoricalBikeCount{
		{Classics: 5, Ebikes: 3, Stamp: baseTime},
		{Classics: 10, Ebikes: 8, Stamp: baseTime.Add(30 * time.Minute)},
		{Classics: 7, Ebikes: 5, Stamp: baseTime.Add(45 * time.Minute)},
	}

	result := redmaple.CompactToBuckets(history, 1, "all")

	if len(result) != 1 {
		t.Errorf("expected 1 bucket, got %d", len(result))
	}

	if result[0].Min != 8 {
		t.Errorf("expected min=8 (5+3), got %d", result[0].Min)
	}
	if result[0].Max != 18 {
		t.Errorf("expected max=18 (10+8), got %d", result[0].Max)
	}
}

func TestCompactToBuckets_OutOfOrder(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)
	history := []citibike.HistoricalBikeCount{
		{Classics: 20, Ebikes: 0, Stamp: baseTime.Add(time.Hour)},
		{Classics: 10, Ebikes: 0, Stamp: baseTime},
	}

	result := redmaple.CompactToBuckets(history, 1, "classic")

	if len(result) != 2 {
		t.Errorf("expected 2 buckets, got %d", len(result))
	}

	if result[0].Min != 10 || result[0].Max != 10 {
		t.Errorf("expected first bucket min=10 max=10, got min=%d max=%d", result[0].Min, result[0].Max)
	}
	if result[1].Min != 20 || result[1].Max != 20 {
		t.Errorf("expected second bucket min=20 max=20, got min=%d max=%d", result[1].Min, result[1].Max)
	}
}

func TestCompactToBuckets_DefaultDays(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	history := []citibike.HistoricalBikeCount{
		{Classics: 10, Ebikes: 5, Stamp: baseTime},
	}

	result := redmaple.CompactToBuckets(history, 0, "all")

	if len(result) != 1 {
		t.Errorf("expected 1 bucket (default to 24), got %d", len(result))
	}
}

func TestCompactToBuckets_DataOutsideTimeRange(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	history := []citibike.HistoricalBikeCount{
		{Classics: 5, Ebikes: 3, Stamp: baseTime},
		{Classics: 10, Ebikes: 8, Stamp: baseTime.Add(25 * time.Hour)},
	}

	result := redmaple.CompactToBuckets(history, 1, "all")

	if len(result) != 2 {
		t.Errorf("expected 2 buckets (data clamped to first and last bucket), got %d", len(result))
	}

	if result[0].Min != 8 {
		t.Errorf("expected first bucket min=8, got %d", result[0].Min)
	}
	if result[1].Min != 18 {
		t.Errorf("expected last bucket min=18, got %d", result[1].Min)
	}
}
