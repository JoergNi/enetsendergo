package main

import "testing"

func TestBlindMoveDown_SendsFullyClosedValue(t *testing.T) {
	fake := NewFakeMobilegate()
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	blind.MoveDown()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"VALUE":100`)
	assertContains(t, fake.CommandMessages[0], `"STATE":"VALUE_BLINDS"`)
}

func TestBlindMoveUp_SendsFullyOpenValue(t *testing.T) {
	fake := NewFakeMobilegate()
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	blind.MoveUp()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"VALUE":0`)
}

func TestBlindMoveTo_SendsRequestedPosition(t *testing.T) {
	fake := NewFakeMobilegate()
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	blind.MoveTo(35)

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"VALUE":35`)
}

func TestBlindMoveHalf_SendsMidpointPosition(t *testing.T) {
	fake := NewFakeMobilegate()
	blind := &Thing{Name: "test", Channel: 18, Type: ThingTypeBlind, Sender: fake}

	blind.MoveHalf()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"VALUE":50`)
}

func TestSwitchTurnOn_SendsOnState(t *testing.T) {
	fake := NewFakeMobilegate()
	sw := &Thing{Name: "test", Channel: 16, Type: ThingTypeSwitch, Sender: fake}

	sw.TurnOn()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"STATE":"ON"`)
}

func TestSwitchTurnOff_SendsOffState(t *testing.T) {
	fake := NewFakeMobilegate()
	sw := &Thing{Name: "test", Channel: 16, Type: ThingTypeSwitch, Sender: fake}

	sw.TurnOff()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"STATE":"OFF"`)
}

func TestDimmerSetBrightness_SendsRequestedValue(t *testing.T) {
	fake := NewFakeMobilegate()
	light := &Thing{Name: "test", Channel: 27, Type: ThingTypeDimmer, Sender: fake}

	light.SetBrightness(75)

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"VALUE":75`)
	assertContains(t, fake.CommandMessages[0], `"STATE":"ON"`)
}

func TestDimmerTurnOff_SendsOffState(t *testing.T) {
	fake := NewFakeMobilegate()
	light := &Thing{Name: "test", Channel: 27, Type: ThingTypeDimmer, Sender: fake}

	light.DimmerTurnOff()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"STATE":"OFF"`)
}

func TestDimmerTurnOn_SendsFullBrightnessValue(t *testing.T) {
	fake := NewFakeMobilegate()
	light := &Thing{Name: "test", Channel: 27, Type: ThingTypeDimmer, Sender: fake}

	light.DimmerTurnOn()

	if len(fake.CommandMessages) != 1 {
		t.Fatalf("expected 1 command, got %d", len(fake.CommandMessages))
	}
	assertContains(t, fake.CommandMessages[0], `"VALUE":100`)
}
