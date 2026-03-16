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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"

	"github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockEncrypter is a mock implementation of encrypter.
type mockEncrypter struct {
	encryptData string
	encryptErr  error
}

func (m *mockEncrypter) Encrypt(ctx context.Context, data string, privateKeyBase64, publicKeyBase64 string) (string, error) {
	return m.encryptData, m.encryptErr
}

// mockSecretManager is a mock implementation of secretManager.
type mockSecretManager struct {
	createSecretResp        *secretmanagerpb.Secret
	createSecretErr         error
	addSecretVersionResp    *secretmanagerpb.SecretVersion
	addSecretVersionErr     error
	accessSecretVersionResp *secretmanagerpb.AccessSecretVersionResponse
	accessSecretVersionErr  error // For AccessSecretVersion
	getSecretVersionResp    *secretmanagerpb.SecretVersion
	getSecretVersionErr     error // For GetSecretVersion

	createSecretCalledWith        *secretmanagerpb.CreateSecretRequest
	addSecretVersionCalledWith    *secretmanagerpb.AddSecretVersionRequest
	accessSecretVersionCalledWith *secretmanagerpb.AccessSecretVersionRequest
	getSecretVersionCalledWith    *secretmanagerpb.GetSecretVersionRequest
}

func (m *mockSecretManager) CreateSecret(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error) {
	m.createSecretCalledWith = req
	return m.createSecretResp, m.createSecretErr
}

func (m *mockSecretManager) AddSecretVersion(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error) {
	m.addSecretVersionCalledWith = req
	return m.addSecretVersionResp, m.addSecretVersionErr
}

func (m *mockSecretManager) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	m.accessSecretVersionCalledWith = req
	return m.accessSecretVersionResp, m.accessSecretVersionErr
}

func (m *mockSecretManager) GetSecretVersion(ctx context.Context, req *secretmanagerpb.GetSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error) {
	m.getSecretVersionCalledWith = req
	return m.getSecretVersionResp, m.getSecretVersionErr
}

// Helper to generate the expected secret name in Secret Manager
func getExpectedSecretName(projectID, keyID string) string {
	return fmt.Sprintf("projects/%s/secrets/%s", projectID, generateSecretID(keyID))
}

func TestEncryptionService_Init_Success(t *testing.T) {
	ctx := context.Background()
	defaultProjectID := "test-project"
	defaultKeyID := "test-key"
	mockEnc := &mockEncrypter{}
	tests := []struct {
		name                       string
		projectID                  string
		keyID                      string
		enc                        encrypter
		configureSM                func(*mockSecretManager)
		wantErrMsg                 string
		expectService              bool
		expectCreateSecretCall     bool
		expectAddSecretVersionCall bool
	}{
		{
			name:      "success - new key created",
			projectID: defaultProjectID,
			keyID:     defaultKeyID,
			enc:       mockEnc,
			configureSM: func(msm *mockSecretManager) { // Simulate AccessSecretVersion returning NotFound first
				msm.accessSecretVersionErr = status.Error(codes.NotFound, "secret not found")
				msm.createSecretResp = &secretmanagerpb.Secret{}            // Then CreateSecret succeeds
				msm.addSecretVersionResp = &secretmanagerpb.SecretVersion{} // Then AddSecretVersion succeeds
			},
			expectService:              true,
			expectCreateSecretCall:     true,
			expectAddSecretVersionCall: true,
		},
		{
			name:      "success - key already exists",
			projectID: defaultProjectID,
			keyID:     defaultKeyID,
			enc:       mockEnc,
			configureSM: func(msm *mockSecretManager) { // Simulate AccessSecretVersion succeeding
				msm.accessSecretVersionResp = &secretmanagerpb.AccessSecretVersionResponse{
					Payload: &secretmanagerpb.SecretPayload{Data: []byte(`{"EncrPublic":"existing-pub-key"}`)},
				}
				// CreateSecret will not be called in this path.
			},
			expectService:              true,
			expectCreateSecretCall:     false,
			expectAddSecretVersionCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSMInstance := &mockSecretManager{}
			if tt.configureSM != nil {
				tt.configureSM(mockSMInstance)
			}

			service, err := NewEcryptionService(ctx, tt.enc, mockSMInstance, tt.projectID, tt.keyID)
			if err != nil {
				t.Fatalf("NewEcryptionService() failed unexpectedly: %v", err)
			}

			// Call the Init method to test the initialization logic
			publicKey, err := service.Init(ctx)

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("Init() error = nil, wantErrMsg %q", tt.wantErrMsg)
				}
				if err.Error() != tt.wantErrMsg {
					t.Errorf("Init() error = %q, wantErrMsg %q", err.Error(), tt.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("Init() unexpected error = %v", err)
				}
				if publicKey == "" {
					t.Error("Init() returned an empty public key on success")
				}
			}

			if tt.expectService {
				if service == nil {
					t.Fatal("NewEcryptionService() service is nil, want non-nil")
				}
				if service.projectID != tt.projectID {
					t.Errorf("NewEcryptionService() service.projectID = %s, want %s", service.projectID, tt.projectID)
				}
				if service.keyID != tt.keyID {
					t.Errorf("NewEcryptionService() service.keyID = %s, want %s", service.keyID, tt.keyID)
				}
				if service.encrypter != tt.enc {
					t.Error("NewEcryptionService() service.encrypter not set correctly")
				}
				if service.sm != mockSMInstance {
					t.Error("NewEcryptionService() service.sm not set correctly")
				}
			} else {
				if service != nil {
					t.Errorf("NewEcryptionService() service = %v, want nil", service)
				}
			}

			if tt.expectCreateSecretCall {
				if mockSMInstance.createSecretCalledWith == nil {
					t.Error("Expected CreateSecret to be called, but it was not")
				} else {
					expectedParent := fmt.Sprintf("projects/%s", tt.projectID)
					if mockSMInstance.createSecretCalledWith.Parent != expectedParent {
						t.Errorf("CreateSecret called with Parent %s, want %s", mockSMInstance.createSecretCalledWith.Parent, expectedParent)
					}
					if mockSMInstance.createSecretCalledWith.SecretId != generateSecretID(tt.keyID) {
						t.Errorf("CreateSecret called with SecretId %s, want %s", mockSMInstance.createSecretCalledWith.SecretId, generateSecretID(tt.keyID))
					}
				}
			} else {
				if mockSMInstance.createSecretCalledWith != nil {
					t.Error("Expected CreateSecret NOT to be called, but it was")
				}
			}

			if tt.expectAddSecretVersionCall {
				if mockSMInstance.addSecretVersionCalledWith == nil {
					t.Error("Expected AddSecretVersion to be called, but it was not")
				} else {
					expectedSecretName := getExpectedSecretName(tt.projectID, tt.keyID)
					if mockSMInstance.addSecretVersionCalledWith.Parent != expectedSecretName {
						t.Errorf("AddSecretVersion called with Parent %s, want %s", mockSMInstance.addSecretVersionCalledWith.Parent, expectedSecretName)
					}
				}
			} else {
				if mockSMInstance.addSecretVersionCalledWith != nil {
					t.Error("Expected AddSecretVersion NOT to be called, but it was")
				}
			}
		})
	}
}

func TestNewEcryptionService(t *testing.T) {
	ctx := context.Background()
	mockEnc := &mockEncrypter{}
	mockSM := &mockSecretManager{}

	tests := []struct {
		name      string
		projectID string
		keyID     string
		enc       encrypter
		sm        secretManager
		wantErr   string
	}{
		{"success", "proj", "key", mockEnc, mockSM, ""},
		{"error - projectID empty", "", "key", mockEnc, mockSM, "projectID cannot be empty"},
		{"error - keyID empty", "proj", "", mockEnc, mockSM, "keyID cannot be empty"},
		{"error - encrypter nil", "proj", "key", nil, mockSM, "encrypter cannot be nil"},
		{"error - secretManager nil", "proj", "key", mockEnc, nil, "secretManager cannot be nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEcryptionService(ctx, tt.enc, tt.sm, tt.projectID, tt.keyID)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("NewEcryptionService() error = %v, want error containing %q", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("NewEcryptionService() unexpected error = %v", err)
			}
		})
	}
}

func TestEncryptionService_Init_Error(t *testing.T) {
	ctx := context.Background()
	defaultProjectID := "test-project"
	defaultKeyID := "test-key"
	mockEnc := &mockEncrypter{}

	tests := []struct {
		name        string
		projectID   string
		keyID       string
		enc         encrypter
		configureSM func(*mockSecretManager)
		wantErrMsg  string
	}{
		{
			name:      "error - CreateSecret fails (not AlreadyExists)",
			projectID: defaultProjectID,
			keyID:     defaultKeyID,
			enc:       mockEnc,
			configureSM: func(msm *mockSecretManager) {
				msm.accessSecretVersionErr = status.Error(codes.NotFound, "secret not found")
				msm.createSecretErr = errors.New("generic create error")
			},
			wantErrMsg: "failed to create secret: generic create error",
		},
		{
			name:      "error - AccessSecretVersion fails with non-NotFound error",
			projectID: defaultProjectID,
			keyID:     defaultKeyID,
			enc:       mockEnc,
			configureSM: func(msm *mockSecretManager) {
				msm.accessSecretVersionErr = status.Error(codes.PermissionDenied, "permission denied")
			},
			wantErrMsg: "failed to get secret version: rpc error: code = PermissionDenied desc = permission denied",
		},
		{
			name:      "error - AddSecretVersion fails",
			projectID: defaultProjectID,
			keyID:     defaultKeyID,
			enc:       mockEnc,
			configureSM: func(msm *mockSecretManager) {
				msm.accessSecretVersionErr = status.Error(codes.NotFound, "secret not found")
				msm.createSecretResp = &secretmanagerpb.Secret{}
				msm.addSecretVersionErr = errors.New("generic add version error")
			},
			wantErrMsg: "failed to add secret version: generic add version error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSMInstance := &mockSecretManager{}
			if tt.configureSM != nil {
				tt.configureSM(mockSMInstance)
			}
			service, err := NewEcryptionService(ctx, tt.enc, mockSMInstance, tt.projectID, tt.keyID)
			if err != nil {
				t.Fatalf("NewEcryptionService() failed unexpectedly: %v", err)
			}

			_, err = service.Init(ctx)

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("Init() error = nil, wantErrMsg %q", tt.wantErrMsg)
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("Init() error = %q, want error containing %q", err.Error(), tt.wantErrMsg)
				}
			}
		})
	}
}

func TestEncryptionService_Encrypt_Success(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	keyID := "test-key"
	testData := "plain text data"
	npKey := "network-participant-public-key"
	expectedEncryptedData := "encrypted_data_mock"

	newTestEncryptionService := func(sm secretManager, enc encrypter) *encryptionService {
		return &encryptionService{
			projectID: projectID,
			keyID:     keyID,
			sm:        sm,
			encrypter: enc,
		}
	}

	validPrivateKeyData := map[string]string{
		"UniqueKeyID": keyID,
		"EncrPrivate": "base64-encoded-private-key",
		"EncrPublic":  "base64-encoded-public-key",
	}
	validPrivateKeyPayload, _ := json.Marshal(validPrivateKeyData)

	t.Run("success", func(t *testing.T) {
		mockSM := &mockSecretManager{}
		mockEnc := &mockEncrypter{}
		setupMocks := func(msm *mockSecretManager, menc *mockEncrypter) {
			msm.accessSecretVersionResp = &secretmanagerpb.AccessSecretVersionResponse{
				Payload: &secretmanagerpb.SecretPayload{Data: validPrivateKeyPayload},
			}
			menc.encryptData = expectedEncryptedData
		}
		setupMocks(mockSM, mockEnc)

		service := newTestEncryptionService(mockSM, mockEnc)
		encryptedData, err := service.Encrypt(ctx, testData, npKey)

		if err != nil {
			t.Fatalf("Encrypt() unexpected error = %v", err)
		}
		if encryptedData != expectedEncryptedData {
			t.Errorf("Encrypt() encryptedData = %q, want %q", encryptedData, expectedEncryptedData)
		}

		if mockSM.accessSecretVersionCalledWith == nil {
			t.Error("Expected AccessSecretVersion to be called")
		} else {
			expectedName := fmt.Sprintf("%s/versions/latest", getExpectedSecretName(projectID, keyID))
			if mockSM.accessSecretVersionCalledWith.Name != expectedName {
				t.Errorf("AccessSecretVersion called with Name %s, want %s", mockSM.accessSecretVersionCalledWith.Name, expectedName)
			}
		}
	})
}

func TestEncryptionService_Encrypt_Error(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	keyID := "test-key"
	testData := "plain text data"
	npKey := "network-participant-public-key"

	newTestEncryptionService := func(sm secretManager, enc encrypter) *encryptionService {
		return &encryptionService{
			projectID: projectID,
			keyID:     keyID,
			sm:        sm,
			encrypter: enc,
		}
	}

	validPrivateKeyData := map[string]string{
		"UniqueKeyID": keyID,
		"EncrPrivate": "base64-encoded-private-key",
		"EncrPublic":  "base64-encoded-public-key",
	}
	validPrivateKeyPayload, _ := json.Marshal(validPrivateKeyData)

	tests := []struct {
		name                  string
		dataToEncrypt         string
		networkParticipantKey string
		setupMocks            func(msm *mockSecretManager, menc *mockEncrypter)
		wantErrMsg            string
	}{
		{
			name:                  "error - privateKey fails - AccessSecretVersion returns NotFound",
			dataToEncrypt:         testData,
			networkParticipantKey: npKey,
			setupMocks: func(msm *mockSecretManager, menc *mockEncrypter) {
				msm.accessSecretVersionErr = status.Error(codes.NotFound, "secret not found")
			},
			wantErrMsg: fmt.Sprintf("failed to get private key: private keys for keyID: %s not found", keyID),
		},
		{
			name:                  "error - privateKey fails - AccessSecretVersion returns generic error",
			dataToEncrypt:         testData,
			networkParticipantKey: npKey,
			setupMocks: func(msm *mockSecretManager, menc *mockEncrypter) {
				msm.accessSecretVersionErr = errors.New("sm access error")
			},
			wantErrMsg: "failed to get private key: failed to access secret version: sm access error",
		},
		{
			name:                  "error - privateKey fails - json.Unmarshal error",
			dataToEncrypt:         testData,
			networkParticipantKey: npKey,
			setupMocks: func(msm *mockSecretManager, menc *mockEncrypter) {
				msm.accessSecretVersionResp = &secretmanagerpb.AccessSecretVersionResponse{
					Payload: &secretmanagerpb.SecretPayload{Data: []byte("invalid json")},
				}
			},
			wantErrMsg: "failed to get private key: failed to unmarshal payload: invalid character 'i' looking for beginning of value",
		},
		{
			name:                  "error - privateKey fails - privateKey not found in secret data",
			dataToEncrypt:         testData,
			networkParticipantKey: npKey,
			setupMocks: func(msm *mockSecretManager, menc *mockEncrypter) {
				invalidKeyData := map[string]string{"someOtherKey": "value"}
				payload, _ := json.Marshal(invalidKeyData)
				msm.accessSecretVersionResp = &secretmanagerpb.AccessSecretVersionResponse{
					Payload: &secretmanagerpb.SecretPayload{Data: payload},
				}
			},
			wantErrMsg: fmt.Sprintf("failed to get private key: private key not found in secret data for keyID: %s", keyID),
		},
		{
			name:                  "error - encrypter.Encrypt fails",
			dataToEncrypt:         testData,
			networkParticipantKey: npKey,
			setupMocks: func(msm *mockSecretManager, menc *mockEncrypter) {
				msm.accessSecretVersionResp = &secretmanagerpb.AccessSecretVersionResponse{
					Payload: &secretmanagerpb.SecretPayload{Data: validPrivateKeyPayload},
				}
				menc.encryptErr = errors.New("encryption failed")
			},
			wantErrMsg: "failed to encrypt data: encryption failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := &mockSecretManager{}
			mockEnc := &mockEncrypter{}
			if tt.setupMocks != nil {
				tt.setupMocks(mockSM, mockEnc)
			}

			service := newTestEncryptionService(mockSM, mockEnc)
			_, err := service.Encrypt(ctx, tt.dataToEncrypt, tt.networkParticipantKey)

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("Encrypt() error = nil, wantErrMsg %q", tt.wantErrMsg)
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("Encrypt() error = %q, wantErrMsg %q", err.Error(), tt.wantErrMsg)
				}
			}

			// Verify AccessSecretVersion was called for errors that occur after it
			if tt.wantErrMsg == "" || tt.wantErrMsg == "failed to encrypt data: encryption failed" {
				if mockSM.accessSecretVersionCalledWith == nil {
					t.Error("Expected AccessSecretVersion to be called")
				} else {
					expectedName := fmt.Sprintf("%s/versions/latest", getExpectedSecretName(projectID, keyID))
					if mockSM.accessSecretVersionCalledWith.Name != expectedName {
						t.Errorf("AccessSecretVersion called with Name %s, want %s", mockSM.accessSecretVersionCalledWith.Name, expectedName)
					}
				}
			}
		})
	}
}

func TestEncodeBase64(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "empty string",
			input: []byte(""),
			want:  "",
		},
		{
			name:  "simple string",
			input: []byte("hello"),
			want:  "aGVsbG8=",
		},
		{
			name:  "string with spaces",
			input: []byte("hello world"),
			want:  "aGVsbG8gd29ybGQ=",
		},
		{
			name:  "string with numbers and symbols",
			input: []byte("123!@#$"),
			want:  "MTIzIUAjJA==",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodeBase64(tt.input); got != tt.want {
				t.Errorf("encodeBase64() = %v, want %v", got, tt.want)
			}
		})
	}
}
