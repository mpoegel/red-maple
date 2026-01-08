package redmaple

import (
	"log/slog"
	"net/http"
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
	data.First.StopName = resp[0].Stop.Name
	data.First.Destination = resp[0].Destination.Name
	data.First.HasIssues = len(alerts) > 0
	data.First.NextTrainIn = minutesUntilArrival(*resp[0].Arrival.Time, s.tz)
	data.First.TrainLine = string(subway.StopIdToLine(stops[0]))
	data.First.FurtherTrains = []int{
		minutesUntilArrival(*resp[1].Arrival.Time, s.tz),
		minutesUntilArrival(*resp[2].Arrival.Time, s.tz),
	}

	resp, alerts, err = s.subwayCli.GetTripsAtStop(r.Context(), stops[1])
	if err != nil {
		slog.Error("failed to get subway station status", "err", err, "stopID", stops[1])
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	data.Second.StopName = resp[0].Stop.Name
	data.Second.Destination = resp[0].Destination.Name
	data.Second.HasIssues = len(alerts) > 0
	data.Second.NextTrainIn = minutesUntilArrival(*resp[0].Arrival.Time, s.tz)
	data.Second.TrainLine = string(subway.StopIdToLine(stops[1]))
	data.Second.FurtherTrains = []int{
		minutesUntilArrival(*resp[1].Arrival.Time, s.tz),
		minutesUntilArrival(*resp[2].Arrival.Time, s.tz),
	}
	slog.Debug("prepared subway partial", "data", data)

	s.executeTemplate(w, "Subway", data)
}

func minutesUntilArrival(arrival int64, tz *time.Location) int {
	return int(time.Until(time.Unix(arrival, 0).In(tz)).Minutes())
}
