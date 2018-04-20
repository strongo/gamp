package gamp

import (
	"net/http"
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

	gaMeasurement := NewBufferedClient(GaHTTPS + "debug/", http.DefaultClient, nil)

	gaEvent := NewEvent("test-category1.2", "test-action2.2", common)
	gaEvent.Label = "test-label1"
	gaMeasurement.Queue(gaEvent)


	if err := gaMeasurement.Flush(); err != nil {
		t.Errorf("Failed to flush(): %v", err.Error())
	}

	gaPageView := NewPageviewWithDocumentHost("test.host", "/test/path2", "Test title")
	gaPageView.Common = common
	if err := gaMeasurement.Queue(gaPageView); err != nil {
		t.Error(err)
	}
	if err := gaMeasurement.Flush(); err != nil {
		t.Errorf("Failed to flush(): %v", err.Error())
	}
}
