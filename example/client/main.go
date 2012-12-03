package main

import (
	"github.com/Nightgunner5/netchan/example/packet"
	"github.com/nsf/termbox-go"
	"log"
	"net"
	"time"
)

func main() {
	c, err := net.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		log.Fatal(err)
	}
	quit := make(chan bool)
	send, recv := packet.Chan(c)

	go func() {
		tick := time.Tick(30 * time.Second)

		ka := &packet.KeepAlive{}
		for {
			select {
			case <-tick:
				send <- ka
			case <-quit:
				return
			}
		}
	}()

	typed := make(chan rune)
	backspace := make(chan bool)
	enter := make(chan bool)
	resized := make(chan struct{w, h int})

	go func() {
		var (
			dimensions struct{w, h int}
			scrollback [][]rune
			typing []rune

			repaintLower = func() {
				termbox.SetCell(0, dimensions.h - 1, '>', termbox.AttrBold, 0)
				i := 2
				show := typing
				if dimensions.w - 3 < len(show) {
					show = show[len(show) - dimensions.w - 3:]
				}
				for _, r := range show {
					termbox.SetCell(i, dimensions.h - 1, r, 0, 0)
					i++
				}
				termbox.SetCursor(i, dimensions.h - 1)
				for i < dimensions.w {
					termbox.SetCell(i, dimensions.h - 1, ' ', 0, 0)
					i++
				}
				termbox.Flush()
			}

			repaint = func() {
				termbox.Clear(0, 0)
				lines := dimensions.h - 1
				linebuffer := scrollback
				for lines > 0 && len(linebuffer) > 0 {
					line := linebuffer[len(linebuffer) - 1]
					linebuffer = linebuffer[:len(linebuffer) - 1]
					lines--
					for i, r := range line {
						termbox.SetCell(i, lines, r, 0, 0)
					}
				}
				repaintLower()
			}
		)

		for {
			select {
			case pkt, ok := <-recv:
				if !ok {
					close(quit)
					return
				}
				switch p := pkt.(type) {
				case *packet.KeepAlive:
					// no-op
				case *packet.Message:
					scrollback = append(scrollback, []rune(p.Text))
					repaint()
				case *packet.Quit:
					close(send)
				}
			case <-backspace:
				if len(typing) > 0 {
					typing = typing[:len(typing) - 1]
				}
				repaintLower()
			case r := <-typed:
				typing = append(typing, r)
				repaintLower()
			case <-enter:
				send <- &packet.Message{Text: string(typing)}
				typing = nil
				repaintLower()
			case r := <-resized:
				dimensions = r
				repaint()
			}
		}
	}()

	if err := termbox.Init(); err != nil {
		log.Fatal(err)
	}
	defer termbox.Close()

	{
		w, h := termbox.Size()
		resized <- struct{w, h int}{w, h}
	}
	for {
		e := termbox.PollEvent()
		switch e.Type {
		case termbox.EventKey:
			if e.Ch == 0 {
				switch e.Key {
				case termbox.KeyBackspace, termbox.KeyBackspace2:
					backspace <- true
				case termbox.KeyEnter:
					enter <- true
				case termbox.KeyCtrlC:
					send <- &packet.Quit{}
					<-quit
					return
				}
			} else {
				typed <- e.Ch
			}
		case termbox.EventResize:
			resized <- struct{w, h int}{e.Width, e.Height}
		case termbox.EventError:
			log.Fatal(e.Err)
		}
	}
}
