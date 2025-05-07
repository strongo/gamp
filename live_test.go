package gamp

import (
	"net/http"
	"strings"
	"testing"
)

func TestLive(t *testing.T) {
	common := Common{
		TrackingID:    "UA-FAKE-ID",
		ClientID:      testClientID,
		DataSource:    "unit-tests",
		ApplicationID: "go.unit-tests",
		UserID:        "user-123",
		UserAgent:     "go-tests",
		UserLanguage:  "en-en",
	}

	gaMeasurement := NewBufferedClient(GaHTTPS+"debug/", http.DefaultClient, nil)

	gaEvent := NewEvent("test-category1.2", "test-action2.2", common)
	gaEvent.Label = "test-label1"
	if err := gaMeasurement.Queue(gaEvent); err != nil {
		t.Errorf("Failed to queue(): %v", err.Error())
	}

	checkErr := func(err error, prefix string) {
		errText := err.Error()
		if strings.Contains(errText, "HTTP status=404") {
			t.Logf("WARNING: %s, HTTP status code: %v - replace TrackingID with a vaild one!", prefix, http.StatusNotFound)
		} else {
			t.Errorf("%s: %s", prefix, err.Error())
		}
	}

	if err := gaMeasurement.Flush(); err != nil {
		checkErr(err, "Failed to flush #1")
		return
	}

	gaPageView := NewPageviewWithDocumentHost("test.host", "/test/path2", "Test title")
	gaPageView.Common = common

	if err := gaMeasurement.Queue(gaPageView); err != nil {
		checkErr(err, "Failed to queue #1")
		return
	}
	if err := gaMeasurement.Flush(); err != nil {
		checkErr(err, "Failed to flush #2")
		return
	}
}
