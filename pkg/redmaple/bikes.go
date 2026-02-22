package redmaple

import (
	"log/slog"
	"net/http"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
	nycdata "github.com/mpoegel/red-maple/pkg/nycdata"
)

func (s *Server) HandleBikeBridges(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	endTime := time.Now()

	dateRange := r.URL.Query().Get("range")
	switch dateRange {
	case "year-to-date":
		startTime = time.Date(endTime.Year(), time.January, 0, 0, 0, 0, 0, s.tz)
	default:
		dateRange = "monthly"
		startTime = time.Date(endTime.Year(), endTime.Month()-1, 0, 0, 0, 0, 0, s.tz)
	}

	countTotals := [4]int{0, 0, 0, 0}
	countIDs := [4]nycdata.CounterID{
		nycdata.QueensboroBridgeCounterID,
		nycdata.WilliamsburgBridgeCounterID,
		nycdata.ManhattanBridgeCounterID,
		nycdata.BrooklynBridgeCounterID,
	}
	for i, cid := range countIDs {
		counts, err := s.nycClient.GetBicycleCounts(r.Context(), cid, nycdata.WithDateRange(startTime, endTime))
		if err != nil {
			slog.Error("failed to get bike bridge counts", "err", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		for _, c := range counts {
			countTotals[i] += c.Counts
		}
	}
	data := api.BikeBridges{
		Queensboro:   countTotals[0],
		Williamsburg: countTotals[1],
		Manhattan:    countTotals[2],
		Brooklyn:     countTotals[3],
		Range:        dateRange,
	}

	slog.Debug("prepared bike bridges", "data", data)
	s.executeTemplate(w, "BikeBridges", data)
}
