package p2p

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

type TCPPeer struct {
	conn     net.Conn
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

func (p *TCPPeer) RemoteAddr() string {
	return p.conn.RemoteAddr().String()
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	rpcch    chan RPC

	mu     sync.RWMutex
	peers  map[net.Addr]*TCPPeer
	closed bool
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		peers:            make(map[net.Addr]*TCPPeer),
		rpcch:            make(chan RPC, 1024),
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error

	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	go t.startAcceptLoop()
	return nil
}

func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

func (t *TCPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return errors.New("transport already closed")
	}

	t.closed = true

	if t.listener != nil {
		if err := t.listener.Close(); err != nil {
			return err
		}
	}

	for _, peer := range t.peers {
		if err := peer.Close(); err != nil {
			fmt.Printf("error closing peer %s: %s\n", peer.RemoteAddr(), err)
		}
	}

	close(t.rpcch)
	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			t.mu.RLock()
			closed := t.closed
			t.mu.RUnlock()

			if closed {
				return
			}

			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				fmt.Printf("TCP: temporary accept error: %s\n", err)
				continue
			}

			fmt.Printf("TCP: accept error: %s\n", err)
			return
		}

		go t.handleConn(conn, false)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var (
		err  error
		peer = NewTCPPeer(conn, outbound)
	)

	defer func() {
		if err != nil {
			peer.Close()
		}
		t.removePeer(peer)
	}()

	if err = t.HandshakeFunc(peer); err != nil {
		fmt.Printf("TCP handshake error with %s: %s\n", peer.RemoteAddr(), err)
		return
	}

	t.addPeer(peer)

	for {
		rpc := RPC{}
		err = t.Decoder.Decode(conn, &rpc)
		if err != nil {
			if err == io.EOF {
				fmt.Printf("connection closed by peer: %s\n", peer.RemoteAddr())
			} else {
				fmt.Printf("TCP decode error from %s: %s\n", peer.RemoteAddr(), err)
			}
			return
		}

		rpc.From = conn.RemoteAddr()

		t.mu.RLock()
		closed := t.closed
		t.mu.RUnlock()

		if closed {
			return
		}

		t.rpcch <- rpc
	}
}

func (t *TCPTransport) addPeer(peer *TCPPeer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.peers[peer.conn.RemoteAddr()] = peer
}

func (t *TCPTransport) removePeer(peer *TCPPeer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.peers, peer.conn.RemoteAddr())
}
