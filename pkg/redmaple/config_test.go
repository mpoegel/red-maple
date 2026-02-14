package redmaple_test

import (
	"testing"
	"time"

	redmaple "github.com/mpoegel/red-maple/pkg/redmaple"
)

func TestLoadConfigWithValidValues(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv("STATIC_DIR", "/custom/static")
	t.Setenv("VENDOR_DIR", "/custom/vendor")
	t.Setenv("TIMEZONE", "America/Los_Angeles")
	t.Setenv("CITIBIKE_STATIONS", "Station A,Station B,Station C")
	t.Setenv("SUBWAY_STOPS", "L03N,L04S")
	t.Setenv("WEATHER_LOC", "40.7128,-74.0060")
	t.Setenv("WEATHER_API_KEY", "test-api-key-123")
	t.Setenv("HA_ENDPOINT", "http://192.168.1.100:8123")
	t.Setenv("HA_API_KEY", "test-ha-token")
	t.Setenv("HA_OUTDOOR_TEMP_ID", "sensor.outdoor_temp")
	t.Setenv("HA_OUTDOOR_HUMID_ID", "sensor.outdoor_humidity")
	t.Setenv("HA_INDOOR_TEMP_ID", "sensor.indoor_temp")
	t.Setenv("HA_INDOOR_HUMID_ID", "sensor.indoor_humidity")
	t.Setenv("INFLUXDB_ENABLED", "true")
	t.Setenv("INFLUXDB_ENDPOINT", "https://us-east-1-1.aws.cloud2.influxdata.com")
	t.Setenv("INFLUXDB_TOKEN", "test-influx-token")
	t.Setenv("INFLUXDB_DATABASE", "mydb")
	t.Setenv("EXPORT_INTERVAL", "30s")

	config := redmaple.LoadConfig()

	if config.Port != 8080 {
		t.Errorf("expected PORT=8080, got %d", config.Port)
	}
	if config.StaticDir != "/custom/static" {
		t.Errorf("expected STATIC_DIR=/custom/static, got %s", config.StaticDir)
	}
	if config.VendorDir != "/custom/vendor" {
		t.Errorf("expected VENDOR_DIR=/custom/vendor, got %s", config.VendorDir)
	}
	if config.Timezone != "America/Los_Angeles" {
		t.Errorf("expected TIMEZONE=America/Los_Angeles, got %s", config.Timezone)
	}
	if len(config.CitibikeStations) != 3 {
		t.Errorf("expected 3 CITIBIKE_STATIONS, got %d", len(config.CitibikeStations))
	}
	if config.CitibikeStations[0] != "Station A" {
		t.Errorf("expected first station=Station A, got %s", config.CitibikeStations[0])
	}
	if config.CitibikeStations[1] != "Station B" {
		t.Errorf("expected second station=Station B, got %s", config.CitibikeStations[1])
	}
	if config.CitibikeStations[2] != "Station C" {
		t.Errorf("expected third station=Station C, got %s", config.CitibikeStations[2])
	}
	if config.SubwayStops != "L03N,L04S" {
		t.Errorf("expected SUBWAY_STOPS=L03N,L04S, got %s", config.SubwayStops)
	}
	if config.WeatherLocation != "40.7128,-74.0060" {
		t.Errorf("expected WEATHER_LOC=40.7128,-74.0060, got %s", config.WeatherLocation)
	}
	if config.WeatherAPIKey != "test-api-key-123" {
		t.Errorf("expected WEATHER_API_KEY=test-api-key-123, got %s", config.WeatherAPIKey)
	}
	if config.HomeAssistant.Endpoint != "http://192.168.1.100:8123" {
		t.Errorf("expected HA_ENDPOINT=http://192.168.1.100:8123, got %s", config.HomeAssistant.Endpoint)
	}
	if config.HomeAssistant.APIKey != "test-ha-token" {
		t.Errorf("expected HA_API_KEY=test-ha-token, got %s", config.HomeAssistant.APIKey)
	}
	if config.HomeAssistant.OutdoorTempID != "sensor.outdoor_temp" {
		t.Errorf("expected HA_OUTDOOR_TEMP_ID=sensor.outdoor_temp, got %s", config.HomeAssistant.OutdoorTempID)
	}
	if config.HomeAssistant.OutdoorHumidityID != "sensor.outdoor_humidity" {
		t.Errorf("expected HA_OUTDOOR_HUMID_ID=sensor.outdoor_humidity, got %s", config.HomeAssistant.OutdoorHumidityID)
	}
	if config.HomeAssistant.IndoorTempID != "sensor.indoor_temp" {
		t.Errorf("expected HA_INDOOR_TEMP_ID=sensor.indoor_temp, got %s", config.HomeAssistant.IndoorTempID)
	}
	if config.HomeAssistant.IndoorHumidityID != "sensor.indoor_humidity" {
		t.Errorf("expected HA_INDOOR_HUMID_ID=sensor.indoor_humidity, got %s", config.HomeAssistant.IndoorHumidityID)
	}
	if !config.InfluxDB.Enabled {
		t.Errorf("expected INFLUXDB_ENABLED=true, got %v", config.InfluxDB.Enabled)
	}
	if config.InfluxDB.Endpoint != "https://us-east-1-1.aws.cloud2.influxdata.com" {
		t.Errorf("expected INFLUXDB_ENDPOINT, got %s", config.InfluxDB.Endpoint)
	}
	if config.InfluxDB.Token != "test-influx-token" {
		t.Errorf("expected INFLUXDB_TOKEN, got %s", config.InfluxDB.Token)
	}
	if config.InfluxDB.Database != "mydb" {
		t.Errorf("expected INFLUXDB_DATABASE=mydb, got %s", config.InfluxDB.Database)
	}
	if config.ExportInterval != 30*time.Second {
		t.Errorf("expected EXPORT_INTERVAL=30s, got %v", config.ExportInterval)
	}
}

func TestLoadConfigWithInvalidValues(t *testing.T) {
	t.Setenv("PORT", "abc")
	config := redmaple.LoadConfig()
	if config.Port != 6556 {
		t.Errorf("expected PORT=6556 (default), got %d", config.Port)
	}

	t.Setenv("PORT", "-1")
	config = redmaple.LoadConfig()
	if config.Port != -1 {
		t.Errorf("expected PORT=-1 (invalid but parseable), got %d", config.Port)
	}

	t.Setenv("INFLUXDB_ENABLED", "invalid")
	config = redmaple.LoadConfig()
	if config.InfluxDB.Enabled != false {
		t.Errorf("expected INFLUXDB_ENABLED=false (default), got %v", config.InfluxDB.Enabled)
	}

	t.Setenv("INFLUXDB_ENABLED", "maybe")
	config = redmaple.LoadConfig()
	if config.InfluxDB.Enabled != false {
		t.Errorf("expected INFLUXDB_ENABLED=false (default), got %v", config.InfluxDB.Enabled)
	}

	t.Setenv("EXPORT_INTERVAL", "not-a-duration")
	config = redmaple.LoadConfig()
	if config.ExportInterval != 1*time.Minute {
		t.Errorf("expected EXPORT_INTERVAL=1m (default), got %v", config.ExportInterval)
	}
}
