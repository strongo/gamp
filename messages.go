package gamp

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/url"
	"strconv"
	"time"
)

type Message interface {
	GetTrackingID() string
	fmt.Stringer
	//
	Write(w io.Writer) (n int, err error)
}

type Common struct {
	TrackingID    string `key:"tid" required:"true"`
	ClientID      string `key:"cid"`
	UserID        string `key:"uid"`
	UserLanguage  string `key:"ul"`
	UserAgent     string `key:"ua"`
	DataSource    string `key:"ds"` // https://developers.google.com/analytics/devguides/collection/protocol/v1/parameters#ds
	ApplicationID string `key:"aid"`
}

func (c Common) GetTrackingID() string {
	return c.TrackingID
}

type Event struct {
	Common
	Category string `key:"ec" required:"true"`
	Action   string `key:"ea" required:"true"`
	Label    string `key:"el"`
	Value    uint   `key:"ev"`
}

var _ Message = (*Event)(nil)

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

func NewEvent(category, action string, common Common) Event {
	return NewEventWithLabel(category, action, "", common)
}

func NewTiming(serverResponseTime time.Duration) Timing {
	return Timing{ServerResponseTime: uint(serverResponseTime.Nanoseconds() / int64(time.Millisecond))}
}

func NewEventWithLabel(category, action, label string, common Common) Event {
	e := Event{
		Category: category,
		Action:   action,
		Label:    label,
		Common:   common,
	}
	if err := e.Validate(); err != nil {
		panic(err.Error())
	}
	return e
}

func (e Event) Validate() error {
	if e.Category == "" {
		return errors.New("Missing required parameter: Category")
	}
	if e.Action == "" {
		return errors.New("Missing required parameter: Action")
	}
	return nil
}

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



func (m Event) String() string {
	return messageToString(m)
}

func (m Timing) String() string {
	return messageToString(m)
}

type Pageview struct {
	Common
	//DocumentLocation string `key:"dl"`
	DocumentHost  string `key:"dh"`
	DocumentPath  string `key:"dp"`
	DocumentTitle string `key:"dt"`
}

var _ Message = (*Pageview)(nil)

func NewPageviewWithDocumentHost(documentHost, documentPath, documentTitle string) Pageview {
	return Pageview{
		DocumentHost:  documentHost,
		DocumentPath:  documentPath,
		DocumentTitle: documentTitle,
	}
}

func (m Pageview) String() string {
	return messageToString(m)
}

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

type Exception struct {
	Common
	Description string `key:"exd"`
	IsFatal     bool   `key:"exf"`
}

var _ Message = (*Exception)(nil)

func NewException(description string, isFatal bool) Exception {
	return Exception{
		Description: description,
		IsFatal:     isFatal,
	}
}

func (m Exception) String() string {
	return messageToString(m)
}

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
