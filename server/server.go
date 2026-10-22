package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"main/p2p"
	"main/store"
	"sync"
	"time"
)

type FileServerOpts struct {
	ListenAddr     string
	Root           string
	BootstrapNodes []string
}

type FileServer struct {
	FileServerOpts
	store     *store.Store
	transport *p2p.TCPTransport

	peerLock sync.RWMutex
	peers    map[string]p2p.Peer

	pendingLock sync.Mutex
	pendingGets map[string]chan []byte

	quitch   chan struct{}
	stopOnce sync.Once
}

func NewFileServer(opts FileServerOpts) *FileServer {
	s := &FileServer{
		FileServerOpts: opts,
		store: store.NewStore(store.StoreOpt{
			Root: opts.Root,
		}),
		peers:       make(map[string]p2p.Peer),
		pendingGets: make(map[string]chan []byte),
		quitch:      make(chan struct{}),
	}

	tcpOpts := p2p.TCPTransportOpts{
		ListenAddr:       opts.ListenAddr,
		OnPeer:           s.onPeer,
		OnPeerDisconnect: s.onPeerDisconnect,
	}
	s.transport = p2p.NewTCPTransport(tcpOpts)
	return s
}

func (s *FileServer) Start() error {
	if err := s.transport.ListenAndAccept(); err != nil {
		return err
	}

	if len(s.BootstrapNodes) > 0 {
		go s.bootstrapNetwork()
	}

	go s.loop()
	return nil
}

func (s *FileServer) Stop() error {
	var err error
	s.stopOnce.Do(func() {
		close(s.quitch)
		err = s.transport.Close()
	})
	return err
}

// shortcut: whole file in memory, no forwarding; upgrade to streaming/gossip if files exceed RAM or mesh is partial
func (s *FileServer) Store(key string, r io.Reader) error {
	var buf bytes.Buffer
	tee := io.TeeReader(r, &buf)

	if err := s.store.WriteStream(key, tee); err != nil {
		return err
	}

	rpc := p2p.RPC{
		From:    s.transport.Addr(),
		Type:    p2p.MsgStore,
		Key:     key,
		Payload: buf.Bytes(),
	}

	return s.Broadcast(rpc)
}

func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(key) {
		return s.store.Read(key)
	}

	resCh := make(chan []byte, 1)
	s.pendingLock.Lock()
	s.pendingGets[key] = resCh
	s.pendingLock.Unlock()

	defer func() {
		s.pendingLock.Lock()
		delete(s.pendingGets, key)
		s.pendingLock.Unlock()
	}()

	rpc := p2p.RPC{
		From: s.transport.Addr(),
		Type: p2p.MsgGet,
		Key:  key,
	}

	if err := s.Broadcast(rpc); err != nil {
		return nil, err
	}

	select {
	case data := <-resCh:
		if err := s.store.WriteStream(key, bytes.NewReader(data)); err != nil {
			return nil, err
		}
		return bytes.NewReader(data), nil
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout fetching file %s from network", key)
	}
}

func (s *FileServer) Delete(key string) error {
	if err := s.store.Delete(key); err != nil {
		return err
	}

	rpc := p2p.RPC{
		From: s.transport.Addr(),
		Type: p2p.MsgDelete,
		Key:  key,
	}
	return s.Broadcast(rpc)
}

func (s *FileServer) Has(key string) bool {
	return s.store.Has(key)
}

func (s *FileServer) Broadcast(rpc p2p.RPC) error {
	s.peerLock.RLock()
	defer s.peerLock.RUnlock()

	var errs []error
	for _, peer := range s.peers {
		if err := peer.Send(rpc); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *FileServer) Peers() []string {
	s.peerLock.RLock()
	defer s.peerLock.RUnlock()

	list := make([]string, 0, len(s.peers))
	for addr := range s.peers {
		list = append(list, addr)
	}
	return list
}

// Private functions below (me.txt convention: public top, private bottom)

func (s *FileServer) onPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	s.peers[p.RemoteAddr()] = p
	return nil
}

func (s *FileServer) onPeerDisconnect(p p2p.Peer) {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	delete(s.peers, p.RemoteAddr())
}

func (s *FileServer) bootstrapNetwork() {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			for i := 0; i < 5; i++ {
				if err := s.transport.Dial(addr); err == nil {
					return
				}
				time.Sleep(150 * time.Millisecond)
			}
		}(addr)
	}
}

func (s *FileServer) loop() {
	for {
		select {
		case <-s.quitch:
			return
		case rpc, ok := <-s.transport.Consume():
			if !ok {
				return
			}
			s.handleRPC(rpc)
		}
	}
}

func (s *FileServer) handleRPC(rpc p2p.RPC) {
	switch rpc.Type {
	case p2p.MsgStore:
		if err := s.store.WriteStream(rpc.Key, bytes.NewReader(rpc.Payload)); err != nil {
			log.Printf("[%s] error storing key %s: %s", s.transport.Addr(), rpc.Key, err)
		}
	case p2p.MsgGet:
		if s.store.Has(rpc.Key) {
			r, err := s.store.Read(rpc.Key)
			if err == nil {
				data, err := io.ReadAll(r)
				r.Close()
				if err == nil {
					fileRPC := p2p.RPC{
						From:    s.transport.Addr(),
						Type:    p2p.MsgFile,
						Key:     rpc.Key,
						Payload: data,
					}
					s.peerLock.RLock()
					peer, ok := s.peers[rpc.From]
					s.peerLock.RUnlock()
					if ok {
						if err := peer.Send(fileRPC); err != nil {
							log.Printf("[%s] error replying MsgFile: %s", s.transport.Addr(), err)
						}
					} else {
						_ = s.Broadcast(fileRPC)
					}
				}
			}
		}
	case p2p.MsgFile:
		s.pendingLock.Lock()
		resCh, ok := s.pendingGets[rpc.Key]
		s.pendingLock.Unlock()
		if ok {
			select {
			case resCh <- rpc.Payload:
			default:
			}
		}
	case p2p.MsgDelete:
		if err := s.store.Delete(rpc.Key); err != nil {
			log.Printf("[%s] error deleting key %s: %s", s.transport.Addr(), rpc.Key, err)
		}
	}
}
