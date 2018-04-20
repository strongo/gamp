// +build !go1.10

package gamp

import "bytes"

func messageToString(m Message) string {
	buffer := new(bytes.Buffer)
	if _, err := m.Write(buffer); err != nil {
		panic(err.Error())
	}
	return buffer.String()
}
