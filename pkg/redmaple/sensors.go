package redmaple

import (
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
	homeassistant "github.com/mpoegel/red-maple/pkg/homeassistant"
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

func (s *Server) HandleIndoorFull(w http.ResponseWriter, r *http.Request) {
	s.executeTemplate(w, "IndoorFull", struct{}{})
}

func (s *Server) HandleIndoorHistory(w http.ResponseWriter, r *http.Request) {
	s.HandleSensorHistory(w, r, "Indoor")
}

func (s *Server) HandleOutdoorHistory(w http.ResponseWriter, r *http.Request) {
	s.HandleSensorHistory(w, r, "Outdoor")
}

func (s *Server) HandleSensorHistory(w http.ResponseWriter, r *http.Request, region string) {
	if s.importer == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	days := 1
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			days = parsed
		}
	}

	dataname := r.URL.Query().Get("dataname")
	if dataname == "" {
		dataname = "temperature"
	}

	slog.Debug("history request", "region", region, "days", days, "dataname", dataname)

	var history []homeassistant.DeviceHistory
	var err error
	switch region {
	case "Outdoor":
		if dataname == "humidity" {
			history, err = s.haClient.GetDeviceHistory(r.Context(), s.importer, s.config.HomeAssistant.OutdoorHumidityID, 24*time.Hour*time.Duration(days))
		} else {
			history, err = s.haClient.GetDeviceHistory(r.Context(), s.importer, s.config.HomeAssistant.OutdoorTempID, 24*time.Hour*time.Duration(days))
		}
	default:
		if dataname == "humidity" {
			history, err = s.haClient.GetDeviceHistory(r.Context(), s.importer, s.config.HomeAssistant.IndoorHumidityID, 24*time.Hour*time.Duration(days))
		} else {
			history, err = s.haClient.GetDeviceHistory(r.Context(), s.importer, s.config.HomeAssistant.IndoorTempID, 24*time.Hour*time.Duration(days))
		}
	}
	if err != nil {
		slog.Error("failed to get device history", "err", err, "days", days)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	slog.Debug("raw device history", "data", history)
	buckets := CompactToBucketsFromDevice(history, days)
	slog.Debug("device history", "buckets", buckets)

	var data []api.GraphPoint
	minY := 100
	maxY := 0
	for _, b := range buckets {
		minY = min(minY, b.Min)
		maxY = max(maxY, b.Max)
	}
	minY--
	maxY++

	yDiff := max(1, maxY-minY)
	for _, b := range buckets {
		bottom := int(200.0 / float64(yDiff) * float64(b.Min-minY))
		data = append(data, api.GraphPoint{
			Min:   bottom,
			Max:   int(200.0/float64(yDiff)*float64(b.Max-minY)) - bottom,
			Width: 200.0 / float64(len(buckets)),
		})
	}

	if len(data) == 0 {
		data = append(data, api.GraphPoint{Min: 0, Max: 0, Width: 100.0})
		minY = 0
		maxY = 0
	}

	var startTimeStr, endTimeStr string
	if len(history) > 0 {
		if days == 1 {
			startTimeStr = history[0].Stamp.In(s.tz).Format("3PM")
			endTimeStr = history[len(history)-1].Stamp.In(s.tz).Format("3PM")
		} else {
			startTimeStr = history[0].Stamp.In(s.tz).Format("Jan02")
			endTimeStr = history[len(history)-1].Stamp.In(s.tz).Format("Jan02")
		}
	} else {
		if days == 1 {
			startTimeStr = "12AM"
			endTimeStr = "11PM"
		} else {
			startTimeStr = "Jan01"
			endTimeStr = "Dec31"
		}
	}

	dataPayload := api.OutdoorHistory{
		Days:      days,
		DataName:  dataname,
		MaxY:      maxY,
		MinY:      minY,
		Data:      data,
		StartTime: startTimeStr,
		EndTime:   endTimeStr,
	}

	slog.Debug("history", "region", region, "data", dataPayload)

	s.executeTemplate(w, region+"History", dataPayload)
}

func CompactToBucketsFromDevice(history []homeassistant.DeviceHistory, days int) []Bucket {
	if len(history) == 0 {
		return nil
	}

	sort.Slice(history, func(i, j int) bool {
		return history[i].Stamp.Before(history[j].Stamp)
	})

	var numBuckets int
	switch days {
	case 1:
		numBuckets = 24
	case 7:
		numBuckets = 21
	case 30:
		numBuckets = 30
	default:
		numBuckets = 24
	}

	buckets := make([]Bucket, numBuckets)
	for i := 0; i < numBuckets; i++ {
		buckets[i].Min = -9999
	}

	firstTime := history[0].Stamp
	var duration time.Duration
	switch days {
	case 1:
		duration = 24 * time.Hour
	case 7:
		duration = 7 * 24 * time.Hour
	case 30:
		duration = 30 * 24 * time.Hour
	default:
		duration = 24 * time.Hour
	}
	bucketDuration := duration / time.Duration(numBuckets)

	for _, h := range history {
		elapsed := h.Stamp.Sub(firstTime)
		bucketIndex := int(elapsed / bucketDuration)
		if bucketIndex >= numBuckets {
			bucketIndex = numBuckets - 1
		}
		if bucketIndex < 0 {
			bucketIndex = 0
		}

		value := int(math.Round(h.Value))
		if buckets[bucketIndex].Min == -9999 {
			buckets[bucketIndex].Min = value
			buckets[bucketIndex].Max = value
		} else {
			buckets[bucketIndex].Min = min(buckets[bucketIndex].Min, value)
			buckets[bucketIndex].Max = max(buckets[bucketIndex].Max, value)
		}
	}

	var result []Bucket
	for _, b := range buckets {
		if b.Min != -9999 {
			result = append(result, b)
		}
	}

	return result
}
