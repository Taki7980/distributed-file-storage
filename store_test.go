package main

import (
	"bytes"
	"testing"
)

func TestStore(t *testing.T) {
	opts := StoreOpt{
		PathTransformFunc: DefaultPathTransformFunc,
	}
	s := NewStore(opts)
	data := bytes.NewReader([]byte("some jpgs"))
	if err := s.writeStream("mySpecialText", data); err != nil {
		t.Error(err)
	}
}
