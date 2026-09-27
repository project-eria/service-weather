package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	eria "github.com/project-eria/eria-core"
	zlog "github.com/rs/zerolog/log"
)

var config = struct {
	Lat  float64 `yaml:"lat" required:"true"`
	Long float64 `yaml:"long" required:"true"`
	Rate uint    `yaml:"rate" default:"900"` // seconds; Open-Meteo refreshes "current" every 15 min
}{}

// Open-Meteo field -> Thing property. Units: °C, %, hPa, m/s, °, UV index.
var fields = map[string]string{
	"temperature_2m":       "temperature",
	"relative_humidity_2m": "humidity",
	"pressure_msl":         "pressure",
	"wind_speed_10m":       "windStrength",
	"wind_direction_10m":   "windAngle",
	"uv_index":             "uvIndex",
}

// Properties typed as integers in the eria-core models
var integers = map[string]bool{"humidity": true, "pressure": true, "windAngle": true}

var httpClient = &http.Client{Timeout: 20 * time.Second}

func main() {
	defer func() {
		zlog.Info().Msg("[main] Stopped")
	}()

	eria.Init("ERIA Weather service", &config)

	td, _ := eria.NewThingDescription(
		"eria:service:weather:1",
		eria.AppVersion,
		"Weather",
		"Current weather from open-meteo.com",
		[]string{"TemperatureSensor", "HygrometerSensor", "BarometerSensor", "WindgaugeSensor", "UVSensor"},
	)
	// Forecast data: no sensor to calibrate
	delete(td.Actions, "calibrateTemperature")
	for _, prop := range fields {
		td.Properties[prop].ReadOnly = true
	}

	producer := eria.Producer("")
	thing, _ := producer.AddThing("", td)
	for _, prop := range fields {
		producer.PropertyUseDefaultHandlers(thing, prop)
	}

	go func() {
		ticker := time.NewTicker(time.Duration(config.Rate) * time.Second)
		for ; ; <-ticker.C {
			values, err := fetch()
			if err != nil {
				// Keep the last values; the next tick retries
				zlog.Warn().Err(err).Msg("[main] Open-Meteo update failed")
				continue
			}
			zlog.Trace().Interface("values", values).Msg("[main] Open-Meteo update")
			for prop, v := range values {
				producer.SetPropertyValue(thing, prop, v)
			}
		}
	}()

	eria.Start("")
}

func fetch() (map[string]interface{}, error) {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&wind_speed_unit=ms"+
		"&current=temperature_2m,relative_humidity_2m,pressure_msl,wind_speed_10m,wind_direction_10m,uv_index",
		config.Lat, config.Long)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	return parse(body)
}

// parse maps the "current" block to property values; null fields are skipped
func parse(body []byte) (map[string]interface{}, error) {
	var data struct {
		Current map[string]json.RawMessage `json:"current"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	values := map[string]interface{}{}
	for field, prop := range fields {
		var v *float64
		if err := json.Unmarshal(data.Current[field], &v); err != nil || v == nil {
			continue
		}
		if integers[prop] {
			values[prop] = int(math.Round(*v))
		} else {
			values[prop] = *v
		}
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("no current data in response")
	}
	return values, nil
}
