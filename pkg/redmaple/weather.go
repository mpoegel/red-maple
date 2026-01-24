package redmaple

import (
	"fmt"
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

func (s *Server) HandleSunrise(w http.ResponseWriter, r *http.Request) {
	weatherData, err := s.weatherCli.GetWeather(r.Context())
	if err != nil {
		slog.Error("failed to get weather data", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	sunriseTime := time.Unix(int64(weatherData.Current.Sunrise), 0).In(s.tz)
	sunsetTime := time.Unix(int64(weatherData.Current.Sunset), 0).In(s.tz)
	partialData := api.SunrisePartial{
		SunriseTime:   fmt.Sprintf("%d:%02d", sunriseTime.Hour(), sunriseTime.Minute()),
		SunsetTime:    fmt.Sprintf("%d:%02d", sunsetTime.Hour()-12, sunsetTime.Minute()),
		MoonPhaseIcon: moonPhaseToIcon(int(weatherData.Daily[0].MoonPhase * 28)),
	}

	pollutionData, err := s.weatherCli.GetPollution(r.Context())
	if err != nil {
		slog.Error("failed to get pollution data", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	partialData.AQI = pollutionData.Data[0].Main.AQI

	slog.Debug("prepared sunrise partial", "data", partialData)

	s.executeTemplate(w, "Sunrise", partialData)
}

func (s *Server) HandleWeatherFull(w http.ResponseWriter, r *http.Request) {
	s.executeTemplate(w, "WeatherFull", struct{}{})
}

func (s *Server) HandleSunriseFull(w http.ResponseWriter, r *http.Request) {
	s.executeTemplate(w, "SunriseFull", struct{}{})
}

func (s *Server) HandleSundial(w http.ResponseWriter, r *http.Request) {
	weatherData, err := s.weatherCli.GetWeather(r.Context())
	if err != nil {
		slog.Error("failed to get weather data", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	sunrise := time.Unix(int64(weatherData.Current.Sunrise), 0)
	sunset := time.Unix(int64(weatherData.Current.Sunset), 0)
	tomorrowSunrise := time.Unix(int64(weatherData.Daily[1].Sunrise), 0)
	// estimate yesterday's sunset
	yesterdaySunset := sunset.AddDate(0, 0, -1)
	now := time.Now()

	data := api.SundialPartial{}
	if now.After(sunrise) && now.Before(sunset) {
		// sun is up
		// progress / daylight = x / 180
		// x = 180 * progress / daylight
		daylight := sunset.Sub(sunrise).Seconds()
		data.Rotation = 180.0 * now.Sub(sunrise).Seconds() / daylight
		data.Color = "#00C6FF"
	} else if now.After(sunset) && now.Hour() >= sunset.Hour() {
		// sun has set, same day
		moonlight := tomorrowSunrise.Sub(sunset).Seconds()
		data.Rotation = (180.0 * now.Sub(sunset).Seconds() / moonlight) + 180
		data.Color = "#303030"
	} else {
		// sun has not yet risen, next day
		moonlight := sunrise.Sub(yesterdaySunset).Seconds()
		data.Rotation = (180.0 * now.Sub(yesterdaySunset).Seconds() / moonlight) + 180
		data.Color = "#303030"
	}
	// midday is 0 deg, so offset by -90def
	data.Rotation -= 90.0

	if data.Rotation >= 85 && data.Rotation < 90 {
		data.Color = "#FF5A36"
	} else if data.Rotation >= 265 && data.Rotation < 270 {
		data.Color = "#FF5A36"
	}

	s.executeTemplate(w, "Sundial", data)
}

func moonPhaseToIcon(i int) string {
	switch i % 28 {
	default:
		return "wi-moon-new"
	case 1:
		return "wi-moon-waxing-crescent-1"
	case 2:
		return "wi-moon-waxing-crescent-2"
	case 3:
		return "wi-moon-waxing-crescent-3"
	case 4:
		return "wi-moon-waxing-crescent-4"
	case 5:
		return "wi-moon-waxing-crescent-5"
	case 6:
		return "wi-moon-waxing-crescent-6"
	case 7:
		return "wi-moon-first-quarter"
	case 8:
		return "wi-moon-waxing-gibbous-1"
	case 9:
		return "wi-moon-waxing-gibbous-2"
	case 10:
		return "wi-moon-waxing-gibbous-3"
	case 11:
		return "wi-moon-waxing-gibbous-4"
	case 12:
		return "wi-moon-waxing-gibbous-5"
	case 13:
		return "wi-moon-waxing-gibbous-6"
	case 14:
		return "wi-moon-full"
	case 15:
		return "wi-moon-waning-gibbous-1"
	case 16:
		return "wi-moon-waning-gibbous-2"
	case 17:
		return "wi-moon-waning-gibbous-3"
	case 18:
		return "wi-moon-waning-gibbous-4"
	case 19:
		return "wi-moon-waning-gibbous-5"
	case 20:
		return "wi-moon-waning-gibbous-6"
	case 21:
		return "wi-moon-third-quarter"
	case 22:
		return "wi-moon-waning-crescent-1"
	case 23:
		return "wi-moon-waning-crescent-2"
	case 24:
		return "wi-moon-waning-crescent-3"
	case 25:
		return "wi-moon-waning-crescent-4"
	case 26:
		return "wi-moon-waning-crescent-5"
	case 27:
		return "wi-moon-waning-crescent-6"
	}
}
