package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPTransport(t *testing.T) {
	listenAddr := ":4000"
	opts := TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: NoOpHandshake,
		Decoder:       DefaultDecoder{},
	}

	tr := NewTCPTransport(opts)
	assert.Equal(t, listenAddr, tr.ListenAddr)
	assert.NotNil(t, tr.HandshakeFunc)
	assert.NotNil(t, tr.Decoder)

	err := tr.ListenAndAccept()
	assert.Nil(t, err)
	assert.NotNil(t, tr.listener)
}
