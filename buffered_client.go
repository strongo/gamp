package gamp

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"errors"
)

type BufferedClient struct {
	sync.Mutex
	endpoint   string
	buffer     bytes.Buffer
	queueDepth int
	httpClient *http.Client
	onError    func(err error)
}

const (
	gaHost = "www.google-analytics.com"

	GaHTTPS    = "https://" + gaHost + "/"
	GaHTTP = "http://" + gaHost + "/"

	bufferSizeLimit = 16*1024*1024 - 1
)

func NewBufferedClient(endpoint string, httpClient *http.Client, onError func(err error)) *BufferedClient {
	switch endpoint {
	case "", "https":
		endpoint = GaHTTPS
	case "http":
		endpoint = GaHTTP
	}
	return &BufferedClient{
		endpoint: endpoint,
		httpClient: httpClient,
		onError:    onError,
	}
}

func (api *BufferedClient) QueueDepth() int {
	return api.queueDepth
}

var ErrNoTrackingID = errors.New("no tracking ID")

func (api *BufferedClient) Queue(message Message) error {
	if message.GetTrackingID() == "" {
		return ErrNoTrackingID
	}
	api.Lock()
	defer api.Unlock()

	bufferSize := api.buffer.Len()

	if api.queueDepth > 0 {
		api.buffer.Write([]byte("\n"))
	}

	if n, err := message.Write(&api.buffer); err != nil {
		api.buffer.Truncate(bufferSize)
		return err
	} else if bufferSize+n > bufferSizeLimit {
		api.buffer.Truncate(bufferSize)
		if err = api.flush(); err != nil {
			return err
		}
		return api.Queue(message)
	}

	if api.queueDepth += 1; api.queueDepth == 20 {
		return api.flush()
	}
	return nil
}

func (api *BufferedClient) Flush() (err error) {
	api.Lock()
	defer api.Unlock()
	return api.flush()
}

func (api *BufferedClient) flush() error {
	switch api.queueDepth {
	case 0:
		return nil
	case 1:
		return api.sendSingle()
	default:
		return api.sendBatch()
	}
}

func (api *BufferedClient) sendSingle() (err error) {
	url := api.endpoint + "collect?" + api.buffer.String()
	resp, err := api.httpClient.Get(url)
	return api.handleApiResponse(url, resp, err)
}

func (api *BufferedClient) sendBatch() error {
	url := api.endpoint + "batch"
	resp, err := api.httpClient.Post(url, "text/plain", &api.buffer)
	return api.handleApiResponse(url, resp, err)
}

func (api *BufferedClient) handleApiResponse(url string, resp *http.Response, err error) error {
	api.buffer.Reset()
	api.queueDepth = 0
	if err != nil {
		if api.onError != nil {
			api.onError(err)
		}
		return err
	} else if resp != nil {
		if resp.StatusCode == http.StatusOK {
			return nil
		} else {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			return fmt.Errorf("%v => HTTP status=%v, body: %v", url, resp.StatusCode, buf.String())
		}
	} else {
		panic("resp is nil")
	}
}
