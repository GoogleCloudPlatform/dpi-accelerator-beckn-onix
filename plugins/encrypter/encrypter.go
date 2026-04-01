// Package encrypter provides functionality to encrypt data using AES-GCM.
// It derives the encryption key using ECDH over X25519 and HKDF from provided private and public keys.
package encrypter

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/beckn/beckn-onix/pkg/model"
	"golang.org/x/crypto/hkdf"

	
)

// encrypter implements the Encrypter interface and handles the encryption process.
type encrypter struct {
}

// New creates a new encrypter instance with the given configuration.
func New(ctx context.Context) (*encrypter, func() error, error) {
	return &encrypter{}, nil, nil
}

func (e *encrypter) Encrypt(ctx context.Context, data string, privateKeyBase64, publicKeyBase64 string) (string, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return "", model.NewBadReqErr(fmt.Errorf("invalid private key: %w", err))
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return "", model.NewBadReqErr(fmt.Errorf("invalid public key: %w", err))
	}

	gcm, err := createAESGCM(privateKeyBytes, publicKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create AES GCM cipher: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	plaintext := []byte(data)
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	encryptedData := append(nonce, ciphertext...)

	return base64.StdEncoding.EncodeToString(encryptedData), nil
}

func createAESGCM(privateKey, publicKey []byte) (cipher.AEAD, error) {
	x25519Curve := ecdh.X25519()
	x25519PrivateKey, err := x25519Curve.NewPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key: %w", err)
	}
	x25519PublicKey, err := x25519Curve.NewPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create public key: %w", err)
	}
	sharedSecret, err := x25519PrivateKey.ECDH(x25519PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive shared secret: %w", err)
	}

	// Use HKDF to derive a key from the shared secret.
	hkdfReader := hkdf.New(sha256.New, sharedSecret, nil, []byte("beckn-onix-encryption"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return gcm, nil
}
