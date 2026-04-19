package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// Integration tests — require a live Mobilegate at 192.168.178.34:9050.
// Run with: go test -v -run TestIntegration ./...

func newIntegrationSender(t *testing.T) MobilegateSender {
	t.Helper()
	host := GetConfig("enet_host", "ENET_HOST", "192.168.178.34")
	port := 9050
	return NewSocketSender(host, port)
}

func TestIntegration_VersionReq_ReturnsFirmwareHardwareEnet(t *testing.T) {
	sender := newIntegrationSender(t)
	probe := &Thing{Name: "probe", Channel: 18, Type: ThingTypeBlind, Sender: sender}

	response := probe.SendRequest(NewVersionRequest(), 2000)
	t.Logf("VERSION_RES: %s", response)

	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		t.Fatal("empty response — is the Mobilegate reachable at 192.168.178.34:9050?")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		t.Fatalf("could not parse response as JSON: %v\nraw: %s", err, trimmed)
	}

	if cmd, _ := parsed["CMD"].(string); cmd != "VERSION_RES" {
		t.Errorf("expected CMD=VERSION_RES, got %q", cmd)
	}

	firmware, _ := parsed["FIRMWARE"].(string)
	if firmware == "" {
		t.Error("FIRMWARE should be non-empty")
	}

	hardware, _ := parsed["HARDWARE"].(string)
	if hardware == "" {
		t.Error("HARDWARE should be non-empty")
	}

	enet, _ := parsed["ENET"].(string)
	if enet == "" {
		t.Error("ENET should be non-empty")
	}

	t.Logf("FIRMWARE=%s  HARDWARE=%s  ENET=%s", firmware, hardware, enet)
}

func TestIntegration_GetChannelInfoAll_Returns40DeviceTypes(t *testing.T) {
	sender := newIntegrationSender(t)
	probe := &Thing{Name: "probe", Channel: 18, Type: ThingTypeBlind, Sender: sender}

	response := probe.SendRequest(NewGetChannelInfoAllRequest(), 2000)
	t.Logf("GET_CHANNEL_INFO_ALL_RES: %s", response)

	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		t.Fatal("empty response — is the Mobilegate reachable at 192.168.178.34:9050?")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		t.Fatalf("could not parse response as JSON: %v\nraw: %s", err, trimmed)
	}

	if cmd, _ := parsed["CMD"].(string); cmd != "GET_CHANNEL_INFO_ALL_RES" {
		t.Errorf("expected CMD=GET_CHANNEL_INFO_ALL_RES, got %q", cmd)
	}

	devArr, ok := parsed["DEVICES"].([]interface{})
	if !ok {
		t.Fatal("DEVICES array should be present")
	}
	if len(devArr) != 40 {
		t.Errorf("expected 40 channel entries, got %d", len(devArr))
	}

	// Log non-zero channels
	var nonZero []string
	for i, v := range devArr {
		if dt, ok := v.(float64); ok && dt != 0 {
			nonZero = append(nonZero, "ch"+itoa(i)+"=type"+itoa(int(dt)))
		}
	}
	t.Logf("Non-zero channels: %s", strings.Join(nonZero, ", "))
}

/*
func TestIntegration_MoveToPositionAndVerifyState(t *testing.T) {
	sender := newIntegrationSender(t)
	blind := &Thing{Name: "RolloArbeitszimmerGarage", Channel: 18, Type: ThingTypeBlind, Sender: sender}

	blind.MoveUp()
	stateAfterUp := waitUntilStopped(t, blind, 40)
	t.Logf("After MoveUp: Value=%d, State=%s", stateAfterUp.Value, stateAfterUp.State)
	if stateAfterUp.Value != 0 {
		t.Errorf("expected blind fully up (0), got %d", stateAfterUp.Value)
	}

	blind.MoveTo(25)
	stateAfterMove := waitUntilStopped(t, blind, 40)
	t.Logf("After MoveTo(25): Value=%d, State=%s", stateAfterMove.Value, stateAfterMove.State)
	if stateAfterMove.Value != 25 {
		t.Errorf("expected blind at 25, got %d", stateAfterMove.Value)
	}
}

func waitUntilStopped(t *testing.T, blind *Thing, timeoutSeconds int) *ThingState {
	t.Helper()
	polls := timeoutSeconds / 2
	var last *ThingState
	stableCount := 0
	for i := 0; i < polls; i++ {
		time.Sleep(2 * time.Second)
		current := blind.GetState()
		if last != nil && current != nil && current.Value == last.Value {
			stableCount++
			if stableCount >= 2 {
				return current
			}
		} else {
			stableCount = 0
		}
		last = current
	}
	return last
}
*/
