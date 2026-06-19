package main

import "testing"

// TestIntegration_SendCommandResponses_Ch23 replays the exact sequence used by
// SendCommand for channel 23 (RolloKueche / kitchen blind) and logs every
// Mobilegate response so we can see what is discarded in production code.
//
// Run with: go test -v -run TestIntegration_SendCommandResponses_Ch23 ./...
func TestIntegration_SendCommandResponses_Ch23(t *testing.T) {
	sender := newIntegrationSender(t)
	ch := 23

	signIn := NewSignInMessage(ch)
	signOut := NewSignOutMessage(ch)
	moveDown := NewBlindsMessage(ch, 100) // same as MoveDown()

	r1 := sender.Send(signIn, 1000)
	t.Logf("signIn #1 response:    %s", r1)

	r2 := sender.Send(signIn, 1000)
	t.Logf("signIn #2 response:    %s", r2)

	r3 := sender.Send(moveDown, 1000)
	t.Logf("ITEM_VALUE_SET response: %s", r3)

	r4 := sender.Send(signOut, 1000)
	t.Logf("signOut response:      %s", r4)
}

// TestIntegration_SendCommandResponses_Ch21 same test for channel 21 (RolloEssen)
// as a control — expected to be in a known-good state.
func TestIntegration_SendCommandResponses_Ch21(t *testing.T) {
	sender := newIntegrationSender(t)
	ch := 21

	signIn := NewSignInMessage(ch)
	signOut := NewSignOutMessage(ch)
	moveDown := NewBlindsMessage(ch, 100)

	r1 := sender.Send(signIn, 1000)
	t.Logf("signIn #1 response:    %s", r1)

	r2 := sender.Send(signIn, 1000)
	t.Logf("signIn #2 response:    %s", r2)

	r3 := sender.Send(moveDown, 1000)
	t.Logf("ITEM_VALUE_SET response: %s", r3)

	r4 := sender.Send(signOut, 1000)
	t.Logf("signOut response:      %s", r4)
}

// TestIntegration_SendCommandResponses_Ch17 office blind RolloArbeitszimmerStraße —
// currently known to be down. No position awareness (no-position blind).
func TestIntegration_SendCommandResponses_Ch17(t *testing.T) {
	sender := newIntegrationSender(t)
	ch := 17

	signIn := NewSignInMessage(ch)
	signOut := NewSignOutMessage(ch)
	moveDown := NewBlindsMessage(ch, 100)

	r1 := sender.Send(signIn, 1000)
	t.Logf("signIn #1 response:    %s", r1)

	r2 := sender.Send(signIn, 1000)
	t.Logf("signIn #2 response:    %s", r2)

	r3 := sender.Send(moveDown, 1000)
	t.Logf("ITEM_VALUE_SET response: %s", r3)

	r4 := sender.Send(signOut, 1000)
	t.Logf("signOut response:      %s", r4)
}

// TestIntegration_SendCommandResponses_Ch18 office blind RolloArbeitszimmerGarage —
// currently known to be down. Position-aware blind.
func TestIntegration_SendCommandResponses_Ch18(t *testing.T) {
	sender := newIntegrationSender(t)
	ch := 18

	signIn := NewSignInMessage(ch)
	signOut := NewSignOutMessage(ch)
	moveDown := NewBlindsMessage(ch, 100)

	r1 := sender.Send(signIn, 1000)
	t.Logf("signIn #1 response:    %s", r1)

	r2 := sender.Send(signIn, 1000)
	t.Logf("signIn #2 response:    %s", r2)

	r3 := sender.Send(moveDown, 1000)
	t.Logf("ITEM_VALUE_SET response: %s", r3)

	r4 := sender.Send(signOut, 1000)
	t.Logf("signOut response:      %s", r4)
}
