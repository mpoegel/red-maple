package redmaple

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
)

func (s *Server) HandleWeather(w http.ResponseWriter, r *http.Request) {
	weatherData, err := s.weatherCli.GetWeather(r.Context())
	if err != nil {
		slog.Error("failed to get weather data", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	partialData := api.WeatherPartial{}
	partialData.CurrentWeatherIcon = weatherData.Current.Description[0].ID
	partialData.TodayHighTemp = int(weatherData.Daily[0].Temperature.Max)
	partialData.TodayLowTemp = int(weatherData.Daily[0].Temperature.Min)
	partialData.TodayRainChance = int(weatherData.Daily[0].ProbabilityOfPrecipitation * 100)
	partialData.Forecast = []api.WeatherForecast{}
	for i, daily := range weatherData.Daily {
		if i == 0 {
			// skip today
			continue
		}
		if i > 3 {
			// 3 day forecast only
			break
		}
		t := time.Unix(int64(daily.Timestamp), 0)
		partialData.Forecast = append(partialData.Forecast, api.WeatherForecast{
			DayOfWeek:   strings.ToUpper(t.Weekday().String())[:3],
			WeatherIcon: daily.Description[0].ID,
			RainChance:  int(daily.ProbabilityOfPrecipitation * 100),
			HighTemp:    int(daily.Temperature.Max),
			LowTemp:     int(daily.Temperature.Min),
		})
	}
	slog.Debug("prepared weather partial", "data", partialData)

	s.executeTemplate(w, "Forecast", partialData)
}
