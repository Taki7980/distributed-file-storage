package p2p

type Peer interface {
	Close() error
	RemoteAddr() string
}

type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}
