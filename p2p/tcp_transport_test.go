package p2p

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTCPTransport(t *testing.T) {
	listenAddr := ":4000"
	opts := TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: NoOpHandshake,
	}

	tr := NewTCPTransport(opts)
	assert.Equal(t, listenAddr, tr.ListenAddr)
	assert.NotNil(t, tr.HandshakeFunc)

	err := tr.ListenAndAccept()
	assert.Nil(t, err)
	assert.NotNil(t, tr.listener)
	_ = tr.Close()
}

func TestTCPTransportDialSendConsume(t *testing.T) {
	trA := NewTCPTransport(TCPTransportOpts{ListenAddr: ":4001"})
	trB := NewTCPTransport(TCPTransportOpts{ListenAddr: ":4002"})

	assert.Nil(t, trA.ListenAndAccept())
	assert.Nil(t, trB.ListenAndAccept())
	defer trA.Close()
	defer trB.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	var received RPC
	go func() {
		for rpc := range trB.Consume() {
			received = rpc
			wg.Done()
			return
		}
	}()

	var peerB Peer
	peerBConnected := make(chan struct{})
	trA.OnPeer = func(p Peer) error {
		peerB = p
		close(peerBConnected)
		return nil
	}

	assert.Nil(t, trA.Dial(":4002"))

	select {
	case <-peerBConnected:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for peer connection")
	}

	testRPC := RPC{
		Type:    MsgStore,
		Key:     "hello-key",
		Payload: []byte("network distributed payload"),
	}

	assert.Nil(t, peerB.Send(testRPC))

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		assert.Equal(t, testRPC.Type, received.Type)
		assert.Equal(t, testRPC.Key, received.Key)
		assert.Equal(t, testRPC.Payload, received.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for RPC delivery")
	}
}
