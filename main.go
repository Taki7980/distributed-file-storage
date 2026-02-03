package main

import (
	"bytes"
	"fmt"
	"log"
	"main/p2p"
	"main/store"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	storeOpts := store.StoreOpt{
		PathTransformFunc: store.DefaultPathTransformFunc,
	}
	s := store.NewStore(storeOpts)

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

			key := fmt.Sprintf("msg_%s_%d", msg.From.String(), time.Now().Unix())
			reader := bytes.NewReader(msg.Payload)

			if err := s.WriteStream(key, reader); err != nil {
				log.Printf("Error storing message: %s\n", err)
			}
		}
	}()

	<-sigChan
	fmt.Println("\nShutting down server...")

	if err := tr.Close(); err != nil {
		log.Printf("Error closing transport: %s\n", err)
	}

	fmt.Println("Server stopped")
}
