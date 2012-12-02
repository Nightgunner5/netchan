package netchan

import (
	"encoding/gob"
	"io"
	"reflect"
)

// Chan represents a type-safe, bidirectional networked chan T, where T can be
// any type.
//
// The channel should always be considered buffered, as the network I/O works
// like a buffer to avoid hanging on slow connections.
type Chan struct {
	in, out reflect.Value
	errors  <-chan error
}

func newChan(t reflect.Type, buffer int, rw io.ReadWriteCloser) (c *Chan) {
	c = new(Chan)
	chanType := reflect.ChanOf(reflect.BothDir, t)

	recv, send := reflect.MakeChan(chanType, buffer), reflect.MakeChan(chanType, buffer)
	c.in = recv.Convert(reflect.ChanOf(reflect.RecvDir, t))
	c.out = send.Convert(reflect.ChanOf(reflect.SendDir, t))

	errors := make(chan error)
	c.errors = errors

	// send
	go func() {
		encoder := gob.NewEncoder(rw)

		for {
			v, ok := send.Recv()
			if err := encoder.Encode(ok); err != nil {
				errors <- err

				// TODO: which errors are fatal?
				continue
			}

			if !ok {
				// Channel will be closed by the reader.
				return
			}

			if err := encoder.EncodeValue(v); err != nil {
				errors <- err

				// TODO: which errors are fatal?
				continue
			}
		}
	}()

	// recv
	go func() {
		v := reflect.New(t).Elem()
		decoder := gob.NewDecoder(rw)

		for {
			var ok bool
			if err := decoder.Decode(&ok); err != nil {
				// We close the channel, other side recieves !ok, they close the connection, then we get EOF.
				if err == io.EOF {
					rw.Close()
					recv.Close()
					close(errors)
					return
				}

				errors <- err

				// TODO: which errors are fatal?
				continue
			}

			if !ok {
				rw.Close()
				recv.Close()
				close(errors)
				return
			}

			if err := decoder.DecodeValue(v); err != nil {
				errors <- err

				// TODO: which errors are fatal?
				continue
			}

			recv.Send(v)
		}
	}()
	return
}

// Send a value to ChanSend. If the connection has been closed, this method will
// panic as described in the Go spec for channels.
func (c *Chan) Send(v interface{}) {
	c.out.Send(reflect.ValueOf(v))
}

// Read a value from ChanRecv. In the case of a closed connection, Recv will
// return the zero value for T for v and false for ok.
func (c *Chan) Recv() (v interface{}, ok bool) {
	var val reflect.Value
	val, ok = c.in.Recv()
	v = val.Interface()
	return
}

// Synonym for close(ChanSend().(chan<- T)).
func (c *Chan) Close() {
	c.out.Close()
}

// Returns a chan<- T which can be used in the same manner as Chan.Send().
func (c *Chan) ChanSend() interface{} {
	return c.out.Interface()
}

// Returns a <-chan T which can be used in the same manner as Chan.Recv().
func (c *Chan) ChanRecv() interface{} {
	return c.in.Interface()
}

// Returns a channel where I/O errors will be sent. This channel is unbuffered
// and the read or write goroutine will wait to send to this channel.
func (c *Chan) Errors() <-chan error {
	return c.errors
}
