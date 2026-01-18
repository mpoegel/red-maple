package redmaple

import (
	"log/slog"
	"math"
	"net/http"
	"strconv"

	api "github.com/mpoegel/red-maple/pkg/api"
)

func (s *Server) HandleIndoor(w http.ResponseWriter, r *http.Request) {
	lastTempData := s.haClient.DeviceCache(s.config.HomeAssistant.IndoorTempID)
	sensorTempData, err := s.haClient.GetDeviceState(r.Context(), s.config.HomeAssistant.IndoorTempID)
	if err != nil {
		slog.Error("failed to get indoor temperature sensor", "err", err, "device", s.config.HomeAssistant.IndoorTempID)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	lastHumidData := s.haClient.DeviceCache(s.config.HomeAssistant.IndoorTempID)
	sensorHumidData, err := s.haClient.GetDeviceState(r.Context(), s.config.HomeAssistant.IndoorHumidityID)
	if err != nil {
		slog.Error("failed to get indoor humidity sensor", "err", err, "device", s.config.HomeAssistant.IndoorHumidityID)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	slog.Debug("got indoor sensor update", "temp", sensorTempData, "humidity", sensorHumidData)

	currTemp, err1 := strconv.ParseFloat(sensorTempData.State, 64)
	currHumid, err2 := strconv.ParseFloat(sensorHumidData.State, 64)
	if err1 != nil || err2 != nil {
		slog.Error("indoor sensor returned invalid state", "err", err, "temp", sensorTempData.State, "humidity", sensorHumidData.State)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	lastTemp := 0.0
	lastHumid := 0.0
	if lastTempData != nil {
		lastTemp, _ = strconv.ParseFloat(lastTempData.State, 64)
	}
	if lastHumidData != nil {
		lastHumid, _ = strconv.ParseFloat(lastHumidData.State, 64)
	}

	intTemp, fracTemp := math.Modf(currTemp)
	intHumid, fracHumid := math.Modf(currHumid)
	data := api.IndoorPartial{
		IntegerTemp:          int(intTemp),
		FractionalTemp:       int(math.Floor(fracTemp * 100)),
		IntegerHumidity:      int(intHumid),
		FractionalHumidity:   int(math.Floor(fracHumid * 100)),
		IsTempTrendingUp:     lastTemp < currTemp,
		IsHumidityTrendingUp: lastHumid < currHumid,
	}
	if data.IntegerHumidity > 60 {
		data.HumidityLevel = 2
	} else if data.IntegerHumidity >= 40 {
		data.HumidityLevel = 1
	}
	s.executeTemplate(w, "Indoor", data)
}

func (s *Server) HandleOutdoor(w http.ResponseWriter, r *http.Request) {
	lastTempData := s.haClient.DeviceCache(s.config.HomeAssistant.IndoorTempID)
	sensorTempData, err := s.haClient.GetDeviceState(r.Context(), s.config.HomeAssistant.OutdoorTempID)
	if err != nil {
		slog.Error("failed to get outdoor temperature sensor", "err", err, "device", s.config.HomeAssistant.OutdoorTempID)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	lastHumidData := s.haClient.DeviceCache(s.config.HomeAssistant.IndoorTempID)
	sensorHumidData, err := s.haClient.GetDeviceState(r.Context(), s.config.HomeAssistant.OutdoorHumidityID)
	if err != nil {
		slog.Error("failed to get outdoor humidity sensor", "err", err, "device", s.config.HomeAssistant.OutdoorHumidityID)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	slog.Debug("got outdoor sensor update", "temp", sensorTempData, "humidity", sensorHumidData)

	currTemp, err1 := strconv.ParseFloat(sensorTempData.State, 64)
	currHumid, err2 := strconv.ParseFloat(sensorHumidData.State, 64)
	if err1 != nil || err2 != nil {
		slog.Error("indoor sensor returned invalid state", "err", err, "temp", sensorTempData.State, "humidity", sensorHumidData.State)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	lastTemp := 0.0
	lastHumid := 0.0
	if lastTempData != nil {
		lastTemp, _ = strconv.ParseFloat(lastTempData.State, 64)
	}
	if lastHumidData != nil {
		lastHumid, _ = strconv.ParseFloat(lastHumidData.State, 64)
	}

	intTemp, fracTemp := math.Modf(currTemp)
	intHumid, fracHumid := math.Modf(currHumid)
	data := api.OutdoorPartial{
		IntegerTemp:          int(intTemp),
		FractionalTemp:       int(math.Floor(fracTemp * 100)),
		IntegerHumidity:      int(intHumid),
		FractionalHumidity:   int(math.Floor(fracHumid * 100)),
		IsTempTrendingUp:     lastTemp < currTemp,
		IsHumidityTrendingUp: lastHumid < currHumid,
	}
	if data.IntegerHumidity > 60 {
		data.HumidityLevel = 2
	} else if data.IntegerHumidity >= 40 {
		data.HumidityLevel = 1
	}
	s.executeTemplate(w, "Outdoor", data)
}

func (s *Server) HandleOutdoorFull(w http.ResponseWriter, r *http.Request) {
	s.executeTemplate(w, "OutdoorFull", struct{}{})
}
