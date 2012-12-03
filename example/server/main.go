package main

import (
	"github.com/Nightgunner5/netchan/example/packet"
	"log"
	"net"
)

func main() {
	ln, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal(err)
	}

	connect, disconnect, message := broadcaster()

	log.Fatal(packet.Listen(ln, func(addr net.Addr, send chan<- interface{}, recv <-chan interface{}) {
		connect <- &send
		for p := range recv {
			log.Printf("%v: %#v", addr, p)
			if m, ok := p.(*packet.Message); ok {
				message <- &packet.Message{
					Text: addr.String() + ": " + m.Text,
				}
			} else {
				send <- p
			}
		}
		close(send)
		disconnect <- &send
	}))
}

func broadcaster() (chan<- *chan<- interface{}, chan<- *chan<- interface{}, chan<- interface{}) {
	connect := make(chan *chan<- interface{})
	disconnect := make(chan *chan<- interface{})
	message := make(chan interface{})

	go func() {
		channels := make(map[*chan<- interface{}]chan<- interface{})

		for {
			select {
			case c := <-connect:
				channels[c] = *c
			case c := <-disconnect:
				delete(channels, c)
			case m := <-message:
				for _, c := range channels {
					// Non-blocking send
					select {
					case c <- m:
					default:
					}
				}
			}
		}
	}()

	return connect, disconnect, message
}
