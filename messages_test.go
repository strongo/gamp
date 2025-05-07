package gamp

import (
	"fmt"
	"net/url"
	"testing"
)

const testClientID = "bd4e1566-9626-4bf8-8f03-e54aa678c23f"

func TestNewEvent(t *testing.T) {
	e := NewEvent("Ca1", "Ac1", Common{ClientID: testClientID})
	if e.Category != "Ca1" {
		t.Error("Category is not set correctly")
	}
	if e.Action != "Ac1" {
		t.Error("Action is not set correctly")
	}
}

func TestMessageString(t *testing.T) {
	common := Common{TrackingID: "Track1", ClientID: "Client2"}
	for _, item := range []struct {
		Message
		Expected string
	}{
		{
			NewEvent("Category3", "Action4", common),
			"v=1&tid=Track1&t=event&cid=Client2&ec=Category3&ea=Action4",
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
