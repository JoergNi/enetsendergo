package main

import (
	"fmt"
	"net"
	"time"
)

// ThingState represents the current state of a device.
type ThingState struct {
	Value int    // 0-100 for position-aware, -1 for non-position-aware
	State string // "OFF"/"ALL_OFF" = up, "ON"/"ALL_ON" = down
}

func (s *ThingState) IsPositionAware() bool {
	return s.Value >= 0 && s.Value <= 100
}

func (s *ThingState) IsUp() bool {
	return s.State == "OFF" || s.State == "ALL_OFF"
}

// MobilegateSender abstracts the TCP transport to the Mobilegate.
type MobilegateSender interface {
	Send(message string, receiveTimeoutMs int) string
	SendCommand(commandMessage string, channel int, thingName string) error
}

// ThingType identifies the type of device.
type ThingType string

const (
	ThingTypeBlind   ThingType = "blind"
	ThingTypeSwitch  ThingType = "switch"
	ThingTypeDimmer  ThingType = "dimmer"
)

// Thing represents a single eNet device.
type Thing struct {
	Name      string
	Channel   int
	Type      ThingType
	Sender    MobilegateSender
}

// RetryDelayMs controls delay between retries. Overridable for tests.
var RetryDelayMs = 1000

// OnCommandFailed is called when all retry attempts are exhausted.
var OnCommandFailed func(msg string)

func (t *Thing) GetState() *ThingState {
	signIn := NewSignInMessage(t.Channel)
	response := t.Sender.Send(signIn, 500)
	return ParseSignInResponse(response)
}

func (t *Thing) SendRequest(message string, receiveTimeoutMs int) string {
	return t.Sender.Send(message, receiveTimeoutMs)
}

func (t *Thing) sendCommandMessage(commandMessage string) {
	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := t.Sender.SendCommand(commandMessage, t.Channel, t.Name)
		if err == nil {
			return
		}
		LogNormal(fmt.Sprintf("SocketException on ch%d (%s), attempt %d/%d: %s",
			t.Channel, t.Name, attempt, maxAttempts, err.Error()))
		if attempt < maxAttempts {
			time.Sleep(time.Duration(RetryDelayMs) * time.Millisecond)
		} else {
			msg := fmt.Sprintf("ch%d (%s) command failed after %d attempts", t.Channel, t.Name, maxAttempts)
			LogNormal("SendCommand failed after " + fmt.Sprintf("%d", maxAttempts) + " attempts: " + fmt.Sprintf("ch%d (%s)", t.Channel, t.Name))
			if OnCommandFailed != nil {
				OnCommandFailed(msg)
			}
		}
	}
}

// Blind commands
func (t *Thing) MoveDown()                { t.sendCommandMessage(NewBlindsMessage(t.Channel, 100)) }
func (t *Thing) MoveUp()                  { t.sendCommandMessage(NewBlindsMessage(t.Channel, 0)) }
func (t *Thing) MoveHalf()                { t.sendCommandMessage(NewBlindsMessage(t.Channel, 50)) }
func (t *Thing) MoveThreeQuarters()       { t.sendCommandMessage(NewBlindsMessage(t.Channel, 75)) }
func (t *Thing) MoveTo(value int)         { t.sendCommandMessage(NewBlindsMessage(t.Channel, value)) }

// Switch commands
func (t *Thing) TurnOn()                  { t.sendCommandMessage(NewOnOffMessage(t.Channel, true)) }
func (t *Thing) TurnOff()                 { t.sendCommandMessage(NewOnOffMessage(t.Channel, false)) }

// Dimmer commands
func (t *Thing) SetBrightness(value int)  { t.sendCommandMessage(NewDimmerMessage(t.Channel, value)) }
func (t *Thing) DimmerTurnOn()            { t.sendCommandMessage(NewDimmerMessage(t.Channel, 100)) }
func (t *Thing) DimmerTurnOff()           { t.sendCommandMessage(NewOnOffMessage(t.Channel, false)) }

// SocketMobilegateSender is the real TCP implementation.
type SocketMobilegateSender struct {
	IP   string
	Port int
}

func NewSocketSender(ip string, port int) *SocketMobilegateSender {
	return &SocketMobilegateSender{IP: ip, Port: port}
}

func (s *SocketMobilegateSender) connect() (net.Conn, error) {
	addr := fmt.Sprintf("%s:%d", s.IP, s.Port)
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *SocketMobilegateSender) Send(message string, receiveTimeoutMs int) string {
	conn, err := s.connect()
	if err != nil {
		return ""
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(time.Duration(receiveTimeoutMs) * time.Millisecond))
	_, err = conn.Write([]byte(message))
	if err != nil {
		return ""
	}

	buf := make([]byte, 65536)
	var result []byte
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return string(result)
}

func (s *SocketMobilegateSender) SendCommand(commandMessage string, channel int, thingName string) error {
	conn, err := s.connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 1024)

	signIn := NewSignInMessage(channel)

	conn.Write([]byte(signIn))
	conn.Read(buf)
	conn.Write([]byte(signIn))
	conn.Read(buf)
	conn.Write([]byte(commandMessage))
	conn.Read(buf)

	signOut := NewSignOutMessage(channel)
	conn.Write([]byte(signOut))
	conn.Read(buf)

	return nil
}
