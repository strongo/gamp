// +build go1.10

package gamp

import (
	"strings"
)

func messageToString(m Message) string {
	buffer := new(strings.Builder)
	if _, err := m.Write(buffer); err != nil {
		panic(err.Error())
	}
	return buffer.String()
}
