package weather

type WeatherData struct {
	Latitude       float64    `json:"lat"`
	Longitude      float64    `json:"lon"`
	Timezone       string     `json:"timezone"`
	TimezoneOffset int        `json:"timezone_offset"`
	Current        Current    `json:"current"`
	Minutely       []Minutely `json:"minutely"`
	Hourly         []Hourly   `json:"hourly"`
	Daily          []Daily    `json:"daily"`
	Alerts         []Alert    `json:"alerts"`
}

type Current struct {
	Timestamp     int           `json:"dt"`
	Sunrise       int           `json:"sunrise"`
	Sunset        int           `json:"sunset"`
	Temperature   float64       `json:"temp"`
	FeelsLike     float64       `json:"feels_like"`
	Pressure      int           `json:"pressure"`
	Humidity      int           `json:"humidity"`
	DewPoint      float64       `json:"dew_point"`
	CloudCover    int           `json:"clouds"`
	UVIndex       float64       `json:"uvi"`
	Visibility    int           `json:"visibility"`
	WindSpeed     float64       `json:"wind_speed"`
	WindGust      float64       `json:"wind_gust"`
	WindDirection int           `json:"wind_deg"`
	Rain          Precipitation `json:"rain"`
	Snow          Precipitation `json:"snow"`
	Description   []Description `json:"weather"`
}

type Minutely struct {
	Timestamp     int     `json:"dt"`
	Precipitation float64 `json:"precipitation"`
}

type Hourly struct {
	Timestamp                  int           `json:"dt"`
	Temperature                float64       `json:"temp"`
	FeelsLike                  float64       `json:"feels_like"`
	Pressure                   int           `json:"pressure"`
	Humidity                   int           `json:"humidity"`
	DewPoint                   float64       `json:"dew_point"`
	CloudCover                 int           `json:"clouds"`
	UVIndex                    float64       `json:"uvi"`
	Visibility                 int           `json:"visibility"`
	WindSpeed                  float64       `json:"wind_speed"`
	WindGust                   float64       `json:"wind_gust"`
	WindDirection              int           `json:"wind_deg"`
	ProbabilityOfPrecipitation float64       `json:"pop"`
	Rain                       Precipitation `json:"rain"`
	Snow                       Precipitation `json:"snow"`
	Description                []Description `json:"weather"`
}

type Daily struct {
	Timestamp   int     `json:"dt"`
	Sunrise     int     `json:"sunrise"`
	Sunset      int     `json:"sunset"`
	Moonrise    int     `json:"moonrise"`
	Moonset     int     `json:"moonset"`
	MoonPhase   float64 `json:"moon_phase"`
	Summary     string  `json:"summary"`
	Temperature struct {
		Morning float64 `json:"morn"`
		Day     float64 `json:"day"`
		Evening float64 `json:"evening"`
		Night   float64 `json:"night"`
		Min     float64 `json:"min"`
		Max     float64 `json:"max"`
	} `json:"temp"`
	FeelsLike struct {
		Morning float64 `json:"morn"`
		Day     float64 `json:"day"`
		Evening float64 `json:"evening"`
		Night   float64 `json:"night"`
	} `json:"feels_like"`
	Pressure                   int           `json:"pressure"`
	Humidity                   int           `json:"humidity"`
	DewPoint                   float64       `json:"dew_point"`
	CloudCover                 int           `json:"clouds"`
	UVIndex                    float64       `json:"uvi"`
	Visibility                 int           `json:"visibility"`
	WindSpeed                  float64       `json:"wind_speed"`
	WindGust                   float64       `json:"wind_gust"`
	WindDirection              int           `json:"wind_deg"`
	ProbabilityOfPrecipitation float64       `json:"pop"`
	Rain                       float64       `json:"rain"`
	Snow                       float64       `json:"snow"`
	Description                []Description `json:"weather"`
}

type Alert struct {
	Sender      string   `json:"sender_name"`
	Event       string   `json:"event"`
	Start       int      `json:"start"`
	End         int      `json:"end"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type Precipitation struct {
	MillimetersPerHour float64 `json:"1h"`
}

type Description struct {
	ID          int    `json:"id"`
	Main        string `json:"main"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}
