package main

import "testing"

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

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return
		}
	}
	t.Errorf("expected %q to contain %q", haystack, needle)
}
