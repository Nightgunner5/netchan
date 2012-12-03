package netchan

import (
	"encoding/gob"
	"errors"
	"io"
	"reflect"
	"strings"
	"sync"
)

// Chan represents a type-safe, bidirectional networked chan T, where T can be
// any type.
//
// The channel should always be considered buffered, as the network I/O works
// like a buffer to avoid hanging on slow connections.
type Chan struct {
	in, out   reflect.Value
	senderror error
	recverror error
	errormtx  sync.Mutex
}

func newChan(t reflect.Type, buffer int, rw io.ReadWriteCloser) (c *Chan) {
	c = new(Chan)
	chanType := reflect.ChanOf(reflect.BothDir, t)

	recv, send := reflect.MakeChan(chanType, buffer), reflect.MakeChan(chanType, buffer)
	c.in = recv.Convert(reflect.ChanOf(reflect.RecvDir, t))
	c.out = send.Convert(reflect.ChanOf(reflect.SendDir, t))

	// send
	go func() {
		encoder := gob.NewEncoder(rw)

		for {
			v, ok := send.Recv()
			if err := encoder.Encode(ok); err != nil {
				if ok {
					c.errormtx.Lock()
					c.senderror = err
					c.errormtx.Unlock()
				}
				return
			}

			if !ok {
				// Channel will be closed by the reader.
				return
			}

			if err := encoder.EncodeValue(v); err != nil {
				c.errormtx.Lock()
				c.senderror = err
				c.errormtx.Unlock()
				return
			}
		}
	}()

	// recv
	go func() {
		defer rw.Close()
		defer recv.Close()

		decoder := gob.NewDecoder(rw)

		for {
			var ok bool
			if err := decoder.Decode(&ok); err != nil {
				// We close the channel, other side recieves !ok, they close the connection, then we get EOF.
				if err == io.EOF {
					return
				}

				c.errormtx.Lock()
				c.recverror = err
				c.errormtx.Unlock()
				return
			}

			if !ok {
				return
			}

			v := reflect.New(t).Elem()
			if err := decoder.DecodeValue(v); err != nil {
				c.errormtx.Lock()
				c.recverror = err
				c.errormtx.Unlock()
				return
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

// Returns any error encountered during sending or recieving. Returns nil if
// no errors have occured.
func (c *Chan) Error() error {
	c.errormtx.Lock()
	defer c.errormtx.Unlock()

	return combineErrors(c.senderror, c.recverror)
}

func combineErrors(errs ...error) error {
	var nonNil []string
	for _, err := range errs {
		if err != nil {
			nonNil = append(nonNil, err.Error())
		}
	}

	if len(nonNil) == 0 {
		return nil
	}

	return errors.New(strings.Join(nonNil, "\n"))
}
