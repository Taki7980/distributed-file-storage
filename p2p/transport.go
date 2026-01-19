package p2p

// peer is an interface that represents the remote node
type Peer interface {
}

// Transport is anything that handles the communication
// between the node in the network this can be like TCP UDP & websockets

type Transport interface {
	ListenAndAccept() error
}
