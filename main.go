package main

import (
	"log"
	"main/p2p"
)

func main() {
	tcpOpts := p2p.TCPTransportOpts{
		ListnenAddr:   ":3000",
		HandshakeFunc: p2p.DumyHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	tr := p2p.NewTCPTransport(tcpOpts)
	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}
	select {}
}
