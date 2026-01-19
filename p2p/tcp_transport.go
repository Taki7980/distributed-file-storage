package p2p

import (
	"fmt"
	"net"
	"sync"
)

// it represents a remote node connection over the TCP established connection
type TCPPeer struct {
	conn net.Conn

	// if we dail and recives a connection then the outbound will be true
	// if we accept and recives a connection then the outbound will be false
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransport struct {
	listenAddress string
	listner       net.Listener

	mu   sync.RWMutex
	peer map[net.Addr]Peer
}

func NewTCPTransport(listenAddr string) *TCPTransport {
	return &TCPTransport{
		listenAddress: listenAddr,
	}
}
func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listner, err = net.Listen("tcp", t.listenAddress)
	if err != nil {
		return err
	}
	go t.startAcceptLoop()
	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listner.Accept()
		if err != nil {
			// Listener will close and will exit the loop
			if ne, ok := err.(*net.OpError); ok && !ne.Temporary() {
				return
			}
			fmt.Printf("TCP: acceptLoop error = %s\n", err)
			continue
		}

		go t.handleConn(conn)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	peer := NewTCPPeer(conn, true)

	fmt.Printf("New incomming connection: %v\n", peer)
}
