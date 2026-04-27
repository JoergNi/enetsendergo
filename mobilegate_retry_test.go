package main

import "testing"

func TestSendCommandRetry_WhenFirstAttemptSucceeds_RecordsCommand(t *testing.T) {
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

func TestSendCommandRetry_WhenSecondAttemptSucceeds_DoesNotInvokeFailureCallback(t *testing.T) {
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

func TestSendCommandRetry_WhenThirdAttemptSucceeds_DoesNotInvokeFailureCallback(t *testing.T) {
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

func TestSendCommandRetry_WhenAllAttemptsFail_InvokesFailureCallback(t *testing.T) {
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

func TestSendCommandRetry_WhenCallbackIsNil_DoesNotPanic(t *testing.T) {
	origDelay := RetryDelayMs
	RetryDelayMs = 0
	defer func() { RetryDelayMs = origDelay }()

	origCb := OnCommandFailed
	OnCommandFailed = nil
	defer func() { OnCommandFailed = origCb }()

	fake := NewFakeMobilegate()
	fake.FailCount = 3
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	blind.MoveDown()
}
