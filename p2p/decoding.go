package p2p

import (
	"encoding/gob"
	"fmt"
	"io"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type GOBDecoder struct{}

func (dec GOBDecoder) Decode(r io.Reader, msg *RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct{}

func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	peekBuf := make([]byte, 4)
	n, err := r.Read(peekBuf)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("empty message")
	}

	buf := make([]byte, 1024)
	copy(buf, peekBuf[:n])

	remaining, err := r.Read(buf[n:])
	if err != nil && err != io.EOF {
		return err
	}

	msg.Payload = buf[:n+remaining]
	return nil
}
