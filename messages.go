package gamp

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Message interface should be implemented by all GA messages
type Message interface {
	GetTrackingID() string
	SetTrackingID(string)
	fmt.Stringer
	//
	Write(w io.Writer) (n int, err error)
}

func messageToString(m Message) string {
	buffer := new(strings.Builder)
	if _, err := m.Write(buffer); err != nil {
		panic(err.Error())
	}
	return buffer.String()
}

// Common properties of GA message
type Common struct {
	TrackingID    string `key:"tid" required:"true"`
	ClientID      string `key:"cid"`
	UserID        string `key:"uid"`
	UserLanguage  string `key:"ul"`
	UserAgent     string `key:"ua"`
	DataSource    string `key:"ds"` // https://developers.google.com/analytics/devguides/collection/protocol/v1/parameters#ds
	ApplicationID string `key:"aid"`
}

// GetTrackingID returns tracking ID of GA message
func (c Common) GetTrackingID() string {
	return c.TrackingID
}

// GetTrackingID returns tracking ID of GA message
func (c *Common) SetTrackingID(v string) {
	c.TrackingID = v
}

// Event is GA message of 'event' type
type Event struct {
	Common
	Category string `key:"ec" required:"true"`
	Action   string `key:"ea" required:"true"`
	Label    string `key:"el"`
	Value    uint   `key:"ev"`
}

var _ Message = (*Event)(nil)

// Timing is GA message of 'timing' type
type Timing struct {
	Common
	ServerResponseTime uint
}

var _ Message = (*Timing)(nil)

func (c Common) write1st(w io.Writer) (n int, err error) {
	_n, err := w.Write([]byte("v=1"))
	n += _n
	if err != nil {
		return n, err
	}
	if c.TrackingID != "" {
		_n, err = w.Write([]byte("&tid=" + c.TrackingID))
		n += _n
		if err != nil {
			return n, err
		}
	}
	return
}

func (c Common) writeRest(w io.Writer) (n int, err error) {
	var _n int
	if c.ClientID != "" {
		_n, err = w.Write([]byte("&cid=" + url.QueryEscape(c.ClientID)))
		n += _n
		if err != nil {
			return n, err
		}
	}
	if c.UserID != "" {
		_n, err = w.Write([]byte("&uid=" + url.QueryEscape(c.UserID)))
		n += _n
		if err != nil {
			return n, err
		}
	}

	if c.UserLanguage != "" {
		_n, err = w.Write([]byte("&ul=" + c.UserLanguage))
		n += _n
		if err != nil {
			return n, err
		}
	}

	if c.UserAgent != "" {
		_n, err = w.Write([]byte("&ua=" + url.QueryEscape(c.UserAgent)))
		n += _n
		if err != nil {
			return n, err
		}
	}

	//if c.ApplicationID != "" {
	//	_n, err = w.Write([]byte("&aid="+url.QueryEscape(c.ApplicationID)))
	//	n += _n
	//	if err != nil {
	//		return n, err
	//	}
	//	_n, err = w.Write([]byte("&an="+url.QueryEscape(c.ApplicationID)))
	//	n += _n
	//	if err != nil {
	//		return n, err
	//	}
	//}
	return
}

// NewEvent creates new event
func NewEvent(category, action string, common Common) *Event {
	return NewEventWithLabel(category, action, "", common)
}

// NewTiming creates new timing
func NewTiming(serverResponseTime time.Duration) *Timing {
	return &Timing{ServerResponseTime: uint(serverResponseTime.Nanoseconds() / int64(time.Millisecond))}
}

// NewEventWithLabel creates new event with label
func NewEventWithLabel(category, action, label string, common Common) *Event {
	event := Event{
		Category: category,
		Action:   action,
		Label:    label,
		Common:   common,
	}
	if err := event.Validate(); err != nil {
		panic(err.Error())
	}
	return &event
}

// Validate is checking message for validity
func (e Event) Validate() error {
	if e.Category == "" {
		return errors.New("Missing required parameter: Category")
	}
	if e.Action == "" {
		return errors.New("Missing required parameter: Action")
	}
	return nil
}

// Write serializes timing message
func (t Timing) Write(w io.Writer) (n int, err error) {
	var _n int
	_n, err = t.Common.write1st(w)
	if n += _n; err != nil {
		return
	}

	_n, err = w.Write([]byte("&t=timing"))
	if n += _n; err != nil {
		return n, err
	}

	_n, err = t.Common.writeRest(w)
	if n += _n; err != nil {
		return
	}

	if t.ServerResponseTime != 0 {
		_n, err = w.Write([]byte("&srt=" + strconv.FormatUint(uint64(t.ServerResponseTime), 10)))
		if n += _n; err != nil {
			return n, err
		}
	}

	return n, err
}

// Write serializes event message
func (e Event) Write(w io.Writer) (n int, err error) {
	var _n int

	_n, err = e.Common.write1st(w)
	if n += _n; err != nil {
		return
	}

	_n, err = w.Write([]byte("&t=event"))
	if n += _n; err != nil {
		return n, err
	}

	_n, err = e.Common.writeRest(w)
	if n += _n; err != nil {
		return
	}

	_n, err = w.Write([]byte("&ec=" + url.QueryEscape(e.Category)))
	if n += _n; err != nil {
		return n, err
	}

	_n, err = w.Write([]byte("&ea=" + url.QueryEscape(e.Action)))
	if n += _n; err != nil {
		return n, err
	}

	if e.Value != 0 {
		_n, err = w.Write([]byte("&ev=" + strconv.FormatUint(uint64(e.Value), 10)))
		if n += _n; err != nil {
			return n, err
		}
	}

	if e.Label != "" {
		_n, err = w.Write([]byte("&el=" + url.QueryEscape(e.Label)))
		if n += _n; err != nil {
			return n, err
		}
	}

	return n, err
}

// String returns message as URL encoded string
func (e *Event) String() string {
	return messageToString(e)
}

// String returns message as URL encoded string
func (t *Timing) String() string {
	return messageToString(t)
}

// Pageview message
type Pageview struct {
	Common
	//DocumentLocation string `key:"dl"`
	DocumentHost  string `key:"dh"`
	DocumentPath  string `key:"dp"`
	DocumentTitle string `key:"dt"`
}

var _ Message = (*Pageview)(nil)

// NewPageviewWithDocumentHost creates new pageview message with document host value
func NewPageviewWithDocumentHost(documentHost, documentPath, documentTitle string) Pageview {
	return Pageview{
		DocumentHost:  documentHost,
		DocumentPath:  documentPath,
		DocumentTitle: documentTitle,
	}
}

// String returns message as URL encoded string
func (m *Pageview) String() string {
	return messageToString(m)
}

// Write serializes page view message
func (m Pageview) Write(w io.Writer) (n int, err error) {
	var _n int

	_n, err = m.Common.write1st(w)
	if n += _n; err != nil {
		return
	}

	_n, err = w.Write([]byte("&t=pageview"))
	if n += _n; err != nil {
		return n, err
	}

	_n, err = m.Common.writeRest(w)
	if n += _n; err != nil {
		return
	}

	_n, err = w.Write([]byte("&dh=" + m.DocumentHost)) // We suppose we do not need to encode host
	if n += _n; err != nil {
		return n, err
	}

	_n, err = w.Write([]byte("&dp=" + url.QueryEscape(m.DocumentPath)))
	if n += _n; err != nil {
		return n, err
	}

	_n, err = w.Write([]byte("&dt=" + url.QueryEscape(m.DocumentTitle)))
	if n += _n; err != nil {
		return n, err
	}

	return
}

// Exception message
type Exception struct {
	Common
	Description string `key:"exd"`
	IsFatal     bool   `key:"exf"`
}

var _ Message = (*Exception)(nil)

// NewException creates new exception message
func NewException(description string, isFatal bool) *Exception {
	return &Exception{
		Description: description,
		IsFatal:     isFatal,
	}
}

func (m *Exception) String() string {
	return messageToString(m)
}

// Write serializes exception message
func (m Exception) Write(w io.Writer) (n int, err error) {
	var _n int

	_n, err = m.Common.write1st(w)
	if n += _n; err != nil {
		return
	}

	_n, err = w.Write([]byte("&t=exception"))
	if n += _n; err != nil {
		return n, err
	}

	_n, err = m.Common.writeRest(w)
	if n += _n; err != nil {
		return
	}

	_n, err = w.Write([]byte("&exd=" + url.QueryEscape(m.Description))) // We suppose we do not need to encode host
	if n += _n; err != nil {
		return n, err
	}

	if m.IsFatal {
		_n, err = w.Write([]byte("&exf=1"))
		if n += _n; err != nil {
			return n, err
		}
	}

	return
}
