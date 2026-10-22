package server

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestThreeNodeClusterReplicationAndFetch(t *testing.T) {
	tempRoot := t.TempDir()

	keyA := NewEncryptionKey()
	keyB := NewEncryptionKey()
	keyC := NewEncryptionKey()

	nodeA := NewFileServer(FileServerOpts{
		ListenAddr: ":5001",
		Root:       filepath.Join(tempRoot, "nodeA"),
		EncKey:     keyA,
	})
	assert.Nil(t, nodeA.Start())
	defer nodeA.Stop()

	nodeB := NewFileServer(FileServerOpts{
		ListenAddr:     ":5002",
		Root:           filepath.Join(tempRoot, "nodeB"),
		BootstrapNodes: []string{":5001"},
		EncKey:         keyB,
	})
	assert.Nil(t, nodeB.Start())
	defer nodeB.Stop()

	nodeC := NewFileServer(FileServerOpts{
		ListenAddr:     ":5003",
		Root:           filepath.Join(tempRoot, "nodeC"),
		BootstrapNodes: []string{":5001", ":5002"},
		EncKey:         keyC,
	})
	assert.Nil(t, nodeC.Start())
	defer nodeC.Stop()

	// Connect mesh from A to B and C
	assert.Nil(t, nodeA.transport.Dial(":5002"))
	assert.Nil(t, nodeA.transport.Dial(":5003"))
	assert.Nil(t, nodeB.transport.Dial(":5003"))

	// Allow mesh connections to settle
	time.Sleep(500 * time.Millisecond)

	fileKey := "distributed-test-file.txt"
	fileData := []byte("hello distributed world from node A")

	// 1. Store on node A -> replicates to B and C
	err := nodeA.Store(fileKey, bytes.NewReader(fileData))
	assert.Nil(t, err)

	// Wait briefly for peer replication
	time.Sleep(300 * time.Millisecond)

	assert.True(t, nodeA.Has(fileKey), "nodeA must have the file")
	assert.True(t, nodeB.Has(fileKey), "nodeB must have replicated file")
	assert.True(t, nodeC.Has(fileKey), "nodeC must have replicated file")

	// Verify encryption on disk: B's stored raw bytes must be ciphertext, not plaintext
	rB, err := nodeB.store.Read(fileKey)
	assert.Nil(t, err)
	rawCiphertextB, err := io.ReadAll(rB)
	rB.Close()
	assert.Nil(t, err)
	assert.NotEqual(t, fileData, rawCiphertextB, "replicated file on nodeB must be encrypted ciphertext")

	// 2. Delete local copy on A only (bypassing broadcast to test network fetch)
	assert.Nil(t, nodeA.store.Delete(fileKey))
	assert.False(t, nodeA.store.Has(fileKey), "local store on A should be empty")

	// 3. Get on node A must fetch it from network (from B or C)
	r, err := nodeA.Get(fileKey)
	assert.Nil(t, err, "nodeA should fetch file from network")
	fetchedData, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(t, fileData, fetchedData)
	assert.True(t, nodeA.Has(fileKey), "nodeA should have cached the fetched file")

	// 4. Delete on node A with broadcast -> must delete from B and C
	assert.Nil(t, nodeA.Delete(fileKey))
	time.Sleep(300 * time.Millisecond)

	assert.False(t, nodeA.Has(fileKey), "nodeA should not have the file")
	assert.False(t, nodeB.Has(fileKey), "nodeB should have deleted the file")
	assert.False(t, nodeC.Has(fileKey), "nodeC should have deleted the file")

	// 5. Peer disconnect tracking: stop node C and verify node A drops it
	peersBefore := len(nodeA.Peers())
	assert.Greater(t, peersBefore, 0)
	nodeC.Stop()
	time.Sleep(300 * time.Millisecond)
	peersAfter := len(nodeA.Peers())
	assert.Less(t, peersAfter, peersBefore, "nodeA peer list should decrease after nodeC stops")
}
