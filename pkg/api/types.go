package api

type DatetimePartial struct {
	Timestamp string
	AMOrPM    string
	Seconds   string
	Date      string
}

type CitibikePartial struct {
	Stations []CitibikeStation
}

type CitibikeStation struct {
	Name       string
	TotalBikes int
	NumBikes   int
	NumEbikes  int
}

type CitibikeHistory struct {
	Days      int
	Station   string
	BikeKind  string
	MaxY      int
	MinY      int
	Data      []GraphPoint
	StartTime string
	EndTime   string
	Stations  []CitibikeStationSelection
}

type GraphPoint struct {
	Min   int
	Max   int
	Width float64
}

type CitibikeStationSelection struct {
	Name        string
	UrlSafeName string
	Days        int
	BikeKind    string
	IsSelected  bool
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

type IndoorHistory struct {
	Days      int
	DataName  string
	MaxY      int
	MinY      int
	Data      []GraphPoint
	StartTime string
	EndTime   string
}

type OutdoorHistory IndoorHistory

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

type SunriseForecast struct {
	Forecast []SunForecast
}

type SunForecast struct {
	DayOfWeek string
	Sunrise   string
	Sunset    string
	MoonIcon  string
	UVIndex   int
}

type SubwayFull struct {
	Line string
}

type SubwayLine struct {
	Segments []SubwaySegment
	Alerts   []string
}

type SubwaySegment struct {
	IsStation      bool
	StationName    string
	HasTrainNorth  bool
	HasTrainSouth  bool
	NoServiceNorth bool
	NoServiceSouth bool
}

type BikeBridges struct {
	Queensboro   int
	Williamsburg int
	Manhattan    int
	Brooklyn     int
	Range        string
}
