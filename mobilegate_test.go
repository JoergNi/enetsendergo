package main

import (
	"testing"
)

// FakeMobilegate is a test double for MobilegateSender.
type FakeMobilegate struct {
	Responses       []string
	responseIdx     int
	CommandMessages []string
	FailCount       int
	callCount       int
}

func NewFakeMobilegate() *FakeMobilegate {
	return &FakeMobilegate{}
}

func (f *FakeMobilegate) Send(message string, receiveTimeoutMs int) string {
	if f.responseIdx < len(f.Responses) {
		resp := f.Responses[f.responseIdx]
		f.responseIdx++
		return resp
	}
	return ""
}

func (f *FakeMobilegate) SendCommand(commandMessage string, channel int, thingName string) error {
	f.callCount++
	if f.callCount <= f.FailCount {
		return &fakeSocketError{msg: "connection refused"}
	}
	f.CommandMessages = append(f.CommandMessages, commandMessage)
	return nil
}

type fakeSocketError struct {
	msg string
}

func (e *fakeSocketError) Error() string { return e.msg }

// Helper response builders (matching C# test helpers)
func SignInResponse(channel, value int, state string) string {
	return `{"CMD":"ITEM_VALUE_SIGN_IN_RES","PROTOCOL":"0.03","ITEMS":[` +
		itoa(channel) + `],"TIMESTAMP":"1421948265"}` + "\r\n" +
		`{"CMD":"ITEM_UPDATE_IND","NUMBER":` + itoa(channel) +
		`,"STATE":"` + state + `","VALUE":"` + itoa(value) +
		`","PROTOCOL":"0.03"}` + "\r\n\r\n"
}

func EmptyResponse() string { return "" }

func UndefinedResponse(channel int) string {
	return `{"CMD":"ITEM_VALUE_SIGN_IN_RES","PROTOCOL":"0.03","ITEMS":[` +
		itoa(channel) + `],"TIMESTAMP":"1421948265"}` + "\r\n" +
		`{"CMD":"ITEM_UPDATE_IND","NUMBER":` + itoa(channel) +
		`,"STATE":"UNDEFINED","PROTOCOL":"0.03"}` + "\r\n\r\n"
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

// ── GetState tests ──────────────────────────────────────────────

func TestGetState_ParsesPositionAndState(t *testing.T) {
	fake := NewFakeMobilegate()
	fake.Responses = append(fake.Responses, SignInResponse(18, 75, "ON"))
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	state := blind.GetState()

	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.Value != 75 {
		t.Errorf("expected Value=75, got %d", state.Value)
	}
	if state.State != "ON" {
		t.Errorf("expected State=ON, got %s", state.State)
	}
}

func TestGetState_FullyUp_IsUpTrue(t *testing.T) {
	fake := NewFakeMobilegate()
	fake.Responses = append(fake.Responses, SignInResponse(18, 0, "OFF"))
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	state := blind.GetState()

	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.Value != 0 {
		t.Errorf("expected Value=0, got %d", state.Value)
	}
	if !state.IsUp() {
		t.Error("expected IsUp=true")
	}
}

func TestGetState_ReturnsNilOnEmptyResponse(t *testing.T) {
	fake := NewFakeMobilegate()
	fake.Responses = append(fake.Responses, EmptyResponse())
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	if blind.GetState() != nil {
		t.Error("expected nil state on empty response")
	}
}

func TestGetState_ReturnsNilWhenNoValueField(t *testing.T) {
	fake := NewFakeMobilegate()
	fake.Responses = append(fake.Responses, UndefinedResponse(22))
	blind := &Thing{Name: "test", Channel: 22, Type: ThingTypeBlind, Sender: fake}

	if blind.GetState() != nil {
		t.Error("expected nil state for UNDEFINED response")
	}
}

func TestGetState_NegativeValuePassedThrough(t *testing.T) {
	fake := NewFakeMobilegate()
	fake.Responses = append(fake.Responses, SignInResponse(17, -1, "ON"))
	blind := &Thing{Name: "test", Channel: 17, Type: ThingTypeBlind, Sender: fake}

	state := blind.GetState()

	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.Value != -1 {
		t.Errorf("expected Value=-1, got %d", state.Value)
	}
	if state.IsPositionAware() {
		t.Error("expected IsPositionAware=false")
	}
}

// ── Blind command tests ────────────────────────────────────────

func TestBlind_MoveDown_SendsValue100(t *testing.T) {
	fake := NewFakeMobilegate()
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	blind.MoveDown()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"VALUE":100`)
	assertContains(t, fake.CommandMessages[0], `"STATE":"VALUE_BLINDS"`)
}

func TestBlind_MoveUp_SendsValue0(t *testing.T) {
	fake := NewFakeMobilegate()
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	blind.MoveUp()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"VALUE":0`)
}

func TestBlind_MoveTo_SendsCorrectValue(t *testing.T) {
	fake := NewFakeMobilegate()
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	blind.MoveTo(35)

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"VALUE":35`)
}

func TestBlind_MoveHalf_SendsValue50(t *testing.T) {
	fake := NewFakeMobilegate()
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	blind.MoveHalf()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"VALUE":50`)
}

// ── Switch command tests ───────────────────────────────────────

func TestSwitch_TurnOn_SendsStateOn(t *testing.T) {
	fake := NewFakeMobilegate()
	sw := &Thing{Name: "test", Channel: 16, Type: ThingTypeSwitch, Sender: fake}

	sw.TurnOn()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"STATE":"ON"`)
}

func TestSwitch_TurnOff_SendsStateOff(t *testing.T) {
	fake := NewFakeMobilegate()
	sw := &Thing{Name: "test", Channel: 16, Type: ThingTypeSwitch, Sender: fake}

	sw.TurnOff()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"STATE":"OFF"`)
}

// ── DimmableLight command tests ────────────────────────────────

func TestDimmableLight_SetBrightness_SendsCorrectValue(t *testing.T) {
	fake := NewFakeMobilegate()
	light := &Thing{Name: "test", Channel: 27, Type: ThingTypeDimmer, Sender: fake}

	light.SetBrightness(75)

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"VALUE":75`)
	assertContains(t, fake.CommandMessages[0], `"STATE":"ON"`)
}

func TestDimmableLight_TurnOff_SendsStateOff(t *testing.T) {
	fake := NewFakeMobilegate()
	light := &Thing{Name: "test", Channel: 27, Type: ThingTypeDimmer, Sender: fake}

	light.DimmerTurnOff()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"STATE":"OFF"`)
}

func TestDimmableLight_TurnOn_SendsValue100(t *testing.T) {
	fake := NewFakeMobilegate()
	light := &Thing{Name: "test", Channel: 27, Type: ThingTypeDimmer, Sender: fake}

	light.DimmerTurnOn()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"VALUE":100`)
}

// ── Retry tests ────────────────────────────────────────────────

func TestRetry_SucceedsFirstAttempt_CommandRecorded(t *testing.T) {
	origDelay := RetryDelayMs
	RetryDelayMs = 0
	defer func() { RetryDelayMs = origDelay }()

	fake := NewFakeMobilegate()
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	blind.MoveDown()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
}

func TestRetry_FailsOnceThenSucceeds_CommandRecorded_NoCallback(t *testing.T) {
	origDelay := RetryDelayMs
	RetryDelayMs = 0
	defer func() { RetryDelayMs = origDelay }()

	var failMsg string
	origCb := OnCommandFailed
	OnCommandFailed = func(msg string) { failMsg = msg }
	defer func() { OnCommandFailed = origCb }()

	fake := NewFakeMobilegate()
	fake.FailCount = 1
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	blind.MoveDown()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	if failMsg != "" {
		t.Errorf("OnCommandFailed should not have been called, got: %s", failMsg)
	}
}

func TestRetry_FailsTwiceThenSucceeds_CommandRecorded_NoCallback(t *testing.T) {
	origDelay := RetryDelayMs
	RetryDelayMs = 0
	defer func() { RetryDelayMs = origDelay }()

	var failMsg string
	origCb := OnCommandFailed
	OnCommandFailed = func(msg string) { failMsg = msg }
	defer func() { OnCommandFailed = origCb }()

	fake := NewFakeMobilegate()
	fake.FailCount = 2
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	blind.MoveDown()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	if failMsg != "" {
		t.Errorf("OnCommandFailed should not have been called, got: %s", failMsg)
	}
}

func TestRetry_FailsAllAttempts_CallsOnCommandFailed(t *testing.T) {
	origDelay := RetryDelayMs
	RetryDelayMs = 0
	defer func() { RetryDelayMs = origDelay }()

	var failMsg string
	origCb := OnCommandFailed
	OnCommandFailed = func(msg string) { failMsg = msg }
	defer func() { OnCommandFailed = origCb }()

	fake := NewFakeMobilegate()
	fake.FailCount = 3
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	blind.MoveDown()

	if len(fake.CommandMessages) != 0 {
		t.Fatalf("expected 0 commands, got %d", len(fake.CommandMessages))
	}
	if failMsg == "" {
		t.Fatal("expected OnCommandFailed to be called")
	}
	assertContains(t, failMsg, "ch18")
	assertContains(t, failMsg, "test")
}

func TestRetry_FailsAllAttempts_NilCallback_NoException(t *testing.T) {
	origDelay := RetryDelayMs
	RetryDelayMs = 0
	defer func() { RetryDelayMs = origDelay }()

	origCb := OnCommandFailed
	OnCommandFailed = nil
	defer func() { OnCommandFailed = origCb }()

	fake := NewFakeMobilegate()
	fake.FailCount = 3
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	// Must not panic
	blind.MoveDown()
}

// ── Helper ─────────────────────────────────────────────────────

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return
		}
	}
	t.Errorf("expected %q to contain %q", haystack, needle)
}
