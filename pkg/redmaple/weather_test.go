package redmaple_test

import (
	"testing"
	"time"

	redmaple "github.com/mpoegel/red-maple/pkg/redmaple"
)

func TestCalculateAQI(t *testing.T) {
	tests := []struct {
		name          string
		concentration float64
		breakpoints   []float64
		expectedAQI   int
	}{
		{
			name:          "PM2.5 low concentration",
			concentration: 10.0,
			breakpoints:   []float64{0.0, 9.0, 9.1, 35.4, 35.5, 55.4, 55.5, 125.4, 125.5, 225.4, 225.5},
			expectedAQI:   53,
		},
		{
			name:          "PM2.5 moderate concentration",
			concentration: 30.0,
			breakpoints:   []float64{0.0, 9.0, 9.1, 35.4, 35.5, 55.4, 55.5, 125.4, 125.5, 225.4, 225.5},
			expectedAQI:   90,
		},
		{
			name:          "zero concentration",
			concentration: 0.0,
			breakpoints:   []float64{0.0, 9.0, 9.1, 35.4, 35.5, 55.4, 55.5, 125.4, 125.5, 225.4, 225.5},
			expectedAQI:   0,
		},
		{
			name:          "high concentration",
			concentration: 200.0,
			breakpoints:   []float64{0.0, 9.0, 9.1, 35.4, 35.5, 55.4, 55.5, 125.4, 125.5, 225.4, 225.5},
			expectedAQI:   275,
		},
		{
			name:          "ozone low",
			concentration: 0.040,
			breakpoints:   []float64{0.0, 0.054, 0.055, 0.070, 0.071, 0.085, 0.086, 0.105, 0.106, 0.200, 0.201},
			expectedAQI:   37,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redmaple.CalculateAQI(tt.concentration, tt.breakpoints)
			if result != tt.expectedAQI {
				t.Errorf("CalculateAQI(%v, _) = %d, want %d", tt.concentration, result, tt.expectedAQI)
			}
		})
	}
}

func TestHourStamp(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "midnight",
			time:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: "12 AM",
		},
		{
			name:     "morning",
			time:     time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC),
			expected: "9 AM",
		},
		{
			name:     "noon",
			time:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: "12 PM",
		},
		{
			name:     "afternoon",
			time:     time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC),
			expected: "3 PM",
		},
		{
			name:     "evening",
			time:     time.Date(2024, 1, 1, 23, 59, 0, 0, time.UTC),
			expected: "11 PM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redmaple.HourStamp(tt.time)
			if result != tt.expected {
				t.Errorf("HourStamp(%v) = %q, want %q", tt.time, result, tt.expected)
			}
		})
	}
}

func TestMoonPhaseToIcon(t *testing.T) {
	tests := []struct {
		phase    int
		expected string
	}{
		{0, "wi-moon-new"},
		{7, "wi-moon-first-quarter"},
		{14, "wi-moon-full"},
		{21, "wi-moon-third-quarter"},
		{1, "wi-moon-waxing-crescent-1"},
		{4, "wi-moon-waxing-crescent-4"},
		{8, "wi-moon-waxing-gibbous-1"},
		{11, "wi-moon-waxing-gibbous-4"},
		{15, "wi-moon-waning-gibbous-1"},
		{18, "wi-moon-waning-gibbous-4"},
		{22, "wi-moon-waning-crescent-1"},
		{25, "wi-moon-waning-crescent-4"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := redmaple.MoonPhaseToIcon(tt.phase)
			if result != tt.expected {
				t.Errorf("MoonPhaseToIcon(%d) = %q, want %q", tt.phase, result, tt.expected)
			}
		})
	}
}
