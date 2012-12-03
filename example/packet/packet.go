package packet

import (
	"github.com/Nightgunner5/netchan"
	"io"
	"net"
	"reflect"
)

const MaxQueuedPackets = 16

func extract(c *netchan.Chan) (chan<- interface{}, <-chan interface{}) {
	send, recv := make(chan interface{}), make(chan interface{})

	go func(w chan<- Packet) {
		for packet := range send {
			switch p := packet.(type) {
			case *KeepAlive:
				w <- Packet{KeepAlive: p}
			case *Message:
				w <- Packet{Message: p}
			case *Quit:
				w <- Packet{Quit: p}
			default:
				panic(packet)
			}
		}
		close(w)
	}(c.ChanSend().(chan<- Packet))

	go func(r <-chan Packet) {
		for p := range r {
			if p.KeepAlive != nil {
				recv <- p.KeepAlive
			}
			if p.Message != nil {
				recv <- p.Message
			}
			if p.Quit != nil {
				recv <- p.Quit
			}
		}
		close(recv)
	}(c.ChanRecv().(<-chan Packet))
	return send, recv
}

func Chan(rw io.ReadWriteCloser) (chan<- interface{}, <-chan interface{}) {
	c := netchan.New(rw, reflect.TypeOf(Packet{}), MaxQueuedPackets)

	return extract(c)
}

func Listen(l net.Listener, f func(net.Addr, chan<- interface{}, <-chan interface{})) error {
	return netchan.Listen(l, func(addr net.Addr, c *netchan.Chan) {
		send, recv := extract(c)
		f(addr, send, recv)
	}, reflect.TypeOf(Packet{}), MaxQueuedPackets)
}

type Packet struct {
	*KeepAlive
	*Message
	*Quit
}
