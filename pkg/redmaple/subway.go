package redmaple

import (
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
	subway "github.com/mpoegel/red-maple/pkg/subway"
)

func (s *Server) HandleSubway(w http.ResponseWriter, r *http.Request) {
	data := api.SubwayPartial{}
	stops := strings.Split(s.config.SubwayStops, ",")

	// TODO cache response and count down
	resp, alerts, err := s.subwayCli.GetTripsAtStop(r.Context(), stops[0])
	if err != nil {
		slog.Error("failed to get subway station status", "err", err, "stopID", stops[0])
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if len(resp) == 0 {
		slog.Warn("no trips found", "stop", stops[0])
	} else {
		data.First.StopName = resp[0].Stop.Name
		data.First.Destination = resp[0].Destination.Name
		data.First.NextTrainIn = MinutesUntilArrival(*resp[0].Arrival.Time, s.tz)
		data.First.FurtherTrains = []int{
			MinutesUntilArrival(*resp[1].Arrival.Time, s.tz),
			MinutesUntilArrival(*resp[2].Arrival.Time, s.tz),
		}
	}
	data.First.TrainLine = string(subway.StopIdToLine(stops[0]))
	data.First.HasIssues = len(alerts) > 0

	resp, alerts, err = s.subwayCli.GetTripsAtStop(r.Context(), stops[1])
	if err != nil {
		slog.Error("failed to get subway station status", "err", err, "stopID", stops[1])
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if len(resp) == 0 {
		slog.Warn("no trips found", "stop", stops[1])
	} else {
		data.Second.StopName = resp[0].Stop.Name
		data.Second.Destination = resp[0].Destination.Name
		data.Second.NextTrainIn = MinutesUntilArrival(*resp[0].Arrival.Time, s.tz)
		data.Second.FurtherTrains = []int{
			MinutesUntilArrival(*resp[1].Arrival.Time, s.tz),
			MinutesUntilArrival(*resp[2].Arrival.Time, s.tz),
		}
	}
	data.Second.TrainLine = string(subway.StopIdToLine(stops[1]))
	data.Second.HasIssues = len(alerts) > 0

	slog.Debug("prepared subway partial", "data", data)

	s.executeTemplate(w, "Subway", data)
}

func (s *Server) HandleSubwayFull(w http.ResponseWriter, r *http.Request) {
	lineParam := r.URL.Query().Get("line")
	if lineParam == "" {
		lineParam = "L"
	}
	s.executeTemplate(w, "SubwayFull", api.SubwayFull{Line: lineParam})
}

func (s *Server) HandleSubwayLine(w http.ResponseWriter, r *http.Request) {
	lineParam := r.URL.Query().Get("line")
	if lineParam == "" {
		lineParam = "L"
	}
	line := subway.ParseTrainLine(lineParam)
	if line == subway.UnknownTrain {
		line = subway.LTrain
	}
	allStops, err := s.subwayCli.GetStopsOnLine(r.Context(), line)
	if err != nil {
		slog.Error("failed to get stops", "err", err, "train", line)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	trains, alerts, err := s.subwayCli.GetTrains(r.Context(), line)
	if err != nil {
		slog.Error("failed to get trains", "err", err, "train", line)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	data := api.SubwayLine{
		Segments: []api.SubwaySegment{},
		Alerts:   []string{},
	}

	stations := []subway.SubwayStop{}
	for _, stop := range allStops {
		if stop.LocationType == subway.RootStationType {
			stations = append(stations, stop)
		}
	}
	slices.SortFunc(stations, func(a, b subway.SubwayStop) int {
		idA, _ := strconv.Atoi(a.ID[1:])
		idB, _ := strconv.Atoi(b.ID[1:])
		if idA < idB {
			return -1
		} else if idA > idB {
			return 1
		} else {
			return 0
		}
	})

	nextStopHasTrainApproaching := false
	for _, station := range stations {
		approachingSegment := api.SubwaySegment{}
		stationSegment := api.SubwaySegment{
			IsStation:      true,
			StationName:    station.Name,
			NoServiceNorth: (station.AreTrainsStopping & subway.TrainsStoppingNorth) == 0,
			NoServiceSouth: (station.AreTrainsStopping & subway.TrainsStoppingSouth) == 0,
		}

		if nextStopHasTrainApproaching {
			approachingSegment.HasTrainNorth = true
			nextStopHasTrainApproaching = false
		}

		// check for a train at or approaching this station
		for _, train := range trains {
			if strings.HasPrefix(train.NextStop.ID, station.ID) {
				if strings.HasSuffix(train.NextStop.ID, "N") {
					if train.IsAtStop {
						stationSegment.HasTrainNorth = true
					} else {
						nextStopHasTrainApproaching = true
					}
				} else if strings.HasSuffix(train.NextStop.ID, "S") {
					if train.IsAtStop {
						stationSegment.HasTrainSouth = true
					} else {
						approachingSegment.HasTrainSouth = true
					}
				}
			}
		}

		data.Segments = append(data.Segments, approachingSegment, stationSegment)
	}
	data.Segments = append(data.Segments, api.SubwaySegment{
		HasTrainNorth: nextStopHasTrainApproaching,
	})

	for _, alert := range alerts {
		data.Alerts = append(data.Alerts, alert.DescriptionText.String())
	}

	s.executeTemplate(w, "SubwayLine", data)
}

func MinutesUntilArrival(arrival int64, tz *time.Location) int {
	return int(time.Until(time.Unix(arrival, 0).In(tz)).Minutes())
}
