# Red Maple

A Go-based home dashboard server that displays weather, subway arrivals, Citibike availability, and indoor/outdoor sensor data. Built with HTMX for dynamic partial page updates.

## Features

- **Weather** - Current conditions and forecast from OpenWeatherMap
- **Subway** - Real-time arrivals for NYC subway stations (GTFS)
- **Citibike** - Live bike availability at configured stations
- **Sensors** - Indoor/outdoor temperature and humidity from Home Assistant
- **Sunrise/Sunset** - Daily sunrise, sunset, and twilight times
- **AQI** - Air Quality Index data
- **InfluxDB Export** - Optional export of sensor data to InfluxDB for time-series analysis

## Prerequisites

- Go 1.21+
- OpenWeatherMap API key (for weather data)
- Home Assistant instance (optional, for sensor data)
- InfluxDB (optional, for data export)

## Running the Project

```bash
# Clone and navigate to the project
cd red-maple

# Install dependencies
go mod download

# Run the application
go run .
```

Or with environment variables:

```bash
# Source the default environment file
source extra/default.env

# Or set your own values
export PORT=6556
export WEATHER_API_KEY=your_api_key
export HA_ENDPOINT=http://localhost:8123
export HA_API_KEY=your_ha_token
# ... other config options

go run .
```

Build a binary:

```bash
go build -o red-maple .
```

## Configuration

All configuration is done via environment variables.

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `6556` | HTTP server port |
| `STATIC_DIR` | `./static` | Directory containing HTML templates and static files |
| `VENDOR_DIR` | `./vendored` | Directory for vendored assets (fonts, libraries) |
| `TIMEZONE` | `America/New_York` | Location timezone (IANA format) |

### Weather

| Variable | Default | Description |
|----------|---------|-------------|
| `WEATHER_LOC` | `40.75261,-73.97728` | Latitude,longitude for weather data |
| `WEATHER_API_KEY` | (none) | OpenWeatherMap API key |

### Subway

| Variable | Default | Description |
|----------|---------|-------------|
| `SUBWAY_STOPS` | `L03S,G29N` | Comma-separated list of NYC subway stop IDs |

Stop IDs are in the format `<station>-<direction>` where direction is `N` (northbound), `S` (southbound), `E` (eastbound), or `W` (westbound). Example: `L03S` is the 14th Street-Union Square station on the L line to Brooklyn.

### Citibike

| Variable | Default | Description |
|----------|---------|-------------|
| `CITIBIKE_STATIONS` | `Park Ave & E 42 St,Park Ave & E 41 St` | Comma-separated list of station names |

### Home Assistant

| Variable | Default | Description |
|----------|---------|-------------|
| `HA_ENDPOINT` | `http://localhost:8123` | Home Assistant URL |
| `HA_API_KEY` | (none) | Home Assistant Long-Lived Access Token |
| `HA_OUTDOOR_TEMP_ID` | (none) | Entity ID for outdoor temperature sensor |
| `HA_OUTDOOR_HUMID_ID` | (none) | Entity ID for outdoor humidity sensor |
| `HA_INDOOR_TEMP_ID` | (none) | Entity ID for indoor temperature sensor |
| `HA_INDOOR_HUMID_ID` | (none) | Entity ID for indoor humidity sensor |

### InfluxDB Export

| Variable | Default | Description |
|----------|---------|-------------|
| `INFLUXDB_ENABLED` | `false` | Enable InfluxDB export |
| `INFLUXDB_ENDPOINT` | (none) | InfluxDB Cloud endpoint URL |
| `INFLUXDB_TOKEN` | (none) | InfluxDB authentication token |
| `INFLUXDB_DATABASE` | (none) | InfluxDB database name |
| `EXPORT_INTERVAL` | `1m` | Interval between data exports (duration format) |

## Endpoints

The server provides both full pages and HTMX partials:

| Endpoint | Description |
|----------|-------------|
| `/` | Main dashboard page |
| `/weather` | Weather page |
| `/outdoor` | Outdoor conditions page |
| `/indoor` | Indoor sensor page |
| `/subway` | Subway arrivals page |
| `/bikes` | Citibike availability page |
| `/sunrise` | Sunrise/sunset times page |
| `/x/*` | HTMX partials (e.g., `/x/weather`, `/x/citibike`) |

## Project Structure

```
red-maple/
├── main.go                 # Application entry point
├── pkg/
│   ├── redmaple/          # Core server package
│   │   ├── server.go      # HTTP server
│   │   └── config.go      # Configuration
│   ├── weather/           # OpenWeatherMap client
│   ├── citibike/          # Citibike API client
│   ├── subway/            # NYC Subway GTFS client
│   ├── homeassistant/     # Home Assistant client
│   └── api/               # Shared API types
├── static/                # HTML, CSS, templates
│   ├── pages/             # Full page templates
│   └── partials/          # HTMX partials
└── vendored/              # Vendored assets
```
