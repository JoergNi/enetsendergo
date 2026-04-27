package main

import "testing"

func TestGetState_WhenResponseContainsValue_ParsesPositionAndState(t *testing.T) {
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

func TestGetState_WhenBlindIsFullyUp_IsUpReturnsTrue(t *testing.T) {
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

func TestGetState_WhenResponseIsEmpty_ReturnsNil(t *testing.T) {
	fake := NewFakeMobilegate()
	fake.Responses = append(fake.Responses, EmptyResponse())
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	if blind.GetState() != nil {
		t.Error("expected nil state on empty response")
	}
}

func TestGetState_WhenValueFieldIsMissing_ReturnsNil(t *testing.T) {
	fake := NewFakeMobilegate()
	fake.Responses = append(fake.Responses, UndefinedResponse(22))
	blind := &Thing{Name: "test", Channel: 22, Type: ThingTypeBlind, Sender: fake}

	if blind.GetState() != nil {
		t.Error("expected nil state for UNDEFINED response")
	}
}

func TestGetState_WhenValueIsNegative_PreservesRawValue(t *testing.T) {
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
