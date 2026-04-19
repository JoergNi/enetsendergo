package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// FakeHTTPClient is a test double for HTTPClient.
type FakeHTTPClient struct {
	Body       string
	StatusCode int
}

func (c *FakeHTTPClient) Get(url string) (*http.Response, error) {
	return &http.Response{
		StatusCode: c.StatusCode,
		Body:       io.NopCloser(strings.NewReader(c.Body)),
	}, nil
}

func newFakeClient(body string, statusCode int) HTTPClient {
	return &FakeHTTPClient{Body: body, StatusCode: statusCode}
}

func currentJSON(temp float64) string {
	return `{"current":{"temperature_2m":` + formatFloat(temp) + `}}`
}

func dailyHighJSON(temp float64) string {
	return `{"daily":{"temperature_2m_max":[` + formatFloat(temp) + `]}}`
}

func formatFloat(f float64) string {
	// Simple float formatting without importing fmt
	s := ""
	if f < 0 {
		s = "-"
		f = -f
	}
	whole := int(f)
	frac := int((f - float64(whole)) * 10)
	s += itoa(whole) + "." + itoa(frac)
	return s
}

func TestGetCurrentTemperature_ReturnsTemperatureFromResponse(t *testing.T) {
	svc := NewWeatherService(50.9, 7.1, newFakeClient(currentJSON(18.5), 200))
	temp := svc.GetCurrentTemperature()
	if temp == nil {
		t.Fatal("expected non-nil temperature")
	}
	if *temp != 18.5 {
		t.Errorf("expected 18.5, got %f", *temp)
	}
}

func TestGetCurrentTemperature_ReturnsNilOnHttpError(t *testing.T) {
	svc := NewWeatherService(50.9, 7.1, newFakeClient("error", 500))
	temp := svc.GetCurrentTemperature()
	if temp != nil {
		t.Error("expected nil on HTTP error")
	}
}

func TestGetCurrentTemperature_ReturnsNilOnInvalidJson(t *testing.T) {
	svc := NewWeatherService(50.9, 7.1, newFakeClient("not json", 200))
	temp := svc.GetCurrentTemperature()
	if temp != nil {
		t.Error("expected nil on invalid JSON")
	}
}

func TestGetDailyHighTemperature_ReturnsTemperatureFromResponse(t *testing.T) {
	svc := NewWeatherService(50.9, 7.1, newFakeClient(dailyHighJSON(28.3), 200))
	temp := svc.GetDailyHighTemperature()
	if temp == nil {
		t.Fatal("expected non-nil temperature")
	}
	if *temp != 28.3 {
		t.Errorf("expected 28.3, got %f", *temp)
	}
}

func TestGetDailyHighTemperature_ReturnsNilOnHttpError(t *testing.T) {
	svc := NewWeatherService(50.9, 7.1, newFakeClient("error", 500))
	temp := svc.GetDailyHighTemperature()
	if temp != nil {
		t.Error("expected nil on HTTP error")
	}
}

func TestIsHot_ReturnsTrueWhenDailyHighAboveThreshold(t *testing.T) {
	svc := NewWeatherService(50.9, 7.1, newFakeClient(dailyHighJSON(25.0), 200))
	if !svc.IsHot(24) {
		t.Error("expected IsHot=true")
	}
}

func TestIsHot_ReturnsFalseWhenDailyHighBelowThreshold(t *testing.T) {
	svc := NewWeatherService(50.9, 7.1, newFakeClient(dailyHighJSON(20.0), 200))
	if svc.IsHot(24) {
		t.Error("expected IsHot=false")
	}
}

func TestIsHot_ReturnsFalseWhenDailyHighExactlyAtThreshold(t *testing.T) {
	svc := NewWeatherService(50.9, 7.1, newFakeClient(dailyHighJSON(24.0), 200))
	if svc.IsHot(24) {
		t.Error("expected IsHot=false when exactly at threshold")
	}
}

func TestIsHot_ReturnsFalseOnFetchFailure(t *testing.T) {
	svc := NewWeatherService(50.9, 7.1, newFakeClient("error", 503))
	if svc.IsHot(24) {
		t.Error("expected IsHot=false on failure")
	}
}
