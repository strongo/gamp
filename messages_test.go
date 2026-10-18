package gamp

import (
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"
)

const testClientID = "bd4e1566-9626-4bf8-8f03-e54aa678c23f"

type stepFailWriter struct {
	failAt int
	step   int
}

func (w *stepFailWriter) Write(p []byte) (n int, err error) {
	w.step++
	if w.step == w.failAt {
		return 0, errors.New("simulated step write error")
	}
	return len(p), nil
}

func TestCommon(t *testing.T) {
	c := Common{}
	c.SetTrackingID("UA-999")
	if c.GetTrackingID() != "UA-999" {
		t.Errorf("expected UA-999, got %s", c.GetTrackingID())
	}
}

func TestNewEvent(t *testing.T) {
	e := NewEvent("Ca1", "Ac1", Common{ClientID: testClientID})
	if e.Category != "Ca1" {
		t.Error("Category is not set correctly")
	}
	if e.Action != "Ac1" {
		t.Error("Action is not set correctly")
	}

	// Validate errors
	if err := (Event{Action: "a"}).Validate(); err == nil || err.Error() != "missing required parameter: Category" {
		t.Errorf("expected missing Category, got %v", err)
	}
	if err := (Event{Category: "c"}).Validate(); err == nil || err.Error() != "missing required parameter: Action" {
		t.Errorf("expected missing Action, got %v", err)
	}

	// Panic on invalid event in NewEventWithLabel
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on missing category")
			}
		}()
		NewEventWithLabel("", "a", "l", Common{})
	}()
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on missing action")
			}
		}()
		NewEventWithLabel("c", "", "l", Common{})
	}()
}

func TestTiming(t *testing.T) {
	timing := NewTiming(2500 * time.Millisecond)
	if timing.ServerResponseTime != 2500 {
		t.Errorf("expected 2500, got %d", timing.ServerResponseTime)
	}
	timing.Common = Common{
		TrackingID:   "T1",
		ClientID:     "C1",
		UserID:       "U1",
		UserLanguage: "en",
		UserAgent:    "Agent",
	}

	s := timing.String()
	expected := "v=1&tid=T1&t=timing&cid=C1&uid=U1&ul=en&ua=Agent&srt=2500"
	if s != expected {
		t.Errorf("\nExpected: %s\n     Got: %s", expected, s)
	}

	// Zero server response time
	timingZero := Timing{Common: Common{TrackingID: "T1"}}
	if sZero := timingZero.String(); sZero != "v=1&tid=T1&t=timing" {
		t.Errorf("expected 'v=1&tid=T1&t=timing', got %s", sZero)
	}
}

func TestException(t *testing.T) {
	exFatal := NewException("fatal crash", true)
	exFatal.Common = Common{TrackingID: "T1"}
	if !exFatal.IsFatal || exFatal.Description != "fatal crash" {
		t.Errorf("unexpected exFatal: %+v", exFatal)
	}
	if s := exFatal.String(); s != "v=1&tid=T1&t=exception&exd=fatal+crash&exf=1" {
		t.Errorf("unexpected fatal string: %s", s)
	}

	exNonFatal := NewException("minor glitch", false)
	exNonFatal.Common = Common{TrackingID: "T1"}
	if s := exNonFatal.String(); s != "v=1&tid=T1&t=exception&exd=minor+glitch" {
		t.Errorf("unexpected non-fatal string: %s", s)
	}
}

func TestMessageString(t *testing.T) {
	common := Common{
		TrackingID:   "Track1",
		ClientID:     "Client2",
		UserID:       "User3",
		UserLanguage: "en-US",
		UserAgent:    "Go-Test-Agent",
	}
	for _, item := range []struct {
		Message
		Expected string
	}{
		{
			&Event{
				Common:   common,
				Category: "Category3",
				Action:   "Action4",
				Label:    "Label5",
				Value:    42,
			},
			"v=1&tid=Track1&t=event&cid=Client2&uid=User3&ul=en-US&ua=Go-Test-Agent&ec=Category3&ea=Action4&ev=42&el=Label5",
		},
		{
			NewPageviewWithDocumentHost("localhost:8080", "/path1/path2", "Title #1"),
			fmt.Sprintf("v=1&t=pageview&dh=%v&dp=%v&dt=%v", "localhost:8080", url.QueryEscape("/path1/path2"), url.QueryEscape("Title #1")),
		},
	} {
		actual := item.String()
		if actual != item.Expected {
			t.Errorf("\nExpected: %v\n     Got: %v", item.Expected, actual)
		}
	}

	// messageToString panic
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when message.Write fails")
			}
		}()
		_ = messageToString(&errMessage{})
	}()
}

func TestNewPageview(t *testing.T) {
	host := "localhost:8080"
	path := "/path1/path2"
	title := "Title #1"
	m := NewPageviewWithDocumentHost(host, path, title)
	if m.DocumentHost != host {
		t.Error("DocumentHost is not set correctly")
	}
	if m.DocumentPath != path {
		t.Error("DocumentPath is not set correctly")
	}
	if m.DocumentTitle != title {
		t.Error("DocumentTitle is not set correctly")
	}
}

func TestWriteErrors(t *testing.T) {
	fullCommon := Common{
		TrackingID:   "T1",
		ClientID:     "C1",
		UserID:       "U1",
		UserLanguage: "en",
		UserAgent:    "UA",
	}

	// Event write errors at every step
	event := &Event{
		Common:   fullCommon,
		Category: "cat",
		Action:   "act",
		Label:    "lbl",
		Value:    10,
	}
	for i := 1; i <= 15; i++ {
		w := &stepFailWriter{failAt: i}
		_, _ = event.Write(w)
	}

	// Timing write errors
	timing := &Timing{
		Common:             fullCommon,
		ServerResponseTime: 100,
	}
	for i := 1; i <= 12; i++ {
		w := &stepFailWriter{failAt: i}
		_, _ = timing.Write(w)
	}

	// Pageview write errors
	pv := &Pageview{
		Common:        fullCommon,
		DocumentHost:  "host",
		DocumentPath:  "path",
		DocumentTitle: "title",
	}
	for i := 1; i <= 12; i++ {
		w := &stepFailWriter{failAt: i}
		_, _ = pv.Write(w)
	}

	// Exception write errors
	ex := &Exception{
		Common:      fullCommon,
		Description: "crash",
		IsFatal:     true,
	}
	for i := 1; i <= 12; i++ {
		w := &stepFailWriter{failAt: i}
		_, _ = ex.Write(w)
	}
}
