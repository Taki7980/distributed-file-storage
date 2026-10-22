package server

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCryptoRoundTrip(t *testing.T) {
	key := NewEncryptionKey()
	assert.Len(t, key, 32)

	plaintext := []byte("top secret distributed file storage data")
	ciphertext, err := encrypt(key, plaintext)
	assert.Nil(t, err)
	assert.False(t, bytes.Equal(plaintext, ciphertext))

	decrypted, err := decrypt(key, ciphertext)
	assert.Nil(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestCryptoTamperFailsAuthentication(t *testing.T) {
	key := NewEncryptionKey()
	plaintext := []byte("confidential payload content")

	ciphertext, err := encrypt(key, plaintext)
	assert.Nil(t, err)

	// Tamper with one byte in the ciphertext
	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[len(tampered)-1] ^= 0x01

	_, err = decrypt(key, tampered)
	assert.NotNil(t, err, "decrypt must fail authentication when ciphertext is tampered")
}

func TestCryptoShortBuffer(t *testing.T) {
	key := NewEncryptionKey()
	shortData := []byte("short")

	_, err := decrypt(key, shortData)
	assert.NotNil(t, err, "decrypt must fail on short buffer")
}
