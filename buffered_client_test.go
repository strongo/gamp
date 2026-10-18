package gamp

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type badBodyReader struct{}

func (b badBodyReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated body read error")
}
func (b badBodyReader) Close() error { return nil }

type failTransport struct{}

func (failTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("simulated network failure")
}

type errMessage struct {
	Common
}

func (m *errMessage) Write(w io.Writer) (int, error) {
	return 0, errors.New("simulated message write error")
}

func (m *errMessage) String() string { return "" }

type hugeMessage struct {
	Common
	called bool
}

func (m *hugeMessage) Write(w io.Writer) (int, error) {
	if !m.called {
		m.called = true
		return bufferSizeLimit + 1, nil
	}
	return w.Write([]byte("v=1&tid=UA-1"))
}

func (m *hugeMessage) String() string { return "" }

func TestNewBufferedClient(t *testing.T) {
	t.Run("empty_endpoint", func(t *testing.T) {
		c := NewBufferedClient("", nil, nil)
		if c.endpoint != GaHTTPS {
			t.Errorf("expected %s, got %s", GaHTTPS, c.endpoint)
		}
	})

	t.Run("https_endpoint", func(t *testing.T) {
		c := NewBufferedClient("https", nil, nil)
		if c.endpoint != GaHTTPS {
			t.Errorf("expected %s, got %s", GaHTTPS, c.endpoint)
		}
	})

	t.Run("http_endpoint", func(t *testing.T) {
		c := NewBufferedClient("http", nil, nil)
		if c.endpoint != GaHTTP {
			t.Errorf("expected %s, got %s", GaHTTP, c.endpoint)
		}
	})

	t.Run("custom_endpoint", func(t *testing.T) {
		c := NewBufferedClient("http://custom.com/", nil, nil)
		if c.endpoint != "http://custom.com/" {
			t.Errorf("expected http://custom.com/, got %s", c.endpoint)
		}
	})
}

func TestBufferedClient_Queue(t *testing.T) {
	t.Run("nil_client", func(t *testing.T) {
		var c *BufferedClient
		if err := c.Queue(NewEvent("c", "a", Common{TrackingID: "UA-1"})); err != nil {
			t.Errorf("expected nil for nil client, got %v", err)
		}
	})

	t.Run("missing_tracking_id", func(t *testing.T) {
		c := NewBufferedClient("", nil, nil)
		err := c.Queue(NewEvent("c", "a", Common{}))
		if err == nil || !errors.Is(err, ErrNoTrackingID) {
			t.Errorf("expected ErrNoTrackingID, got %v", err)
		}
	})

	t.Run("message_write_error", func(t *testing.T) {
		c := NewBufferedClient("", nil, nil)
		err := c.Queue(&errMessage{Common: Common{TrackingID: "UA-1"}})
		if err == nil || !strings.Contains(err.Error(), "simulated message write error") {
			t.Errorf("expected write error, got %v", err)
		}
	})

	t.Run("buffer_size_limit_exceeded_flush_fails", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		c := NewBufferedClient(srv.URL+"/", srv.Client(), nil)
		c.queueDepth = 1
		c.buffer.WriteString("pre-existing")

		msg := &hugeMessage{Common: Common{TrackingID: "UA-1"}}
		err := c.Queue(msg)
		if err == nil {
			t.Fatal("expected flush error, got nil")
		}
	})

	t.Run("buffer_size_limit_exceeded_flush_succeeds", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		c := NewBufferedClient(srv.URL+"/", srv.Client(), nil)
		c.queueDepth = 1
		c.buffer.WriteString("pre-existing")

		msg := &hugeMessage{Common: Common{TrackingID: "UA-1"}}
		err := c.Queue(msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.QueueDepth() != 1 {
			t.Errorf("expected queueDepth 1, got %d", c.QueueDepth())
		}
	})

	t.Run("auto_flush_at_20", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		c := NewBufferedClient(srv.URL+"/", srv.Client(), nil)
		msg := NewEvent("c", "a", Common{TrackingID: "UA-1"})
		for i := 0; i < 20; i++ {
			if err := c.Queue(msg); err != nil {
				t.Fatalf("failed at %d: %v", i, err)
			}
		}
		if c.QueueDepth() != 0 {
			t.Errorf("expected queue depth 0 after auto-flush, got %d", c.QueueDepth())
		}
	})
}

func TestBufferedClient_Flush(t *testing.T) {
	t.Run("empty_queue", func(t *testing.T) {
		c := NewBufferedClient("", nil, nil)
		if err := c.Flush(); err != nil {
			t.Errorf("expected nil for empty flush, got %v", err)
		}
	})

	t.Run("single_message", func(t *testing.T) {
		var requestedURL string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedURL = r.URL.String()
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		c := NewBufferedClient(srv.URL+"/", srv.Client(), nil)
		_ = c.Queue(NewEvent("cat", "act", Common{TrackingID: "UA-1"}))
		if c.QueueDepth() != 1 {
			t.Fatalf("expected queue depth 1, got %d", c.QueueDepth())
		}
		if err := c.Flush(); err != nil {
			t.Fatalf("Flush error: %v", err)
		}
		if !strings.Contains(requestedURL, "/collect?") {
			t.Errorf("expected /collect?, got %s", requestedURL)
		}
	})

	t.Run("batch_messages", func(t *testing.T) {
		var requestedURL string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedURL = r.URL.String()
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		c := NewBufferedClient(srv.URL+"/", srv.Client(), nil)
		_ = c.Queue(NewEvent("cat", "act", Common{TrackingID: "UA-1"}))
		_ = c.Queue(NewEvent("cat2", "act2", Common{TrackingID: "UA-1"}))
		if c.QueueDepth() != 2 {
			t.Fatalf("expected queue depth 2, got %d", c.QueueDepth())
		}
		if err := c.Flush(); err != nil {
			t.Fatalf("Flush error: %v", err)
		}
		if !strings.Contains(requestedURL, "/batch") {
			t.Errorf("expected /batch, got %s", requestedURL)
		}
	})
}

func TestBufferedClient_HandleAPIResponse(t *testing.T) {
	t.Run("http_error_with_onError", func(t *testing.T) {
		var reportedErr error
		client := &http.Client{Transport: failTransport{}}
		c := NewBufferedClient("http://localhost/", client, func(err error) {
			reportedErr = err
		})
		_ = c.Queue(NewEvent("cat", "act", Common{TrackingID: "UA-1"}))
		err := c.Flush()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if reportedErr == nil {
			t.Error("expected onError to be called")
		}
	})

	t.Run("http_error_without_onError", func(t *testing.T) {
		client := &http.Client{Transport: failTransport{}}
		c := NewBufferedClient("http://localhost/", client, nil)
		_ = c.Queue(NewEvent("cat", "act", Common{TrackingID: "UA-1"}))
		err := c.Flush()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("bad_response_body_read", func(t *testing.T) {
		c := NewBufferedClient("", nil, nil)
		resp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       badBodyReader{},
		}
		err := c.handleAPIResponse("http://example.com", resp, nil)
		if err == nil || !strings.Contains(err.Error(), "failed to read response body") {
			t.Errorf("expected read body error, got %v", err)
		}
	})

	t.Run("resp_nil_panic", func(t *testing.T) {
		c := NewBufferedClient("", nil, nil)
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when resp and err are both nil")
			}
		}()
		_ = c.handleAPIResponse("http://example.com", nil, nil)
	})
}
