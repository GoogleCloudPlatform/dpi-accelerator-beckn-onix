// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"

	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	becknmodel "github.com/beckn/beckn-onix/pkg/model"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// encrypter defines the methods for encryption.
type encrypter interface {
	// Encrypt encrypts the given body using the provided privateKeyBase64 and publicKeyBase64.
	Encrypt(ctx context.Context, data string, privateKeyBase64, publicKeyBase64 string) (string, error)
}

// secretManager defines the methods from secretmanager.Client that are used.
// This allows for mocking in tests.
type secretManager interface {
	CreateSecret(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error)
	AddSecretVersion(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error)
	AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
	GetSecretVersion(ctx context.Context, req *secretmanagerpb.GetSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error)
}

type encryptionService struct {
	projectID string
	sm        secretManager
	encrypter encrypter
	keyID     string
}

// New method creates a new KeyManager instance.
func NewEcryptionService(ctx context.Context, encrypter encrypter, sm secretManager, projectID, keyID string) (*encryptionService, error) {
	if projectID == "" {
		slog.ErrorContext(ctx, "projectID cannot be empty")
		return nil, fmt.Errorf("projectID cannot be empty")
	}
	if encrypter == nil {
		slog.ErrorContext(ctx, "encrypter cannot be empty")
		return nil, fmt.Errorf("encrypter cannot be nil")
	}
	if sm == nil {
		slog.ErrorContext(ctx, "secretManager cannot be empty")
		return nil, fmt.Errorf("secretManager cannot be nil")
	}
	if keyID == "" {
		slog.ErrorContext(ctx, "keyID cannot be empty")
		return nil, fmt.Errorf("keyID cannot be empty")
	}

	es := &encryptionService{
		projectID: projectID,
		sm:        sm,
		encrypter: encrypter,
		keyID:     keyID,
	}

	return es, nil
}

// Init handles creating or retrieving private keys from the secret manager.
func (es *encryptionService) Init(ctx context.Context) (string, error) {
	secretID := generateSecretID(es.keyID)
	secretName := fmt.Sprintf("projects/%s/secrets/%s", es.projectID, secretID)
	latestVersionName := fmt.Sprintf("%s/versions/latest", secretName)

	//	Try to get the secret first
	getVersionReq := &secretmanagerpb.AccessSecretVersionRequest{Name: latestVersionName}
	existingVersion, err := es.sm.AccessSecretVersion(ctx, getVersionReq)

	// Case 1: Secret and version already exist.
	if err == nil {
		slog.InfoContext(ctx, "Secret version already exists, using it.", "secretName", secretName)
		var keyData becknmodel.Keyset
		if err := json.Unmarshal(existingVersion.Payload.Data, &keyData); err != nil {
			return "", fmt.Errorf("failed to unmarshal existing secret payload: %w", err)
		}
		return keyData.EncrPublic, nil // Return the existing public key
	}

	// Case 2: Secret not found, or has no versions. This is the "first run" scenario.
	if status.Code(err) == codes.NotFound {
		slog.InfoContext(ctx, "Secret not found, creating a new secret and version.", "secretName", secretName)

		// Generate new keys
		encrPrivateKey, genErr := ecdh.X25519().GenerateKey(rand.Reader)
		if genErr != nil {
			return "", fmt.Errorf("failed to generate encryption key pair: %w", genErr)
		}

		keyData := &becknmodel.Keyset{
			UniqueKeyID: es.keyID,
			EncrPrivate: encodeBase64(encrPrivateKey.Bytes()),
			EncrPublic:  encodeBase64(encrPrivateKey.PublicKey().Bytes()),
		}

		payload, marshalErr := json.Marshal(keyData)
		if marshalErr != nil {
			return "", fmt.Errorf("failed to marshal payload: %w", marshalErr)
		}

		// Create the secret "container". We ignore "AlreadyExists" errors here.
		createSecretReq := &secretmanagerpb.CreateSecretRequest{
			Parent:   fmt.Sprintf("projects/%s", es.projectID),
			SecretId: secretID,
			Secret: &secretmanagerpb.Secret{
				Replication: &secretmanagerpb.Replication{
					Replication: &secretmanagerpb.Replication_Automatic_{
						Automatic: &secretmanagerpb.Replication_Automatic{},
					},
				},
			},
		}
		if _, createErr := es.sm.CreateSecret(ctx, createSecretReq); createErr != nil {
			if status.Code(createErr) != codes.AlreadyExists {
				return "", fmt.Errorf("failed to create secret: %w", createErr)
			}
		}

		// Add the new key as a new version.
		addVersionReq := &secretmanagerpb.AddSecretVersionRequest{
			Parent:  secretName,
			Payload: &secretmanagerpb.SecretPayload{Data: payload},
		}
		if _, addErr := es.sm.AddSecretVersion(ctx, addVersionReq); addErr != nil {
			return "", fmt.Errorf("failed to add secret version: %w", addErr)
		}

		slog.InfoContext(ctx, "Successfully created and stored new secret version.", "secretName", secretName)
		return keyData.EncrPublic, nil
	}

	// Case 3: Some other unexpected error occurred (e.g., PermissionDenied).
	return "", fmt.Errorf("failed to get secret version: %w", err)
}

// Constants for secret ID generation.
const (
	maxSecretIDLen = 255
	hashSuffixLen  = 43 // SHA-256 (32 bytes) base64url encoded, no padding.
	// Max prefix length ensures (prefix + separator + hash) <= maxSecretIDLen.
	maxPrefixLen      = maxSecretIDLen - hashSuffixLen - 1 // -1 for the separator char
	invalidCharsRegex = `[^a-zA-Z0-9_-]+`                  // Matches anything not alphanumeric, underscore, or hyphen.
)

// generateSecretID creates a Secret Manager compatible secret ID from a subscriber ID.
// It sanitizes the subscriber ID, hashes it, and combines them.
func generateSecretID(keyID string) string {
	// Sanitize the prefix.
	reg := regexp.MustCompile(invalidCharsRegex)
	sanitizedPrefix := reg.ReplaceAllString(keyID, "-")

	// Truncate if necessary, ensuring space for hash suffix and separator.
	if len(sanitizedPrefix) > maxPrefixLen {
		sanitizedPrefix = sanitizedPrefix[:maxPrefixLen]
	}

	// Generate SHA-256 hash of the original subscriberID.
	hash := sha256.Sum256([]byte(keyID))

	// Base64URL encode the hash.
	hashSuffix := encodeBase64URL(hash[:])

	secretID := fmt.Sprintf("%s_%s", sanitizedPrefix, hashSuffix)

	return secretID
}

// encodeBase64URL encodes byte data to base64url.
func encodeBase64URL(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// Encoding byte data to base64.
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// privateKey fetches private key from sercret manager.
func (es *encryptionService) privateKey(ctx context.Context) (string, error) {
	secretID := generateSecretID(es.keyID)

	secretName := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", es.projectID, secretID)

	res, err := es.sm.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretName,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", fmt.Errorf("private keys for keyID: %s not found", es.keyID)
		}
		return "", fmt.Errorf("failed to access secret version: %w", err)
	}
	var keys becknmodel.Keyset
	if err := json.Unmarshal(res.Payload.Data, &keys); err != nil {
		return "", fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	if keys.EncrPrivate == "" {
		return "", fmt.Errorf("private key not found in secret data for keyID: %s", es.keyID)
	}

	return keys.EncrPrivate, nil
}

// Encrypt encrypts the given body using the private key and the provided public key.
func (es *encryptionService) Encrypt(ctx context.Context, data string, npKey string) (string, error) {
	privateKey, err := es.privateKey(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get private key: %w", err)
	}
	encryptedData, err := es.encrypter.Encrypt(ctx, data, privateKey, npKey)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt data: %w", err)
	}
	return encryptedData, nil
}
