# Distributed File Storage (DFS)

A high-performance peer-to-peer distributed file storage system written in Go, featuring Content-Addressable Storage (CAS), SHA-1 path sharding, full mesh replication, and network file retrieval.

## Features

- **Content-Addressable Storage (CAS)**: Files are stored using cryptographic SHA-1 hashes partitioned into 5-character directory tiers (`e0c04/54727/...`) to prevent directory saturation.
- **Zero-Knowledge Encryption at Rest (AES-256-GCM)**: Replicated network files are encrypted with authenticated encryption before broadcast. Remote peers store only ciphertext, and retrieval authenticates and decrypts payloads on the owner node.
- **Cryptographic Tamper Verification**: Any corrupted or tampered ciphertext from peers fails GCM authentication check and is rejected before reaching disk or caller.
- **P2P Transport**: Persistent TCP transport with bidirectional streaming using Go's `gob` serialization.
- **Mesh Replication**: Storing a file on one node automatically broadcasts and replicates across all connected peers.
- **On-Demand Network Retrieval**: When a file is requested on a node that doesn't have it locally, the node queries the mesh, retrieves the payload, caches it locally, and serves it.
- **Distributed Deletion**: Deleting a key propagates a delete request across the cluster.
- **Interactive Node REPL**: Built-in CLI for running nodes, bootstrapping peers, and storing/retrieving files.

## Project Structure

```
.
├── main.go                 # Node entrypoint with interactive REPL
├── me.txt                  # Project conventions (public top, private bottom)
├── p2p/
│   ├── handshake.go        # Peer handshake protocol
│   ├── message.go          # RPC message specifications
│   ├── tcp_transport.go    # TCP transport and peer connection manager
│   ├── tcp_transport_test.go
│   └── transport.go        # Transport & Peer interfaces
├── server/
│   ├── crypto.go           # AES-256-GCM encryption & key generator
│   ├── crypto_test.go      # Roundtrip and tamper authentication tests
│   ├── server.go           # FileServer managing store, peers, and RPCs
│   └── server_test.go      # 3-node cluster integration test
└── store/
    ├── store.go            # CAS store with directory sharding
    └── store_test.go       # CAS and storage unit tests
```

## Running the Cluster

### Start Node 1 (Bootstrap Node)
```bash
go run main.go -listen :3000 -root ./data_3000
```

### Start Node 2 (Connected to Node 1)
```bash
go run main.go -listen :4000 -root ./data_4000 -peers :3000
```

### Start Node 3 (Connected to Node 1 and Node 2)
```bash
go run main.go -listen :5000 -root ./data_5000 -peers :3000,:4000
```

## Interactive REPL Commands

- `put <key> <file_or_text>`: Stores data in local CAS and replicates to all connected peers.
- `get <key> [out_filepath]`: Fetches data from local storage or across the network.
- `del <key>`: Deletes data locally and across the network.
- `has <key>`: Checks whether the file exists in local store.
- `peers`: Lists all connected peer addresses.
- `help`: Displays available commands.
- `quit` / `exit`: Gracefully shuts down the node.

## Running Tests

Run all unit and integration tests with the Go race detector:
```bash
go test -v -race ./...
```


