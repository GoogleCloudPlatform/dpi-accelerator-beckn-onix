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
	"errors"
	"testing"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"github.com/google/go-cmp/cmp"
)

// mockLRORepository is a mock implementation of LRORepository.
type mockLRORepository struct {
	lro *model.LRO // LRO to be returned by InsertOperation on success
	err error      // Error to be returned by InsertOperation
}

// InsertOperation mocks the database insertion of an LRO.
func (m *mockLRORepository) InsertOperation(ctx context.Context, lro *model.LRO) (*model.LRO, error) {
	return m.lro, m.err
}

func (m *mockLRORepository) GetOperation(ctx context.Context, operationID string) (*model.LRO, error) {
	return m.lro, m.err
}

func TestNewLROService_Success(t *testing.T) {
	mockRepo := &mockLRORepository{}
	service, err := NewLROService(mockRepo)

	if err != nil {
		t.Fatalf("NewLROService() error = %v, wantErr nil", err)
	}
	if service == nil {
		t.Fatal("NewLROService() returned nil service")
	}
	if service.repo != mockRepo {
		t.Errorf("NewLROService() repo not set correctly. got %v, want %v", service.repo, mockRepo)
	}
}

func TestNewLROService_Error(t *testing.T) {
	_, err := NewLROService(nil)

	if err == nil {
		t.Fatal("NewLROService() error = nil, want error for nil repository")
	}
	if err.Error() != "LRORepository cannot be nil" {
		t.Errorf("NewLROService() error message = %q, want %q", err.Error(), "LRORepository cannot be nil")
	}
}
func TestLROService_Create_Success(t *testing.T) {
	ctx := context.Background()
	inputLRO := &model.LRO{
		OperationID: "op-create-success",
		Type:        model.OperationTypeCreateSubscription,
		Status:      model.LROStatusPending,
		RequestJSON: []byte(`{"data":"test request"}`),
	}
	wantLRO := &model.LRO{
		OperationID: "op-create-success",
		Type:        model.OperationTypeCreateSubscription,
		Status:      model.LROStatusPending,
		RequestJSON: []byte(`{"data":"test request"}`),
	}

	mockRepo := &mockLRORepository{
		lro: wantLRO,
	}
	service, _ := NewLROService(mockRepo)

	createdLRO, err := service.Create(ctx, inputLRO)

	if err != nil {
		t.Fatalf("Create() error = %v, wantErr nil", err)
	}
	if diff := cmp.Diff(wantLRO, createdLRO); diff != "" {
		t.Errorf("Create() LRO mismatch (-want +got):\n%s", diff)
	}
}

func TestLROService_Create_Error(t *testing.T) {
	ctx := context.Background()
	inputLRO := &model.LRO{
		OperationID: "op-create-error",
		Type:        model.OperationTypeUpdateSubscription,
	}
	repoErr := errors.New("repository insert operation failed")

	mockRepo := &mockLRORepository{
		err: repoErr,
	}
	service, _ := NewLROService(mockRepo)

	createdLRO, err := service.Create(ctx, inputLRO)

	if !errors.Is(err, repoErr) {
		t.Fatalf("Create() error = %v, want error wrapping %v", err, repoErr)
	}
	if createdLRO != nil {
		t.Errorf("Create() createdLRO = %v, want nil", createdLRO)
	}
}

func TestLROService_Get_Success(t *testing.T) {
	ctx := context.Background()
	opID := "test-operation-id"
	now := time.Now()
	expectedLRO := &model.LRO{
		OperationID: opID,
		Status:      model.LROStatusPending,                // Use defined constants
		Type:        model.OperationTypeCreateSubscription, // Use defined constants
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	repo := &mockLRORepository{
		lro: expectedLRO,
	}
	svc, _ := NewLROService(repo)

	got, err := svc.Get(ctx, opID)
	if err != nil {
		t.Fatalf("Get() error = %v, wantErr nil", err)
	}
	if diff := cmp.Diff(expectedLRO, got); diff != "" {
		t.Errorf("Get() mismatch (-want +got):\n%s", diff)
	}
}

func TestLROService_Get_Error_NotFound(t *testing.T) {
	ctx := context.Background()
	opID := "nonexistent-id"
	expectedErr := repository.ErrOperationNotFound // Simulate a specific error

	repo := &mockLRORepository{
		lro: nil, // No LRO to return
		err: expectedErr,
	}
	svc, _ := NewLROService(repo)

	_, err := svc.Get(ctx, opID)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Get() error = %v, wantErr %v", err, expectedErr)
	}
}
