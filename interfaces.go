package gamp

import (
	"fmt"
	"io"
)

// Sender sends analytic messages
type Sender interface {
	Send(message Message) error
}

// BatchSender sends a batch of messages to analytics provider
type BatchSender interface {
	SendBatch(messages []Message) error
}

type writeStringer interface {
	io.Writer
	fmt.Stringer
}

type Buffer interface {
	Queue(message Message) error
	Flush() error
}