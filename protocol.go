package main

import (
	"encoding/json"
	"fmt"
)

// EnetCommandMessage is the base message for all Mobilegate commands.
type EnetCommandMessage struct {
	CMD       string `json:"CMD"`
	Protocol  string `json:"PROTOCOL"`
	Timestamp string `json:"TIMESTAMP"`
	Items     []int  `json:"ITEMS,omitempty"`
}

func NewCommandMessage(command string, channel int) EnetCommandMessage {
	return EnetCommandMessage{
		CMD:       command,
		Protocol:  "0.03",
		Timestamp: "1421948265",
		Items:     []int{channel},
	}
}

func (m EnetCommandMessage) String() string {
	b, _ := json.Marshal(m)
	return string(b) + "\r\n\r\n"
}

// BlindState represents a blind value in a SET command.
type BlindState struct {
	Number int    `json:"NUMBER"`
	State  string `json:"STATE"`
	Value  int    `json:"VALUE"`
}

// SwitchState represents an on/off value in a SET command.
type SwitchState struct {
	Number int    `json:"NUMBER"`
	State  string `json:"STATE"`
}

// DimmerState represents a dimmer value in a SET command.
type DimmerState struct {
	Number int    `json:"NUMBER"`
	State  string `json:"STATE"`
	Value  int    `json:"VALUE"`
}

type valueSetMessage struct {
	CMD       string      `json:"CMD"`
	Protocol  string      `json:"PROTOCOL"`
	Timestamp string      `json:"TIMESTAMP,omitempty"`
	Values    interface{} `json:"VALUES"`
}

func (m valueSetMessage) String() string {
	b, _ := json.Marshal(m)
	return string(b) + "\r\n\r\n"
}

func NewBlindsMessage(channel, value int) string {
	msg := valueSetMessage{
		CMD:       "ITEM_VALUE_SET",
		Protocol:  "0.03",
		Timestamp: "1421948266",
		Values:    []BlindState{{Number: channel, State: "VALUE_BLINDS", Value: value}},
	}
	return msg.String()
}

func NewOnOffMessage(channel int, on bool) string {
	state := "OFF"
	if on {
		state = "ON"
	}
	msg := valueSetMessage{
		CMD:       "ITEM_VALUE_SET",
		Protocol:  "0.03",
		Timestamp: "1421948266",
		Values:    []SwitchState{{Number: channel, State: state}},
	}
	return msg.String()
}

func NewDimmerMessage(channel, value int) string {
	msg := valueSetMessage{
		CMD:      "ITEM_VALUE_SET",
		Protocol: "0.03",
		Values:   []DimmerState{{Number: channel, State: "ON", Value: value}},
	}
	return msg.String()
}

func NewSignInMessage(channel int) string {
	return NewCommandMessage("ITEM_VALUE_SIGN_IN_REQ", channel).String()
}

func NewSignOutMessage(channel int) string {
	return NewCommandMessage("ITEM_VALUE_SIGN_OUT_REQ", channel).String()
}

func NewVersionRequest() string {
	msg := EnetCommandMessage{
		CMD:       "VERSION_REQ",
		Protocol:  "0.03",
		Timestamp: "1421948265",
	}
	b, _ := json.Marshal(msg)
	return string(b) + "\r\n\r\n"
}

func NewGetChannelInfoAllRequest() string {
	msg := EnetCommandMessage{
		CMD:       "GET_CHANNEL_INFO_ALL_REQ",
		Protocol:  "0.03",
		Timestamp: "1421948265",
	}
	b, _ := json.Marshal(msg)
	return string(b) + "\r\n\r\n"
}

// ParseSignInResponse extracts VALUE and STATE from a Mobilegate SIGN_IN response.
func ParseSignInResponse(response string) *ThingState {
	if response == "" {
		return nil
	}
	return parseValueState(response)
}

func parseValueState(s string) *ThingState {
	// Look for "VALUE":"<number>" and "STATE":"<string>"
	valueIdx := findField(s, `"VALUE":"`)
	stateIdx := findField(s, `"STATE":"`)

	if valueIdx < 0 || stateIdx < 0 {
		return nil
	}

	valueStr := extractQuotedValue(s, valueIdx)
	stateStr := extractQuotedValue(s, stateIdx)

	if valueStr == "" || stateStr == "" {
		return nil
	}

	var value int
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return nil
	}

	return &ThingState{
		Value: value,
		State: stateStr,
	}
}

func findField(s, field string) int {
	for i := 0; i <= len(s)-len(field); i++ {
		if s[i:i+len(field)] == field {
			return i + len(field)
		}
	}
	return -1
}

func extractQuotedValue(s string, start int) string {
	end := start
	for end < len(s) && s[end] != '"' {
		end++
	}
	if end >= len(s) {
		return ""
	}
	return s[start:end]
}
