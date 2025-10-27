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

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
)

// mockLROService is a mock implementation of the lroService interface.
type mockLROService struct {
	lro *model.LRO
	err error
}

func (m *mockLROService) Get(ctx context.Context, id string) (*model.LRO, error) {
	return m.lro, m.err
}

func TestNewLROHandler_Success(t *testing.T) {
	mockService := &mockLROService{}
	handler, err := NewLROHandler(mockService)
	if err != nil {
		t.Fatalf("NewLROHandler() error = %v, wantErr nil", err)
	}
	if handler == nil {
		t.Fatal("NewLROHandler() returned nil handler with no error")
	}
	if handler.srv != mockService {
		t.Error("NewLROHandler() did not correctly assign the service")
	}
}

func TestNewLROHandler_Error(t *testing.T) {
	_, err := NewLROHandler(nil)
	if err == nil {
		t.Fatal("NewLROHandler() with nil service, error = nil, wantErr non-nil")
	}
	expectedErr := "lroService dependency is nil"
	if err.Error() != expectedErr {
		t.Errorf("NewLROHandler() error = %q, wantErr %q", err.Error(), expectedErr)
	}
}

func TestLROHandler_Get_Success(t *testing.T) {
	opID := "test-op-123"
	now := time.Now()
	validRequestJSON, _ := json.Marshal(map[string]string{"data": "req"})
	validResultJSON, _ := json.Marshal(map[string]string{"data": "res"})
	lro := &model.LRO{
		OperationID: opID,
		Type:        model.OperationTypeCreateSubscription,
		Status:      model.LROStatusApproved,
		CreatedAt:   now,
		UpdatedAt:   now,
		RequestJSON: validRequestJSON,
		ResultJSON:  validResultJSON,
	}
	srv := &mockLROService{lro: lro}

	handler, err := NewLROHandler(srv)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/operations/"+opID, nil)
	rr := httptest.NewRecorder()

	// Need to wrap the handler with chi router to correctly parse URL params
	router := chi.NewRouter()
	router.Get("/operations/{operation_id}", handler.Get)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler.Get() status code = %d, want %d. Body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	// Normalize JSON strings for comparison to avoid issues with spacing/ordering
	var got model.LRO
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)

	}
	if diff := cmp.Diff(got, *lro); diff != "" {
		t.Fatalf("getOperation() mismatch (-want +got):\n%s", diff)
	}

}

func TestLROHandler_Get_Error(t *testing.T) {
	opID := "test-op-123"
	notFoundOpID := "not-found-op"

	tests := []struct {
		name           string
		operationID    string
		srv            lroService
		wantStatusCode int
		wantResponse   model.ErrorResponse
	}{
		{
			name:           "operation not found",
			operationID:    notFoundOpID,
			srv:            &mockLROService{err: repository.ErrOperationNotFound},
			wantStatusCode: http.StatusNotFound,
			wantResponse: model.ErrorResponse{
				Error: model.Error{
					Type:    model.ErrorTypeNotFoundError,
					Code:    model.ErrorCodeOperationNotFound,
					Message: fmt.Sprintf("Operation with id %s not found.", notFoundOpID),
					Path:    "",
				},
			},
		},
		{
			name:           "internal server error from service",
			operationID:    opID,
			srv:            &mockLROService{err: errors.New("some internal service error")},
			wantStatusCode: http.StatusInternalServerError,
			wantResponse: model.ErrorResponse{
				Error: model.Error{
					Type:    model.ErrorTypeInternalError,
					Code:    model.ErrorCodeInternalServerError,
					Message: "Failed to retrieve operation status due to an internal error.",
					Path:    "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewLROHandler(tt.srv)
			if err != nil {
				t.Fatalf("Failed to create handler for test %s: %v", tt.name, err)
			}

			req := httptest.NewRequest(http.MethodGet, "/operations/"+tt.operationID, nil)
			rr := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Get("/operations/{operation_id}", handler.Get)
			router.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("handler.Get() status code = %d, want %d. Body: %s", rr.Code, tt.wantStatusCode, rr.Body.String())
			}

			var gotResponse model.ErrorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &gotResponse); err != nil {
				t.Fatalf("Failed to unmarshal error response body: %v. Body: %s", err, rr.Body.String())
			}
			if diff := cmp.Diff(tt.wantResponse, gotResponse); diff != "" {
				t.Errorf("handler.Get() response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
