package redmaple

import (
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
)

const (
	CentimetersToInches = 0.393701
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
		MoonPhaseIcon: MoonPhaseToIcon(int(weatherData.Daily[0].MoonPhase * 28)),
	}

	pollutionData, err := s.weatherCli.GetPollution(r.Context())
	if err != nil {
		slog.Error("failed to get pollution data", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	co := pollutionData.Data[0].Components.CarbonMonoxide
	no2 := pollutionData.Data[0].Components.NitrogenDioxide
	o3 := pollutionData.Data[0].Components.Ozone
	so2 := pollutionData.Data[0].Components.SulfurDioxide
	pm25 := pollutionData.Data[0].Components.Particulates2_5
	pm10 := pollutionData.Data[0].Components.Particulates10

	aqi := max(
		CalculateAQI(co/1.15/1000, co_con_breakpoints),
		CalculateAQI(o3/1.96/1000, o3_con_breakpoints),
		CalculateAQI(pm25, pm25_con_breakpoints),
		CalculateAQI(pm10, pm10_con_breakpoints),
		CalculateAQI(so2/2.62, so2_con_breakpoints),
		CalculateAQI(no2/1.88, no2_con_breakpoints),
	)
	if aqi <= 50 {
		partialData.AQI = 1
	} else if aqi <= 100 {
		partialData.AQI = 2
	} else if aqi <= 150 {
		partialData.AQI = 3
	} else if aqi <= 200 {
		partialData.AQI = 4
	} else {
		partialData.AQI = 5
	}
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

func (s *Server) HandleForecastFull(w http.ResponseWriter, r *http.Request) {
	weatherData, err := s.weatherCli.GetWeather(r.Context())
	if err != nil {
		slog.Error("failed to get weather data", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	data := api.WeatherFull{
		Hourly: []api.HourlyWeather{},
		Daily:  []api.DailyWeather{},
		Alerts: []api.WeatherAlert{},
	}

	for i, hour := range weatherData.Hourly {
		hourData := api.HourlyWeather{
			Stamp:       "",
			Icon:        hour.Description[0].ID,
			Temperature: int(hour.Temperature),
			Humidity:    hour.Humidity,
			WindSpeed:   int(hour.WindSpeed),
			RainChance:  int(hour.ProbabilityOfPrecipitation * 100),
		}
		t := time.Unix(int64(hour.Timestamp), 0).In(s.tz)
		hourData.Stamp = HourStamp(t)
		if hour.Rain.MillimetersPerHour > 0 {
			hourData.TotalRain = fmt.Sprintf("%.1f", hour.Rain.MillimetersPerHour*CentimetersToInches)
			hourData.RainOrSnowIcon = "wi-rain"
		} else if hour.Snow.MillimetersPerHour > 0 {
			hourData.TotalRain = fmt.Sprintf("%.1f", hour.Snow.MillimetersPerHour*CentimetersToInches)
			hourData.RainOrSnowIcon = "wi-snow"
		}
		if hour.Rain.MillimetersPerHour > 0 && hour.Snow.MillimetersPerHour > 0 {
			hourData.RainOrSnowIcon = "wi-rain-mix"
		}
		data.Hourly = append(data.Hourly, hourData)
		if i >= 12 {
			break
		}
	}

	for i, day := range weatherData.Daily {
		t := time.Unix(int64(day.Timestamp), 0).In(s.tz)
		dayData := api.DailyWeather{
			DayOfWeek:  strings.ToUpper(t.Weekday().String())[:3],
			Icon:       day.Description[0].ID,
			HighTemp:   int(day.Temperature.Max),
			LowTemp:    int(day.Temperature.Min),
			Humidity:   day.Humidity,
			RainChance: int(day.ProbabilityOfPrecipitation * 100),
		}
		if day.Rain > 0 {
			dayData.TotalRain = fmt.Sprintf("%.1f", day.Rain*CentimetersToInches)
			dayData.RainOrSnowIcon = "wi-rain"
		} else if day.Snow > 0 {
			dayData.TotalRain = fmt.Sprintf("%.1f", day.Snow*CentimetersToInches)
			dayData.RainOrSnowIcon = "wi-snow"
		}
		if day.Rain > 0 && day.Snow > 0 {
			dayData.RainOrSnowIcon = "wi-rain-mix"
		}
		data.Daily = append(data.Daily, dayData)
		if i >= 5 {
			break
		}
	}

	for _, alert := range weatherData.Alerts {
		start := time.Unix(int64(alert.Start), 0).In(s.tz)
		end := time.Unix(int64(alert.End), 0).In(s.tz)
		data.Alerts = append(data.Alerts, api.WeatherAlert{
			Title: alert.Event,
			Stamp: fmt.Sprintf("%s %d %s to %s %d %s",
				strings.ToUpper(start.Month().String()[:3]),
				start.Day(),
				HourStamp(start),
				strings.ToUpper(end.Month().String()[:3]),
				end.Day(),
				HourStamp(end),
			),
			Description: alert.Description,
		})
	}

	s.executeTemplate(w, "FullForecast", data)
}

func (s *Server) HandleAqiPartial(w http.ResponseWriter, r *http.Request) {
	pollutionData, err := s.weatherCli.GetPollution(r.Context())
	if err != nil {
		slog.Error("failed to get weather data", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	co := pollutionData.Data[0].Components.CarbonMonoxide
	no2 := pollutionData.Data[0].Components.NitrogenDioxide
	o3 := pollutionData.Data[0].Components.Ozone
	so2 := pollutionData.Data[0].Components.SulfurDioxide
	pm25 := pollutionData.Data[0].Components.Particulates2_5
	pm10 := pollutionData.Data[0].Components.Particulates10

	data := api.AqiPartial{
		CarbonMonoxide:  CalculateAQI(co/1.15/1000, co_con_breakpoints),
		Ozone:           CalculateAQI(o3/1.96/1000, o3_con_breakpoints),
		Particulates2_5: CalculateAQI(pm25, pm25_con_breakpoints),
		Particulates10:  CalculateAQI(pm10, pm10_con_breakpoints),
		SulfurDioxide:   CalculateAQI(so2/2.62, so2_con_breakpoints),
		NitrogenDioxide: CalculateAQI(no2/1.88, no2_con_breakpoints),
	}
	data.AQI = max(
		data.CarbonMonoxide,
		data.Ozone,
		data.Particulates2_5,
		data.Particulates10,
		data.SulfurDioxide,
		data.NitrogenDioxide,
	)

	slog.Info("pollution", "data", data, "raw", pollutionData.Data[0].Components)
	s.executeTemplate(w, "AQI", data)
}

func (s *Server) HandleSunrises(w http.ResponseWriter, r *http.Request) {
	weatherData, err := s.weatherCli.GetWeather(r.Context())
	if err != nil {
		slog.Error("failed to get weather data", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	data := api.SunriseForecast{
		Forecast: []api.SunForecast{},
	}
	for i, day := range weatherData.Daily {
		sunrise := time.Unix(int64(day.Sunrise), 0).In(s.tz)
		sunset := time.Unix(int64(day.Sunset), 0).In(s.tz)
		data.Forecast = append(data.Forecast, api.SunForecast{
			DayOfWeek: strings.ToUpper(sunrise.Weekday().String())[:3],
			Sunrise:   fmt.Sprintf("%d:%02d", sunrise.Hour(), sunrise.Minute()),
			Sunset:    fmt.Sprintf("%d:%02d", sunset.Hour(), sunset.Minute()),
			MoonIcon:  MoonPhaseToIcon(int(day.MoonPhase * 28)),
			UVIndex:   int(math.Round(day.UVIndex)),
		})
		if i >= 4 {
			break
		}
	}
	s.executeTemplate(w, "SunriseForecast", data)
}

var (
	// https://document.airnow.gov/technical-assistance-document-for-the-reporting-of-daily-air-quailty.pdf
	aqi_breakpoints      = []float64{0.0, 50, 51, 100, 101, 150, 151, 200, 201, 300, 301}
	o3_con_breakpoints   = []float64{0.0, 0.054, 0.055, 0.070, 0.071, 0.085, 0.086, 0.105, 0.106, 0.200, 0.201}
	pm25_con_breakpoints = []float64{0.0, 9.0, 9.1, 35.4, 35.5, 55.4, 55.5, 125.4, 125.5, 225.4, 225.5}
	pm10_con_breakpoints = []float64{0.0, 54, 55, 154, 155, 254, 255, 354, 355, 424, 425}
	co_con_breakpoints   = []float64{0.0, 4.4, 4.5, 9.4, 9.5, 12.4, 12.5, 15.4, 15.5, 30.4, 30.5}
	so2_con_breakpoints  = []float64{0.0, 35, 36, 75, 76, 185, 186, 304, 305, 604, 605}
	no2_con_breakpoints  = []float64{0.0, 53, 54, 100, 101, 360, 361, 649, 650, 1249, 1250}
)

func CalculateAQI(concentration float64, conBreakpoints []float64) int {
	i := 1
	for i < len(conBreakpoints)-1 {
		if concentration < conBreakpoints[i] {
			aqi := (aqi_breakpoints[i]-aqi_breakpoints[i-1])/
				(conBreakpoints[i]-conBreakpoints[i-1])*
				(concentration-conBreakpoints[i-1]) +
				aqi_breakpoints[i-1]
			return int(math.Round(aqi))
		}
		i += 2
	}
	return 0
}

func HourStamp(t time.Time) string {
	if t.Hour() == 0 {
		return "12 AM"
	} else if t.Hour() < 12 {
		return fmt.Sprintf("%d AM", t.Hour())
	} else if t.Hour() == 12 {
		return "12 PM"
	} else {
		return fmt.Sprintf("%d PM", t.Hour()-12)
	}
}

func MoonPhaseToIcon(i int) string {
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
