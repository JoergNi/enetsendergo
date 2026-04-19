package main

import (
	"net"
	"testing"
)

// Tests that exercise real SocketMobilegateSender TCP error paths.
// Uses a loopback port with no listener (connection refused).

func getFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func TestSend_ConnectionRefused_ReturnsNil(t *testing.T) {
	port := getFreePort(t)
	sender := NewSocketSender("127.0.0.1", port)
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: sender}

	state := blind.GetState()

	if state != nil {
		t.Error("GetState() must return nil when TCP connection is refused")
	}
}

func TestSendCommand_ConnectionRefused_CallsOnCommandFailedAfterRetries(t *testing.T) {
	origDelay := RetryDelayMs
	RetryDelayMs = 0
	defer func() { RetryDelayMs = origDelay }()

	var failMsg string
	origCb := OnCommandFailed
	OnCommandFailed = func(msg string) { failMsg = msg }
	defer func() { OnCommandFailed = origCb }()

	port := getFreePort(t)
	sender := NewSocketSender("127.0.0.1", port)
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: sender}

	blind.MoveDown()

	if failMsg == "" {
		t.Fatal("OnCommandFailed must be invoked after all retry attempts fail")
	}
	assertContains(t, failMsg, "ch18")
	assertContains(t, failMsg, "test")
}

func TestMobilegateLogic_ResponseWithVersionRes_ClassifiedAsOk(t *testing.T) {
	fake := NewFakeMobilegate()
	fake.Responses = append(fake.Responses, `{"CMD":"VERSION_RES","FIRMWARE":"0.91","PROTOCOL":"0.03"}`+"\r\n\r\n")
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	response := blind.SendRequest("irrelevant", 2000)

	var result string
	if contains(response, "VERSION_RES") {
		result = "ok"
	} else {
		result = "down"
	}
	if result != "ok" {
		t.Errorf("expected 'ok', got '%s'", result)
	}
}

func TestMobilegateLogic_EmptyResponse_ClassifiedAsDown(t *testing.T) {
	fake := NewFakeMobilegate()
	// empty queue → returns ""
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	response := blind.SendRequest("irrelevant", 2000)

	var result string
	if contains(response, "VERSION_RES") {
		result = "ok"
	} else {
		result = "down"
	}
	if result != "down" {
		t.Errorf("expected 'down', got '%s'", result)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
