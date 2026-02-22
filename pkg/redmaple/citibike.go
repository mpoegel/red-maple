package redmaple

import (
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
	citibike "github.com/mpoegel/red-maple/pkg/citibike"
)

func (s *Server) HandleCitibike(w http.ResponseWriter, r *http.Request) {
	data := api.CitibikePartial{
		Stations: []api.CitibikeStation{},
	}
	for i := 0; i < max(len(s.config.CitibikeStations), 2); i++ {
		name := s.config.CitibikeStations[i]
		numClassics, numEbikes, err := s.citibike.GetNumBikesAtStation(r.Context(), name)
		if err != nil {
			slog.Error("failed to get citibike station status", "err", err, "station", name)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		data.Stations = append(data.Stations, api.CitibikeStation{
			Name:       name,
			TotalBikes: numClassics + numEbikes,
			NumBikes:   numClassics,
			NumEbikes:  numEbikes,
		})
	}

	s.executeTemplate(w, "Citibike", data)
}

func (s *Server) HandleBikesFull(w http.ResponseWriter, r *http.Request) {
	s.executeTemplate(w, "BikesFull", struct{}{})
}

func (s *Server) HandleCitiBikeHistory(w http.ResponseWriter, r *http.Request) {
	if s.importer == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	station, err := url.QueryUnescape(r.URL.Query().Get("station"))
	if station == "" || err != nil {
		if len(s.config.CitibikeStations) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		station = s.config.CitibikeStations[0]
	}

	days := 1
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			days = parsed
		}
	}

	bikeKind := r.URL.Query().Get("kind")
	if bikeKind == "" {
		bikeKind = "all"
	}

	slog.Debug("citibike history request", "station", station, "days", days, "bikeKind", bikeKind)

	var history []citibike.HistoricalBikeCount

	switch days {
	case 7:
		history, err = s.citibike.GetHistoricalBikeCounts7Days(r.Context(), s.importer, station)
	case 30:
		history, err = s.citibike.GetHistoricalBikeCounts30Days(r.Context(), s.importer, station)
	default:
		history, err = s.citibike.GetHistoricalBikeCounts24Hours(r.Context(), s.importer, station)
		days = 1
	}

	if err != nil {
		slog.Error("failed to get historical bike counts", "err", err, "station", station, "days", days)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	slog.Debug("citibike history raw data", "count", len(history))
	if len(history) > 0 {
		slog.Debug("citibike history time range",
			"first", history[0].Stamp,
			"last", history[len(history)-1].Stamp,
			"firstClassics", history[0].Classics,
			"firstEbikes", history[0].Ebikes)
	}

	buckets := CompactToBuckets(history, days, bikeKind)
	slog.Debug("citibike history", "buckets", buckets)

	var data []api.GraphPoint
	var minY, maxY int
	for _, b := range buckets {
		minY = min(minY, b.Min)
		maxY = max(maxY, b.Max)
	}
	// maxY++

	yDiff := maxY - minY
	for _, b := range buckets {
		bottom := int(200.0 / float64(yDiff) * float64(b.Min))
		data = append(data, api.GraphPoint{
			Min:   bottom,
			Max:   int(200.0/float64(yDiff)*float64(b.Max)) - bottom,
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

	slog.Debug("citibike history graph data",
		"dataPoints", len(data),
		"minY", minY,
		"maxY", maxY,
		"startTime", startTimeStr,
		"endTime", endTimeStr)

	stations := make([]api.CitibikeStationSelection, len(s.config.CitibikeStations))
	for i, name := range s.config.CitibikeStations {
		stations[i] = api.CitibikeStationSelection{
			Name:        name,
			UrlSafeName: url.QueryEscape(name),
			Days:        days,
			BikeKind:    bikeKind,
			IsSelected:  name == station,
		}
	}

	dataPayload := api.CitibikeHistory{
		Days:      days,
		Station:   url.QueryEscape(station),
		BikeKind:  bikeKind,
		MaxY:      maxY,
		MinY:      minY,
		Data:      data,
		StartTime: startTimeStr,
		EndTime:   endTimeStr,
		Stations:  stations,
	}

	s.executeTemplate(w, "CitibikeHistory", dataPayload)
}

type Bucket struct {
	Min int
	Max int
}

func CompactToBuckets(history []citibike.HistoricalBikeCount, days int, bikeKind string) []Bucket {
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
		buckets[i].Min = -1
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

		var value int
		switch bikeKind {
		case "classic":
			value = h.Classics
		case "electric":
			value = h.Ebikes
		default:
			value = h.Classics + h.Ebikes
		}

		if buckets[bucketIndex].Min == -1 {
			buckets[bucketIndex].Min = value
			buckets[bucketIndex].Max = value
		} else {
			if value < buckets[bucketIndex].Min {
				buckets[bucketIndex].Min = value
			}
			if value > buckets[bucketIndex].Max {
				buckets[bucketIndex].Max = value
			}
		}
	}

	var result []Bucket
	for _, b := range buckets {
		if b.Min != -1 {
			result = append(result, b)
		}
	}

	return result
}
