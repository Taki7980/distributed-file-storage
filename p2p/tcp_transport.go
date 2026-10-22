package p2p

import (
	"encoding/gob"
	"errors"
	"io"
	"net"
	"sync"
)

type TCPPeer struct {
	net.Conn
	outbound bool
	encMu    sync.Mutex
	enc      *gob.Encoder
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		enc:      gob.NewEncoder(conn),
	}
}

func (p *TCPPeer) Close() error {
	return p.Conn.Close()
}

func (p *TCPPeer) RemoteAddr() string {
	return p.Conn.RemoteAddr().String()
}

func (p *TCPPeer) Send(rpc RPC) error {
	p.encMu.Lock()
	defer p.encMu.Unlock()
	return p.enc.Encode(rpc)
}

type TCPTransportOpts struct {
	ListenAddr         string
	HandshakeFunc      HandshakeFunc
	OnPeer             func(Peer) error
	OnPeerDisconnect   func(Peer)
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	rpcch    chan RPC

	mu     sync.RWMutex
	peers  map[string]*TCPPeer
	closed bool
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	if opts.HandshakeFunc == nil {
		opts.HandshakeFunc = NoOpHandshake
	}
	return &TCPTransport{
		TCPTransportOpts: opts,
		peers:            make(map[string]*TCPPeer),
		rpcch:            make(chan RPC, 1024),
	}
}

func (t *TCPTransport) Addr() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.listener != nil {
		return t.listener.Addr().String()
	}
	return t.ListenAddr
}

func (t *TCPTransport) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return errors.New("transport already closed")
	}
	t.closed = true

	var err error
	if t.listener != nil {
		err = t.listener.Close()
	}

	peersToClose := make([]*TCPPeer, 0, len(t.peers))
	for _, peer := range t.peers {
		peersToClose = append(peersToClose, peer)
	}
	t.peers = make(map[string]*TCPPeer)
	t.mu.Unlock()

	for _, peer := range peersToClose {
		peer.Close()
	}

	close(t.rpcch)
	return err
}

func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConn(conn, true)
	return nil
}

func (t *TCPTransport) ListenAndAccept() error {
	ln, err := net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}
	t.listener = ln

	go t.startAcceptLoop()
	return nil
}

// Private functions below (me.txt convention: public top, private bottom)

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
			continue
		}

		go t.handleConn(conn, false)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	peer := NewTCPPeer(conn, outbound)

	defer func() {
		peer.Close()
		t.removePeer(peer)
		if t.OnPeerDisconnect != nil {
			t.OnPeerDisconnect(peer)
		}
	}()

	if err := t.HandshakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err := t.OnPeer(peer); err != nil {
			return
		}
	}

	t.addPeer(peer)

	dec := gob.NewDecoder(conn)
	for {
		var rpc RPC
		err := dec.Decode(&rpc)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			return
		}

		if len(rpc.From) == 0 {
			rpc.From = conn.RemoteAddr().String()
		}

		t.mu.RLock()
		closed := t.closed
		t.mu.RUnlock()

		if closed {
			return
		}

		func() {
			defer func() {
				_ = recover()
			}()
			t.rpcch <- rpc
		}()
	}
}

func (t *TCPTransport) addPeer(peer *TCPPeer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.peers[peer.RemoteAddr()] = peer
}

func (t *TCPTransport) removePeer(peer *TCPPeer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.peers, peer.RemoteAddr())
}
