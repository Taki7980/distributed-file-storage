package main

import (
	"fmt"
	"log"
	"main/p2p"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NoOpHandshake,
		Decoder:       p2p.DefaultDecoder{},
	}

	tr := p2p.NewTCPTransport(tcpOpts)

	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Server listening on %s\n", tcpOpts.ListenAddr)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for msg := range tr.Consume() {
			fmt.Printf("Received message from %s: %s\n", msg.From, string(msg.Payload))
		}
	}()

	<-sigChan
	fmt.Println("\nShutting down server...")

	if err := tr.Close(); err != nil {
		log.Printf("Error closing transport: %s\n", err)
	}

	fmt.Println("Server stopped")
}
