package api

type DatetimePartial struct {
	Timestamp string
	AMOrPM    string
	Seconds   string
	Date      string
}

type CitibikePartial struct {
	First  CitibikeStation
	Second CitibikeStation
}

type CitibikeStation struct {
	Name       string
	TotalBikes int
	NumBikes   int
	NumEbikes  int
}

type SubwayPartial struct {
	First  SubwayUpdate
	Second SubwayUpdate
}

type SubwayUpdate struct {
	TrainLine     string
	StopName      string
	NextTrainIn   int
	Destination   string
	HasIssues     bool
	FurtherTrains []int
}

type WeatherPartial struct {
	CurrentWeatherIcon int
	TodayHighTemp      int
	TodayLowTemp       int
	TodayRainChance    int
	Forecast           []WeatherForecast
}

type WeatherForecast struct {
	DayOfWeek   string
	WeatherIcon int
	RainChance  int
	HighTemp    int
	LowTemp     int
}

type IndoorPartial struct {
	IntegerTemp          int
	FractionalTemp       int
	IsTempTrendingUp     bool
	IntegerHumidity      int
	FractionalHumidity   int
	IsHumidityTrendingUp bool
	HumidityLevel        int
}

type OutdoorPartial IndoorPartial

type SunrisePartial struct {
	SunriseTime   string
	SunsetTime    string
	AQI           int
	MoonPhaseIcon string
}

type SundialPartial struct {
	Rotation float64
	Color    string
}

type WeatherFull struct {
	Hourly []HourlyWeather
	Daily  []DailyWeather
	Alerts []WeatherAlert
}

type HourlyWeather struct {
	Stamp          string
	Icon           int
	Temperature    int
	Humidity       int
	WindSpeed      int
	RainChance     int
	TotalRain      string
	RainOrSnowIcon string
}

type DailyWeather struct {
	DayOfWeek      string
	Icon           int
	HighTemp       int
	LowTemp        int
	Humidity       int
	RainChance     int
	TotalRain      string
	RainOrSnowIcon string
}

type WeatherAlert struct {
	Title       string
	Stamp       string
	Description string
}

type AqiPartial struct {
	AQI              int
	CarbonMonoxide   int
	NitrogenMonoxide int
	NitrogenDioxide  int
	Ozone            int
	SulfurDioxide    int
	Particulates2_5  int
	Particulates10   int
	Ammonia          int
}
