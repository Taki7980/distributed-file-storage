package p2p

type HandshakeFunc func(Peer) error

func DumyHandshakeFunc(Peer) error { return nil }
