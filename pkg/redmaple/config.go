package redmaple

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port             int
	StaticDir        string
	VendorDir        string
	Timezone         string
	CitibikeStations []string
	SubwayStops      string
	WeatherLocation  string
	WeatherAPIKey    string
	HomeAssistant    HomeAssistantConfig
	ExportInterval   time.Duration
	InfluxDB         InfluxDBConfig
	S3               S3Config
}

type HomeAssistantConfig struct {
	Endpoint          string
	APIKey            string
	OutdoorTempID     string
	OutdoorHumidityID string
	IndoorTempID      string
	IndoorHumidityID  string
}

type InfluxDBConfig struct {
	Enabled  bool
	Endpoint string
	Token    string
	Database string
}

type S3Config struct {
	Enabled       bool
	Endpoint      string
	Scheme        string
	Bucket        string
	Region        string
	AccessKey     string
	SecretKey     string
	RetentionDays int
	FlushInterval time.Duration
}

func LoadConfig() Config {
	return Config{
		Port:             loadIntEnv("PORT", 6556),
		StaticDir:        loadStrEnv("STATIC_DIR", "./static"),
		VendorDir:        loadStrEnv("VENDOR_DIR", "./vendored"),
		Timezone:         loadStrEnv("TIMEZONE", "America/New_York"),
		CitibikeStations: loadStrListEnv("CITIBIKE_STATIONS", []string{"Park Ave & E 42 St", "Park Ave & E 41 St"}),
		SubwayStops:      loadStrEnv("SUBWAY_STOPS", "L03S,G29N"),
		WeatherLocation:  loadStrEnv("WEATHER_LOC", "40.75261,-73.97728"),
		WeatherAPIKey:    loadStrEnv("WEATHER_API_KEY", ""),
		HomeAssistant: HomeAssistantConfig{
			Endpoint:          loadStrEnv("HA_ENDPOINT", "http://localhost:8123"),
			APIKey:            loadStrEnv("HA_API_KEY", ""),
			OutdoorTempID:     loadStrEnv("HA_OUTDOOR_TEMP_ID", ""),
			OutdoorHumidityID: loadStrEnv("HA_OUTDOOR_HUMID_ID", ""),
			IndoorTempID:      loadStrEnv("HA_INDOOR_TEMP_ID", ""),
			IndoorHumidityID:  loadStrEnv("HA_INDOOR_HUMID_ID", ""),
		},
		ExportInterval: loadDurationEnv("EXPORT_INTERVAL", 1*time.Minute),
		InfluxDB: InfluxDBConfig{
			Enabled:  loadBoolEnv("INFLUXDB_ENABLED", false),
			Endpoint: loadStrEnv("INFLUXDB_ENDPOINT", ""),
			Token:    loadStrEnv("INFLUXDB_TOKEN", ""),
			Database: loadStrEnv("INFLUXDB_DATABASE", ""),
		},
		S3: S3Config{
			Enabled:       loadBoolEnv("S3_ENABLED", false),
			Endpoint:      loadStrEnv("S3_ENDPOINT", ""),
			Scheme:        loadStrEnv("S3_SCHEME", "https"),
			Bucket:        loadStrEnv("S3_BUCKET", ""),
			Region:        loadStrEnv("S3_REGION", "us-east-1"),
			AccessKey:     loadStrEnv("S3_ACCESS_KEY", ""),
			SecretKey:     loadStrEnv("S3_SECRET_KEY", ""),
			RetentionDays: loadIntEnv("S3_RETENTION_DAYS", 30),
			FlushInterval: loadDurationEnv("S3_FLUSH_INTERVAL", 1*time.Minute),
		},
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

func loadDurationEnv(name string, defaultVal time.Duration) time.Duration {
	valStr, ok := os.LookupEnv(name)
	if !ok {
		return defaultVal
	}
	val, err := time.ParseDuration(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func loadStrListEnv(name string, defaultVal []string) []string {
	val, ok := os.LookupEnv(name)
	if !ok {
		return defaultVal
	}
	return strings.Split(val, ",")
}
