package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"main/server"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	listenAddr := flag.String("listen", ":3000", "Listen address for this node (e.g. :3000)")
	root := flag.String("root", "./data_3000", "Storage root directory for this node")
	peersFlag := flag.String("peers", "", "Comma-separated bootstrap peers (e.g. :4000,:5000)")
	flag.Parse()

	var bootstrapNodes []string
	if len(*peersFlag) > 0 {
		for _, p := range strings.Split(*peersFlag, ",") {
			p = strings.TrimSpace(p)
			if len(p) > 0 {
				bootstrapNodes = append(bootstrapNodes, p)
			}
		}
	}

	// shortcut: key lives only in memory, so network copies can't be recovered after a restart; upgrade to a key file or KMS for persistence.
	fs := server.NewFileServer(server.FileServerOpts{
		ListenAddr:     *listenAddr,
		Root:           *root,
		BootstrapNodes: bootstrapNodes,
		EncKey:         server.NewEncryptionKey(),
	})

	if err := fs.Start(); err != nil {
		log.Fatalf("Failed to start file server: %s", err)
	}

	fmt.Printf("FileServer running on %s (root: %s)\n", *listenAddr, *root)
	if len(bootstrapNodes) > 0 {
		fmt.Printf("Bootstrapping to peers: %v\n", bootstrapNodes)
	}
	printHelp()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	replQuit := make(chan struct{})
	go startREPL(fs, replQuit)

	select {
	case <-sigChan:
		fmt.Println("\nReceived shutdown signal...")
	case <-replQuit:
		fmt.Println("\nExiting REPL...")
	}

	if err := fs.Stop(); err != nil {
		log.Printf("Error stopping server: %s\n", err)
	}
	fmt.Println("Server stopped successfully")
}

// Private functions below (me.txt convention: public top, private bottom)

func printHelp() {
	fmt.Println("\nCommands:")
	fmt.Println("  put <key> <file_or_text>  - Store data locally and replicate to peers")
	fmt.Println("  get <key> [out_filepath]  - Retrieve data (local or from network)")
	fmt.Println("  del <key>                 - Delete data locally and across network")
	fmt.Println("  has <key>                 - Check if key exists in local store")
	fmt.Println("  peers                     - List connected peers")
	fmt.Println("  help                      - Show command help")
	fmt.Println("  exit | quit               - Stop node and exit")
}

func startREPL(fs *server.FileServer, quit chan struct{}) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			close(quit)
			return
		}

		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}

		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "exit", "quit":
			close(quit)
			return
		case "help":
			printHelp()
		case "peers":
			peers := fs.Peers()
			fmt.Printf("Connected peers (%d): %v\n", len(peers), peers)
		case "has":
			if len(parts) < 2 {
				fmt.Println("Usage: has <key>")
				continue
			}
			key := parts[1]
			fmt.Printf("Has %s: %v\n", key, fs.Has(key))
		case "put":
			if len(parts) < 3 {
				fmt.Println("Usage: put <key> <file_or_text>")
				continue
			}
			key := parts[1]
			value := strings.Join(parts[2:], " ")

			var r io.Reader
			if fileData, err := os.ReadFile(value); err == nil {
				r = bytes.NewReader(fileData)
				fmt.Printf("Read %d bytes from file: %s\n", len(fileData), value)
			} else {
				r = strings.NewReader(value)
			}

			if err := fs.Store(key, r); err != nil {
				fmt.Printf("Error storing %s: %s\n", key, err)
			} else {
				fmt.Printf("Successfully stored and replicated: %s\n", key)
			}
		case "get":
			if len(parts) < 2 {
				fmt.Println("Usage: get <key> [out_filepath]")
				continue
			}
			key := parts[1]
			r, err := fs.Get(key)
			if err != nil {
				fmt.Printf("Error getting %s: %s\n", key, err)
				continue
			}

			data, err := io.ReadAll(r)
			if err != nil {
				fmt.Printf("Error reading data for %s: %s\n", key, err)
				continue
			}

			if len(parts) >= 3 {
				outPath := parts[2]
				if err := os.WriteFile(outPath, data, 0644); err != nil {
					fmt.Printf("Error writing output file %s: %s\n", outPath, err)
				} else {
					fmt.Printf("Saved %d bytes to %s\n", len(data), outPath)
				}
			} else {
				fmt.Printf("Key: %s\nContent: %s\n", key, string(data))
			}
		case "del":
			if len(parts) < 2 {
				fmt.Println("Usage: del <key>")
				continue
			}
			key := parts[1]
			if err := fs.Delete(key); err != nil {
				fmt.Printf("Error deleting %s: %s\n", key, err)
			} else {
				fmt.Printf("Successfully deleted %s locally and across network\n", key)
			}
		default:
			fmt.Printf("Unknown command: %s. Type 'help' for options.\n", cmd)
		}
	}
}
