// Package decrypter provides functionality to decrypt data using a combination of ECDH and AES-GCM.
package decrypter

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/beckn/beckn-onix/pkg/model"
	"golang.org/x/crypto/hkdf"

	
)

// decrypter implements the Decrypter interface and handles the decryption process.
type decrypter struct {
}

// New creates a new decrypter instance with the given configuration.
func New(ctx context.Context) (*decrypter, func() error, error) {
	return &decrypter{}, nil, nil
}

// Decrypt decrypts the given encryptedData using the provided privateKeyBase64 and publicKeyBase64.
func (d *decrypter) Decrypt(ctx context.Context, encryptedData, privateKeyBase64, publicKeyBase64 string) (string, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return "", model.NewBadReqErr(fmt.Errorf("invalid private key: %w", err))
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return "", model.NewBadReqErr(fmt.Errorf("invalid public key: %w", err))
	}

	// Decode the Base64 encoded encrypted data.
	encryptedBytes, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", model.NewBadReqErr(fmt.Errorf("failed to decode encrypted data: %w", err))
	}

	gcm, err := createAESGCM(privateKeyBytes, publicKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create AES GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedBytes) < nonceSize {
		return "", fmt.Errorf("encrypted data is too short")
	}

	nonce, ciphertext := encryptedBytes[:nonceSize], encryptedBytes[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt data: %w", err)
	}

	return string(plaintext), nil
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
