# P2P Network Transport Layer

<div align="center">
  <img src="./images/mio.gif" alt="Overwhelmed by networking" width="400"/>
  <p><i>Me trying to understand network programming before building this</i></p>
</div>

## What's This About?

Basically, I'm building the plumbing for computers to talk to each other. You know how apps like BitTorrent or blockchain nodes chat with each other without a central server? This is that - the boring but essential part that handles connections, reads messages, and makes sure everything doesn't explode when 10 computers try to connect at once.

## Features

- **TCP Transport**: Handles network connections between peers using TCP protocol
- **Flexible Message Decoding**: Choose how your messages are decoded (GOB format or raw bytes)
- **Handshake Support**: Verify and authenticate peers before accepting their connections
- **Concurrent Connection Handling**: Manages multiple peer connections at the same time
- **Simple API**: Easy-to-use interface for setting up network nodes

## How It Works

1. **Start a Transport**: Create a TCP transport that listens on a specific address (like `:3000`)
2. **Accept Connections**: The transport automatically accepts incoming connections from other peers
3. **Handshake**: When a peer connects, perform a handshake to verify the connection
4. **Receive Messages**: Decode and process messages from connected peers

## Quick Start

```go
package main

import (
    "log"
    "main/p2p"
)

func main() {
    // Configure your transport
    tcpOpts := p2p.TCPTransportOpts{
        ListnenAddr:   ":3000",
        HandshakeFunc: p2p.DumyHandshakeFunc,
        Decoder:       p2p.DefaultDecoder{},
    }

    // Create and start the transport
    tr := p2p.NewTCPTransport(tcpOpts)
    if err := tr.ListenAndAccept(); err != nil {
        log.Fatal(err)
    }

    // Keep the server running
    select {}
}
```

## Project Structure

```
.
├── main.go                 # Example usage
├── images/
│   └── mio.gif            # Essential project documentation
└── p2p/
    ├── transport.go        # Transport and Peer interfaces
    ├── tcp_transport.go    # TCP implementation
    ├── message.go          # Message structure
    ├── decoding.go         # Message decoders
    └── handshake.go        # Handshake functions
```

## Components

### Message

A simple container for data being sent between peers. Contains the sender's address and the payload (actual data).

### Decoder

Determines how raw network bytes are converted into usable messages. Two options:

- **DefaultDecoder**: Reads raw bytes (simple and fast)
- **GOBDecoder**: Uses Go's GOB encoding (good for Go-to-Go communication)

### Handshake

A function that runs when peers connect, allowing you to verify or authenticate connections before accepting them.

## Running Tests

```bash
go test ./p2p
```

## Next Steps

This is a foundational layer. To build a complete P2P application, you'd want to add:

- Message routing logic
- Peer discovery mechanisms
- Protocol definitions for your specific use case
- Error recovery and reconnection handling

## License
