package redmaple

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
	citibike "github.com/mpoegel/red-maple/pkg/citibike"
	ha "github.com/mpoegel/red-maple/pkg/homeassistant"
	subway "github.com/mpoegel/red-maple/pkg/subway"
	weather "github.com/mpoegel/red-maple/pkg/weather"
)

type Server struct {
	s      http.Server
	config Config
	tz     *time.Location
	wg     sync.WaitGroup

	citibike   citibike.Client
	subwayCli  subway.Client
	weatherCli weather.Client
	haClient   ha.Client

	exportHub *ExportHub
}

func NewServer(config Config) (*Server, error) {
	mux := http.NewServeMux()

	tz, err := time.LoadLocation(config.Timezone)
	if err != nil {
		return nil, err
	}

	subwayCli, err := subway.NewClient(config.VendorDir)
	if err != nil {
		return nil, err
	}

	weatherCoords := strings.Split(config.WeatherLocation, ",")
	if len(weatherCoords) != 2 {
		return nil, errors.New("invalid weather coordinates")
	}
	weatherLat, err1 := strconv.ParseFloat(weatherCoords[0], 64)
	weatherLon, err2 := strconv.ParseFloat(weatherCoords[1], 64)
	if err1 != nil || err2 != nil {
		return nil, errors.Join(err1, err2)
	}

	s := Server{
		s: http.Server{
			Addr:         fmt.Sprintf("0.0.0.0:%d", config.Port),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			Handler:      mux,
		},
		config:     config,
		tz:         tz,
		wg:         sync.WaitGroup{},
		citibike:   citibike.NewClient(),
		subwayCli:  subwayCli,
		weatherCli: weather.NewClient(weatherLat, weatherLon, config.WeatherAPIKey),
		haClient:   ha.NewClient(config.HomeAssistant.Endpoint, config.HomeAssistant.APIKey),
		exportHub:  NewExportHub(config.ExportInterval),
	}

	if config.InfluxDB.Enabled {
		client, err := NewInfluxDBExporter(&config.InfluxDB)
		if err != nil {
			return nil, err
		}
		s.exportHub.AddExporter(client)
	}
	s.exportHub.AddProvider(s.haClient.GetProvider(
		s.config.HomeAssistant.IndoorTempID,
		s.config.HomeAssistant.IndoorHumidityID,
		s.config.HomeAssistant.OutdoorTempID,
		s.config.HomeAssistant.OutdoorHumidityID))

	s.LoadRoutes(mux)

	return &s, nil
}

func (s *Server) LoadRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", s.HandleIndex)
	mux.HandleFunc("GET /outdoor", s.HandleOutdoorFull)
	mux.HandleFunc("GET /indoor", s.HandleIndoorFull)
	mux.HandleFunc("GET /subway", s.HandleSubwayFull)
	mux.HandleFunc("GET /sunrise", s.HandleSunriseFull)
	mux.HandleFunc("GET /bikes", s.HandleBikesFull)
	mux.HandleFunc("GET /weather", s.HandleWeatherFull)
	mux.HandleFunc("GET /x/datetime", s.HandleDatetime)
	mux.HandleFunc("GET /x/citibike", s.HandleCitibike)
	mux.HandleFunc("GET /x/subway", s.HandleSubway)
	mux.HandleFunc("GET /x/weather", s.HandleWeather)
	mux.HandleFunc("GET /x/indoor", s.HandleIndoor)
	mux.HandleFunc("GET /x/outdoor", s.HandleOutdoor)
	mux.HandleFunc("GET /x/sunrise", s.HandleSunrise)
	mux.HandleFunc("GET /x/sundial", s.HandleSundial)
	mux.HandleFunc("GET /x/forecast", s.HandleForecastFull)
	mux.HandleFunc("GET /x/aqi", s.HandleAqiPartial)
	mux.HandleFunc("GET /x/sunrises", s.HandleSunrises)

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(s.config.StaticDir))))

	mux.Handle("GET /vendor/departure-mono/", http.StripPrefix("/vendor/departure-mono/", http.FileServer(http.Dir(s.config.VendorDir+"/departure-mono"))))
	mux.Handle("GET /vendor/htmx/", http.StripPrefix("/vendor/htmx/", http.FileServer(http.Dir(s.config.VendorDir+"/htmx"))))
	mux.Handle("GET /vendor/weather-icons/", http.StripPrefix("/vendor/weather-icons/", http.FileServer(http.Dir(s.config.VendorDir+"/weather-icons"))))
}

func (s *Server) Start(ctx context.Context) error {
	// start the export hub
	s.wg.Go(func() {
		s.exportHub.Run(ctx)
	})

	// start the HTTP server
	slog.Debug("listening", "addr", s.s.Addr)
	if err := s.s.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("http listen error", "error", err)
		return err
	}

	s.wg.Wait()
	return nil
}

func (s *Server) Stop(ctx context.Context) {
	s.s.Shutdown(ctx)
}

func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	s.executeTemplate(w, "Index", struct{}{})
}

func (s *Server) HandleDatetime(w http.ResponseWriter, r *http.Request) {
	now := time.Now().In(s.tz)
	AMorPM := "AM"
	hour := now.Hour()
	if now.Hour() >= 12 {
		AMorPM = "PM"
	}
	if hour > 13 {
		hour -= 12
	}
	s.executeTemplate(w, "Datetime", api.DatetimePartial{
		Timestamp: fmt.Sprintf("%02d:%02d", hour, now.Minute()),
		AMOrPM:    AMorPM,
		Seconds:   fmt.Sprintf("%02d", now.Second()),
		Date: fmt.Sprintf("%s %s %02d %d",
			strings.ToUpper(now.Weekday().String()),
			strings.ToUpper(now.Month().String())[:3],
			now.Day(),
			now.Year(),
		),
	})
}

func (s *Server) loadTemplates(w http.ResponseWriter) *template.Template {
	plate, err := template.ParseGlob(path.Join(s.config.StaticDir, "pages/*.html"))
	if err != nil {
		slog.Error("template html parse failure", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}
	plate, err = plate.ParseGlob(path.Join(s.config.StaticDir, "partials/*.html"))
	if err != nil {
		slog.Error("template snippets parse failure", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}
	return plate
}

func (s *Server) executeTemplate(w http.ResponseWriter, name string, data any) {
	// TODO optimize template loading
	plate := s.loadTemplates(w)
	if plate == nil {
		return
	}
	if err := plate.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("template execution failure", "name", name, "error", err, "data", data)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
