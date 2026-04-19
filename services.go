package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// GetConfig reads configuration with priority: /data/options.json → env var → default.
func GetConfig(jsonKey, envVar, defaultValue string) string {
	const optionsFile = "/data/options.json"
	if data, err := os.ReadFile(optionsFile); err == nil {
		var opts map[string]interface{}
		if json.Unmarshal(data, &opts) == nil {
			if v, ok := opts[jsonKey]; ok {
				return fmt.Sprintf("%v", v)
			}
		}
	}
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return defaultValue
}

// WeatherService fetches temperature from Open-Meteo.
type WeatherService struct {
	Latitude  float64
	Longitude float64
	Client    HTTPClient
}

// HTTPClient interface for testability.
type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

func NewWeatherService(lat, lon float64, client HTTPClient) *WeatherService {
	if client == nil {
		client = http.DefaultClient
	}
	return &WeatherService{Latitude: lat, Longitude: lon, Client: client}
}

func (w *WeatherService) GetCurrentTemperature() *float64 {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current=temperature_2m",
		w.Latitude, w.Longitude)
	return w.fetchTemperature(url, func(data map[string]interface{}) *float64 {
		current, ok := data["current"].(map[string]interface{})
		if !ok {
			return nil
		}
		temp, ok := current["temperature_2m"].(float64)
		if !ok {
			return nil
		}
		return &temp
	})
}

func (w *WeatherService) GetDailyHighTemperature() *float64 {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&daily=temperature_2m_max&forecast_days=1",
		w.Latitude, w.Longitude)
	return w.fetchTemperature(url, func(data map[string]interface{}) *float64 {
		daily, ok := data["daily"].(map[string]interface{})
		if !ok {
			return nil
		}
		maxArr, ok := daily["temperature_2m_max"].([]interface{})
		if !ok || len(maxArr) == 0 {
			return nil
		}
		temp, ok := maxArr[0].(float64)
		if !ok {
			return nil
		}
		return &temp
	})
}

func (w *WeatherService) IsHot(thresholdCelsius float64) bool {
	temp := w.GetDailyHighTemperature()
	if temp != nil {
		LogNormal(fmt.Sprintf("Today's high: %.1f°C", *temp))
	}
	return temp != nil && *temp > thresholdCelsius
}

func (w *WeatherService) fetchTemperature(url string, extract func(map[string]interface{}) *float64) *float64 {
	resp, err := w.Client.Get(url)
	if err != nil {
		LogNormal(fmt.Sprintf("Weather fetch failed: %s", err.Error()))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		LogNormal(fmt.Sprintf("Weather fetch failed: HTTP %d", resp.StatusCode))
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		LogNormal(fmt.Sprintf("Weather fetch failed: %s", err.Error()))
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		LogNormal(fmt.Sprintf("Weather fetch failed: %s", err.Error()))
		return nil
	}

	return extract(data)
}

// FilterJobLog returns log entries on or after the cutoff date.
func FilterJobLog(entries []string, cutoffDate string) []string {
	var result []string
	for _, l := range entries {
		if len(l) >= 10 && strings.Compare(l[:10], cutoffDate) >= 0 {
			result = append(result, l)
		}
	}
	return result
}
