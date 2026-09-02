package p2p

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTCPTransport(t *testing.T) {
	listenAddr := "127.0.0.1:0"
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
	assert.NoError(t, err)
	assert.NotNil(t, tr.listener)
	assert.NoError(t, tr.Close())
}

func TestTCPTransportSecurityDefaults(t *testing.T) {
	tr := NewTCPTransport(TCPTransportOpts{
		ListenAddr:    "127.0.0.1:0",
		HandshakeFunc: NoOpHandshake,
		Decoder:       DefaultDecoder{},
	})

	assert.Equal(t, 128, tr.MaxConnections)
	assert.Equal(t, 2*time.Minute, tr.IdleTimeout)
}

func TestTCPTransportConnectionSlots(t *testing.T) {
	tr := NewTCPTransport(TCPTransportOpts{
		ListenAddr:     "127.0.0.1:0",
		HandshakeFunc:  NoOpHandshake,
		Decoder:        DefaultDecoder{},
		MaxConnections: 1,
	})

	assert.True(t, tr.tryAcquireConnection())
	assert.False(t, tr.tryAcquireConnection())
	tr.releaseConnection()
	assert.True(t, tr.tryAcquireConnection())
	tr.releaseConnection()
}
