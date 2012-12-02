package netchan

import (
	"net"
	"reflect"
	"testing"
)

type Type1 struct {
	ID   uint64
	Name string
}

func TestChanSend(t *testing.T) {
	t.Parallel()

	l, r := net.Pipe()
	left := newChan(reflect.TypeOf(Type1{}), 1, l)
	right := newChan(reflect.TypeOf(Type1{}), 1, r)

	t1 := Type1{42, "potato"}
	left.Send(t1)
	left.Close()
	v, ok := right.Recv()
	if !ok {
		t.Error("Channel was closed too soon")
	}
	t2 := v.(Type1)

	if t1 != t2 {
		t.Errorf("Expected %#v but got %#v", t1, t2)
	}

	_, ok = right.Recv()
	if ok {
		t.Error("Recv returned success after channel was closed.")
	}

	for err := range left.Errors() {
		t.Error(err)
	}

	for err := range right.Errors() {
		t.Error(err)
	}
}
