package netchan

import (
	"net"
	"reflect"
)

// Listens for connections. For each connection, f will be called in a new
// goroutine with a new networked channel (of type chan t) with the given buffer
// size. A buffer size <= 0 is treated as a buffer size of 1.
//
// If a fatal network error is encountered, it will be returned and no new
// connections will be handled. Non-fatal network errors are ignored.
func Listen(l net.Listener, f func(*Chan), t reflect.Type, buffer int) error {
	buffer--
	if buffer < 0 {
		buffer = 0
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			if !err.(net.Error).Temporary() {
				return err
			}
		}
		go f(newChan(t, buffer, conn))
	}
	return nil
}
