package p2p

type Peer interface {
	Close() error
	RemoteAddr() string
	Send(RPC) error
}

type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
	Dial(addr string) error
	Addr() string
}
