package netchan

import (
	"io"
	"reflect"
)

// Returns a new channel (of type chan t). No other reads from or writes to
// the io.ReadWriteCloser should occur. A buffer size <= 0 is treated as a
// buffer size of 1.
func New(rw io.ReadWriteCloser, t reflect.Type, buffer int) *Chan {
	buffer--
	if buffer < 0 {
		buffer = 0
	}
	return newChan(t, buffer, rw)
}
