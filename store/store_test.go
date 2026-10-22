package store

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathTransformFunc(t *testing.T) {
	key := "pictures_best_moments"
	pathKey := CASPathTransformFunc(key)
	assert.NotEmpty(t, pathKey.PathName)
	assert.NotEmpty(t, pathKey.Filename)
	assert.NotEmpty(t, pathKey.FirstPathName())
}

func TestStoreWriteReadHasDelete(t *testing.T) {
	tempDir := t.TempDir()
	opts := StoreOpt{
		Root:              tempDir,
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)

	key := "test-key-file"
	data := []byte("some distributed storage payload bytes")

	// 1. Has should be false initially
	assert.False(t, s.Has(key))

	// 2. Write
	n, err := s.Write(key, bytes.NewReader(data))
	assert.Nil(t, err)
	assert.Equal(t, int64(len(data)), n)

	// 3. Has should be true
	assert.True(t, s.Has(key))

	// 4. Read back
	r, err := s.Read(key)
	assert.Nil(t, err)
	readBytes, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(t, data, readBytes)
	r.Close()

	// 5. Delete
	err = s.Delete(key)
	assert.Nil(t, err)

	// 6. Has should be false after delete
	assert.False(t, s.Has(key))
}

func TestStoreClear(t *testing.T) {
	tempDir := t.TempDir()
	s := NewStore(StoreOpt{Root: tempDir})

	for _, key := range []string{"key1", "key2", "key3"} {
		_, err := s.Write(key, bytes.NewReader([]byte("content")))
		assert.Nil(t, err)
		assert.True(t, s.Has(key))
	}

	assert.Nil(t, s.Clear())
	assert.False(t, s.Has("key1"))
	assert.False(t, s.Has("key2"))
	assert.False(t, s.Has("key3"))
}

func TestStoreDeleteDoesNotDeleteSiblingKeysInSameShard(t *testing.T) {
	tempDir := t.TempDir()
	opts := StoreOpt{
		Root: tempDir,
		PathTransformFunc: func(key string) PathKey {
			return PathKey{
				PathName: "common_shard",
				Filename: key,
			}
		},
	}
	s := NewStore(opts)

	key1 := "file1.dat"
	key2 := "file2.dat"

	_, err := s.Write(key1, bytes.NewReader([]byte("content 1")))
	assert.Nil(t, err)
	_, err = s.Write(key2, bytes.NewReader([]byte("content 2")))
	assert.Nil(t, err)

	assert.True(t, s.Has(key1))
	assert.True(t, s.Has(key2))

	// Delete key1 only
	assert.Nil(t, s.Delete(key1))

	// key1 should be gone, but key2 MUST still be present
	assert.False(t, s.Has(key1))
	assert.True(t, s.Has(key2))

	// Delete key2 as well
	assert.Nil(t, s.Delete(key2))
	assert.False(t, s.Has(key2))
}

