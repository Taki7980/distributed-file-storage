package p2p

import (
	"fmt"
	"io"
	"net"
	"sync"
)

// it represents a remote node connection over the TCP established connection
type TCPPeer struct {
	conn net.Conn
	addr net.Addr

	// if we dail and recives a connection then the outbound will be true
	// if we accept and recives a connection then the outbound will be false
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		addr:     conn.RemoteAddr(),
		outbound: outbound,
	}
}

type TCPTransportOpts struct {
	ListnenAddr   string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		peers:            make(map[net.Addr]Peer),
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error

	t.listener, err = net.Listen("tcp", t.ListnenAddr)
	if err != nil {
		return err
	}

	go t.startAcceptLoop()
	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			if ne, ok := err.(*net.OpError); ok && !ne.Temporary() {
				return
			}
			fmt.Printf("TCP: acceptLoop error = %s\n", err)
			continue
		}

		go t.handleConn(conn)
	}
}

type Temp struct{}

func (t *TCPTransport) handleConn(conn net.Conn) {
	defer conn.Close()

	peer := NewTCPPeer(conn, false)

	if err := t.HandshakeFunc(peer); err != nil {
		fmt.Printf("TCP Handshake error: %s\n", err)
		return
	}

	t.mu.Lock()
	t.peers[peer.addr] = peer
	t.mu.Unlock()

	msg := &Message{}

	for {
		if err := t.Decoder.Decode(conn, msg); err != nil {
			if err == io.EOF {
				fmt.Println("connection closed by peer:", peer.addr)
			} else {
				fmt.Printf("TCP read error: %s\n", err)
			}
			return
		}

		msg.From = peer.addr
		fmt.Printf("message: %v\n", msg)
	}

}
