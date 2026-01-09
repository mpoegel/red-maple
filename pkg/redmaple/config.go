package redmaple

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port             int
	StaticDir        string
	VendorDir        string
	Timezone         string
	CitibikeStations string
	SubwayStops      string
	WeatherLocation  string
	WeatherAPIKey    string
}

func LoadConfig() Config {
	return Config{
		Port:             loadIntEnv("PORT", 6556),
		StaticDir:        loadStrEnv("STATIC_DIR", "./static"),
		VendorDir:        loadStrEnv("VENDOR_DIR", "./vendored"),
		Timezone:         loadStrEnv("TIMEZONE", "America/New_York"),
		CitibikeStations: loadStrEnv("CITIBIKE_STATIONS", "Park Ave & E 42 St,Park Ave & E 41 St"),
		SubwayStops:      loadStrEnv("SUBWAY_STOPS", "L03S,G29N"),
		WeatherLocation:  loadStrEnv("WEATHER_LOC", "40.75261,-73.97728"),
		WeatherAPIKey:    loadStrEnv("WEATHER_API_KEY", ""),
	}
}

func loadStrEnv(name, defaultVal string) string {
	val, ok := os.LookupEnv(name)
	if !ok {
		return defaultVal
	}
	return val
}

func loadBoolEnv(name string, defaultVal bool) bool {
	val, ok := os.LookupEnv(name)
	if !ok {
		return defaultVal
	}
	return strings.ToLower(val) == "true" || val == "1"
}

func loadIntEnv(name string, defaultVal int) int {
	valStr, ok := os.LookupEnv(name)
	if !ok {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
