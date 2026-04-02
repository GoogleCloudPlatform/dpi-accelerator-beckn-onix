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

package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"github.com/google/go-cmp/cmp"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/doug-martin/goqu/v9"
	"github.com/lib/pq"
	"github.com/DATA-DOG/go-sqlmock"

	"cloud.google.com/go/cloudsqlconn"
)

// newMockRegistry is a helper function to set up a new registry with a mocked DB.
// It now handles the error returned by NewRegistry.
func newMockRegistry(t *testing.T) (*registry, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	r, err := NewRegistry(db)
	if err != nil {
		t.Fatalf("NewRegistry returned an error: %v", err)
	}
	return r, mock, db
}

func TestNewRegistry(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		db, _, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		r, err := NewRegistry(db)
		if err != nil {
			t.Fatalf("NewRegistry returned an error: %v", err)
		}

		if r == nil {
			t.Fatal("NewRegistry returned nil, expected a valid registry")
		}
		if r.db == nil {
			t.Error("NewRegistry returned a registry with a nil db, expected a non-nil db")
		}
	})

	t.Run("Failure_DBNil", func(t *testing.T) {
		r, err := NewRegistry(nil)

		if err == nil {
			t.Fatalf("Expected an error but got nil when passing a nil DB")
		}

		if r != nil {
			t.Fatal("Expected NewRegistry to return nil registry for a nil DB, but got a non-nil registry")
		}
		if !errors.Is(err, ErrDBNil) {
			t.Errorf("NewRegistry(nil) error = %v, wantErr %v", err, ErrDBNil)
		}
	})
}

func TestValidateLRO_Success(t *testing.T) {
	validRequestJSON, _ := json.Marshal(map[string]string{"key": "value"})
	lro := &model.LRO{OperationID: "op1", Type: "create", RequestJSON: validRequestJSON}
	err := validateLRO(lro)
	if err != nil {
		t.Errorf("validateLRO() with valid LRO returned error = %v, wantErr nil", err)
	}
}

func TestValidateLRO_Failure(t *testing.T) {
	validRequestJSON, _ := json.Marshal(map[string]string{"key": "value"})
	tests := []struct {
		name    string
		lro     *model.LRO
		wantErr error
	}{
		{name: "nil LRO", lro: nil, wantErr: ErrLROIsNil},
		{name: "missing OperationID", lro: &model.LRO{Type: "create", RequestJSON: validRequestJSON}, wantErr: ErrLROOperationIDMissing},
		{name: "missing Type", lro: &model.LRO{OperationID: "op1", RequestJSON: validRequestJSON}, wantErr: ErrLROTypeMissing},
		{name: "missing RequestJSON", lro: &model.LRO{OperationID: "op1", Type: "create"}, wantErr: ErrLRORequestJSONMissing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLRO(tt.lro)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateLRO() error = %v, wantErr %v", err, tt.wantErr)
			}
			// Ensure that for failure cases, the error is not nil
			if tt.wantErr != nil && err == nil {
				t.Errorf("validateLRO() error is nil, wantErr %v", tt.wantErr)
			}
		})
	}
}

func TestNewConnectionPool(t *testing.T) {
	// Mock the driver registration to avoid global state changes.
	originalRegisterer := pgxv5Registerer
	defer func() { pgxv5Registerer = originalRegisterer }()

	ctx := context.Background()
	validConfig := &Config{
		ConnectionName: "proj:region:inst",
		User:           "test-user",
		Name:           "test-db",
	}

	tests := []struct {
		name          string
		config        *Config
		setupMocks    func(mock sqlmock.Sqlmock)
		pgxv5RegErr   error
		wantErrMsg    string
		wantCleanup   bool
		checkDBConfig func(t *testing.T, db *sql.DB, cfg *Config)
	}{
		{
			name:   "success",
			config: validConfig,
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing()
			},
			wantCleanup: true,
		},
		{
			name: "success with all pool settings",
			config: &Config{
				ConnectionName:  "proj:region:inst",
				User:            "test-user",
				Name:            "test-db",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxIdleTime: 1 * time.Minute,
				ConnMaxLifetime: 5 * time.Minute,
			},
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing()
			},
			wantCleanup: true,
		},
		{
			name:       "error on missing ConnectionName",
			config:     &Config{User: "user", Name: "db"},
			wantErrMsg: "db.connectionName is required in config",
		},
		{
			name:       "error on missing User",
			config:     &Config{ConnectionName: "conn", Name: "db"},
			wantErrMsg: "db.user is required in config",
		},
		{
			name:       "error on missing Name",
			config:     &Config{ConnectionName: "conn", User: "user"},
			wantErrMsg: "db.name is required in config",
		},
		{
			name:        "error on driver registration",
			config:      validConfig,
			pgxv5RegErr: errors.New("driver registration failed"),
			wantErrMsg:  "pgxv5.RegisterDriver: driver registration failed",
		},
		{
			name:   "error on ping",
			config: validConfig,
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing().WillReturnError(errors.New("ping failed"))
			},
			wantErrMsg: "db.Ping failed: ping failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgxv5Registerer = func(name string, opts ...cloudsqlconn.Option) (func() error, error) {
				return func() error { return nil }, tt.pgxv5RegErr
			}

			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()
			sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) { return db, nil }
			defer func() { sqlOpen = sql.Open }()

			if tt.setupMocks != nil {
				tt.setupMocks(mock)
			}

			_, cleanup, err := NewConnectionPool(ctx, tt.config)
			if tt.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("NewConnectionPool() error = %v, want error containing %q", err, tt.wantErrMsg)
				}
			} else if err != nil {
				t.Errorf("NewConnectionPool() unexpected error = %v", err)
			}

			if tt.wantCleanup {
				if cleanup == nil {
					t.Error("NewConnectionPool() cleanup function is nil, want non-nil")
				}
			} else if cleanup != nil {
				t.Error("NewConnectionPool() cleanup function is not nil, want nil")
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

// baseTime is a fixed time for consistent testing of time fields.
var baseTime = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

func TestRegistry_Lookup_Success(t *testing.T) {
	tests := []struct {
		name         string
		filter       *model.Subscription
		mockSetup    func(sqlmock.Sqlmock, *model.Subscription)
		expectedSubs []model.Subscription
	}{
		{
			name:   "No filter, select all",
			filter: &model.Subscription{},
			mockSetup: func(mock sqlmock.Sqlmock, filter *model.Subscription) {
				rows := sqlmock.NewRows([]string{
					"subscriber_id", "url", "type", "domain", "location", "key_id",
					"signing_public_key", "encr_public_key", "valid_from", "valid_until",
					"status", "created_at", "updated_at", // Note: changed from "created", "updated" to "created_at", "updated_at"
				}).
					AddRow("sub1", "http://url1.com", "BAP", "domain1", nil, "key1", "sign1", "encr1", baseTime, baseTime.Add(time.Hour), "SUBSCRIBED", baseTime, baseTime).
					AddRow("sub2", "http://url2.com", "BPP", "domain2", nil, "key2", "sign2", "encr2", baseTime, baseTime.Add(time.Hour), "SUBSCRIBED", baseTime, baseTime)

				dataset := goqu.From(subscriptionsTableName).Select(
					"subscriber_id", "url", "type", "domain", "location", "key_id",
					"signing_public_key", "encr_public_key", "valid_from", "valid_until",
					"status", "created_at", "updated_at", // Note: changed from "created", "updated" to "created_at", "updated_at"
				)
				sqlStr, _, _ := dataset.ToSQL()

				mock.ExpectQuery(regexp.QuoteMeta(sqlStr)).
					WillReturnRows(rows)
			},
			expectedSubs: []model.Subscription{
				{
					Subscriber: model.Subscriber{SubscriberID: "sub1", URL: "http://url1.com", Type: model.RoleBAP, Domain: "domain1"},
					KeyID:      "key1", SigningPublicKey: "sign1", EncrPublicKey: "encr1", ValidFrom: baseTime, ValidUntil: baseTime.Add(time.Hour), Status: "SUBSCRIBED", Created: baseTime, Updated: baseTime,
				},
				{
					Subscriber: model.Subscriber{SubscriberID: "sub2", URL: "http://url2.com", Type: model.RoleBPP, Domain: "domain2"},
					KeyID:      "key2", SigningPublicKey: "sign2", EncrPublicKey: "encr2", ValidFrom: baseTime, ValidUntil: baseTime.Add(time.Hour), Status: "SUBSCRIBED", Created: baseTime, Updated: baseTime,
				},
			},
		},
		{
			name: "Filter by SubscriberID",
			filter: &model.Subscription{
				Subscriber: model.Subscriber{SubscriberID: "test_sub_id"},
			},
			mockSetup: func(mock sqlmock.Sqlmock, filter *model.Subscription) {
				rows := sqlmock.NewRows([]string{
					"subscriber_id", "url", "type", "domain", "location", "key_id",
					"signing_public_key", "encr_public_key", "valid_from", "valid_until",
					"status", "created_at", "updated_at",
				}).
					AddRow("test_sub_id", "http://test.com", "BAP", "test_domain", nil, "test_key", "test_sign", "test_encr", baseTime, baseTime.Add(time.Hour), "SUBSCRIBED", baseTime, baseTime)

				dataset := goqu.From(subscriptionsTableName).Select(
					"subscriber_id", "url", "type", "domain", "location", "key_id",
					"signing_public_key", "encr_public_key", "valid_from", "valid_until",
					"status", "created_at", "updated_at",
				).Where(
					buildLookupConditions(filter)...,
				)
				sqlStr, args, _ := dataset.ToSQL()

				driverArgs := make([]driver.Value, len(args))
				for i, v := range args {
					driverArgs[i] = v
				}

				mock.ExpectQuery(regexp.QuoteMeta(sqlStr)).
					WithArgs(driverArgs...).
					WillReturnRows(rows)
			},
			expectedSubs: []model.Subscription{
				{
					Subscriber: model.Subscriber{SubscriberID: "test_sub_id", URL: "http://test.com", Type: model.RoleBAP, Domain: "test_domain"},
					KeyID:      "test_key", SigningPublicKey: "test_sign", EncrPublicKey: "test_encr", ValidFrom: baseTime, ValidUntil: baseTime.Add(time.Hour), Status: "SUBSCRIBED", Created: baseTime, Updated: baseTime,
				},
			},
		},
		{
			name: "Filter by Type and Domain",
			filter: &model.Subscription{
				Subscriber: model.Subscriber{Type: model.RoleBPP, Domain: "example.com"},
			},
			mockSetup: func(mock sqlmock.Sqlmock, filter *model.Subscription) {
				rows := sqlmock.NewRows([]string{
					"subscriber_id", "url", "type", "domain", "location", "key_id",
					"signing_public_key", "encr_public_key", "valid_from", "valid_until",
					"status", "created_at", "updated_at",
				}).
					AddRow("sub3", "http://bpp.example.com", "BPP", "example.com", nil, "key3", "sign3", "encr3", baseTime, baseTime.Add(time.Hour), "SUBSCRIBED", baseTime, baseTime)

				dataset := goqu.From(subscriptionsTableName).Select(
					"subscriber_id", "url", "type", "domain", "location", "key_id",
					"signing_public_key", "encr_public_key", "valid_from", "valid_until",
					"status", "created_at", "updated_at",
				).Where(
					buildLookupConditions(filter)...,
				)
				sqlStr, args, _ := dataset.ToSQL()

				driverArgs := make([]driver.Value, len(args))
				for i, v := range args {
					driverArgs[i] = v
				}

				mock.ExpectQuery(regexp.QuoteMeta(sqlStr)).
					WithArgs(driverArgs...).
					WillReturnRows(rows)
			},
			expectedSubs: []model.Subscription{
				{
					Subscriber: model.Subscriber{SubscriberID: "sub3", URL: "http://bpp.example.com", Type: model.RoleBPP, Domain: "example.com"},
					KeyID:      "key3", SigningPublicKey: "sign3", EncrPublicKey: "encr3", ValidFrom: baseTime, ValidUntil: baseTime.Add(time.Hour), Status: "SUBSCRIBED", Created: baseTime, Updated: baseTime,
				},
			},
		},
		{
			name: "Filter by Location.City.Name",
			filter: &model.Subscription{
				Subscriber: model.Subscriber{
					Location: &model.Location{
						City: &model.City{Name: "Bengaluru"},
					},
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock, filter *model.Subscription) {
				locationJSON, _ := json.Marshal(&model.Location{City: &model.City{Name: "Bengaluru"}})
				rows := sqlmock.NewRows([]string{
					"subscriber_id", "url", "type", "domain", "location", "key_id",
					"signing_public_key", "encr_public_key", "valid_from", "valid_until",
					"status", "created_at", "updated_at",
				}).
					AddRow("sub4", "http://city.com", "BAP", "city.domain", locationJSON, "key4", "sign4", "encr4", baseTime, baseTime.Add(time.Hour), "SUBSCRIBED", baseTime, baseTime)

				dataset := goqu.From(subscriptionsTableName).Select(
					"subscriber_id", "url", "type", "domain", "location", "key_id",
					"signing_public_key", "encr_public_key", "valid_from", "valid_until",
					"status", "created_at", "updated_at",
				).Where(
					buildLookupConditions(filter)...,
				)
				sqlStr, args, _ := dataset.ToSQL()

				driverArgs := make([]driver.Value, len(args))
				for i, v := range args {
					driverArgs[i] = v
				}

				mock.ExpectQuery(regexp.QuoteMeta(sqlStr)).
					WithArgs(driverArgs...).
					WillReturnRows(rows)
			},
			expectedSubs: []model.Subscription{
				{
					Subscriber: model.Subscriber{
						SubscriberID: "sub4", URL: "http://city.com", Type: model.RoleBAP, Domain: "city.domain",
						Location: &model.Location{City: &model.City{Name: "Bengaluru"}},
					},
					KeyID: "key4", SigningPublicKey: "sign4", EncrPublicKey: "encr4", ValidFrom: baseTime, ValidUntil: baseTime.Add(time.Hour), Status: "SUBSCRIBED", Created: baseTime, Updated: baseTime,
				},
			},
		},
		{
			name: "No matching records",
			filter: &model.Subscription{
				Subscriber: model.Subscriber{SubscriberID: "non_existent_id"},
			},
			mockSetup: func(mock sqlmock.Sqlmock, filter *model.Subscription) {
				rows := sqlmock.NewRows([]string{
					"subscriber_id", "url", "type", "domain", "location", "key_id",
					"signing_public_key", "encr_public_key", "valid_from", "valid_until",
					"status", "created_at", "updated_at",
				})

				dataset := goqu.From(subscriptionsTableName).Select(
					"subscriber_id", "url", "type", "domain", "location", "key_id",
					"signing_public_key", "encr_public_key", "valid_from", "valid_until",
					"status", "created_at", "updated_at",
				).Where(
					buildLookupConditions(filter)...,
				)
				sqlStr, args, _ := dataset.ToSQL()

				driverArgs := make([]driver.Value, len(args))
				for i, v := range args {
					driverArgs[i] = v
				}

				mock.ExpectQuery(regexp.QuoteMeta(sqlStr)).
					WithArgs(driverArgs...).
					WillReturnRows(rows)
			},
			expectedSubs: []model.Subscription{}, // Expect empty slice
		},
		{
			name: "Complex filter with direct and nested location fields",
			filter: &model.Subscription{
				Subscriber: model.Subscriber{
					SubscriberID: "complex_sub",
					Type:         "BG",
					Location: &model.Location{
						Address:  "123 Main St",
						District: "Central",
						State:    &model.State{Code: "KA"},
						Country:  &model.Country{Name: "India"},
					},
				},
				Status: "INITIATED",
			},
			mockSetup: func(mock sqlmock.Sqlmock, filter *model.Subscription) {
				locationJSON, _ := json.Marshal(&model.Location{
					Address:  "123 Main St",
					District: "Central",
					State:    &model.State{Code: "KA"},
					Country:  &model.Country{Name: "India"},
				})
				rows := sqlmock.NewRows([]string{
					"subscriber_id", "url", "type", "domain", "location", "key_id",
					"signing_public_key", "encr_public_key", "valid_from", "valid_until",
					"status", "created_at", "updated_at",
				}).
					AddRow("complex_sub", "http://complex.com", "BG", "complex.domain", locationJSON, "complex_key", "complex_sign", "complex_encr", baseTime, baseTime.Add(time.Hour), "INITIATED", baseTime, baseTime)

				dataset := goqu.From(subscriptionsTableName).Select(
					"subscriber_id", "url", "type", "domain", "location", "key_id",
					"signing_public_key", "encr_public_key", "valid_from", "valid_until",
					"status", "created_at", "updated_at",
				).Where(
					buildLookupConditions(filter)...,
				)
				sqlStr, args, _ := dataset.ToSQL()

				driverArgs := make([]driver.Value, len(args))
				for i, v := range args {
					driverArgs[i] = v
				}

				mock.ExpectQuery(regexp.QuoteMeta(sqlStr)).
					WithArgs(driverArgs...).
					WillReturnRows(rows)
			},
			expectedSubs: []model.Subscription{
				{
					Subscriber: model.Subscriber{
						SubscriberID: "complex_sub",
						URL:          "http://complex.com",
						Type:         "BG",
						Domain:       "complex.domain",
						Location: &model.Location{
							Address:  "123 Main St",
							District: "Central",
							State:    &model.State{Code: "KA"},
							Country:  &model.Country{Name: "India"},
						},
					},
					KeyID: "complex_key", SigningPublicKey: "complex_sign", EncrPublicKey: "complex_encr", Status: "INITIATED", ValidFrom: baseTime, ValidUntil: baseTime.Add(time.Hour), Created: baseTime, Updated: baseTime,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, mock, db := newMockRegistry(t)
			defer db.Close()

			tt.mockSetup(mock, tt.filter)

			gotSubs, err := r.Lookup(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("Lookup() returned an unexpected error: %v", err)
			}

			// Truncate times for comparison as DB might store with different precision
			for i := range gotSubs {
				gotSubs[i].ValidFrom = gotSubs[i].ValidFrom.Truncate(time.Second)
				gotSubs[i].ValidUntil = gotSubs[i].ValidUntil.Truncate(time.Second)
				gotSubs[i].Created = gotSubs[i].Created.Truncate(time.Second)
				gotSubs[i].Updated = gotSubs[i].Updated.Truncate(time.Second)
			}
			for i := range tt.expectedSubs {
				tt.expectedSubs[i].ValidFrom = tt.expectedSubs[i].ValidFrom.Truncate(time.Second)
				tt.expectedSubs[i].ValidUntil = tt.expectedSubs[i].ValidUntil.Truncate(time.Second)
				tt.expectedSubs[i].Created = tt.expectedSubs[i].Created.Truncate(time.Second)
				tt.expectedSubs[i].Updated = tt.expectedSubs[i].Updated.Truncate(time.Second)
			}

			if !reflect.DeepEqual(gotSubs, tt.expectedSubs) {
				t.Errorf("Lookup() gotSubs mismatch:\nGot:   %+v\nWant: %+v", gotSubs, tt.expectedSubs)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRegistry_Lookup_Failure(t *testing.T) {
	tests := []struct {
		name          string
		filter        *model.Subscription
		mockSetup     func(sqlmock.Sqlmock, *model.Subscription)
		expectedError error
	}{
		{
			name: "db.SelectContext error",
			filter: &model.Subscription{
				Subscriber: model.Subscriber{SubscriberID: "error_sub"},
			},
			mockSetup: func(mock sqlmock.Sqlmock, filter *model.Subscription) {
				dataset := goqu.From(subscriptionsTableName).Select(
					"subscriber_id", "url", "type", "domain", "location", "key_id",
					"signing_public_key", "encr_public_key", "valid_from", "valid_until",
					"status", "created_at", "updated_at",
				).Where(
					buildLookupConditions(filter)...,
				)
				sqlStr, args, _ := dataset.ToSQL()

				driverArgs := make([]driver.Value, len(args))
				for i, v := range args {
					driverArgs[i] = v
				}

				mock.ExpectQuery(regexp.QuoteMeta(sqlStr)).
					WithArgs(driverArgs...).
					WillReturnError(errors.New("database connection lost"))
			},
			expectedError: errors.New("failed to execute lookup query: database connection lost"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, mock, db := newMockRegistry(t)
			defer db.Close()

			tt.mockSetup(mock, tt.filter)

			_, err := r.Lookup(context.Background(), tt.filter)

			if !errors.Is(err, tt.expectedError) && (err == nil || tt.expectedError == nil || err.Error() != tt.expectedError.Error()) {
				t.Errorf("Lookup() error = %v, wantErr %v", err, tt.expectedError)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRegistry_InsertOperation_Success(t *testing.T) {
	ctx := context.Background()
	r, mock, db := newMockRegistry(t)
	defer db.Close()

	opID := "test-op-success"
	now := time.Now().Truncate(time.Second) // Truncate for comparison
	requestJSON, _ := json.Marshal(map[string]string{"key": "value"})

	lro := &model.LRO{
		OperationID: opID,
		Status:      model.LROStatusPending,
		Type:        model.OperationTypeCreateSubscription,
		RequestJSON: requestJSON,
	}

	rows := sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now)
	mock.ExpectQuery(regexp.QuoteMeta(insertOperationQuery)).
		WithArgs(lro.OperationID, lro.Status, lro.Type, lro.RequestJSON).
		WillReturnRows(rows)

	insertedLRO, err := r.InsertOperation(ctx, lro)
	if err != nil {
		t.Fatalf("InsertOperation() returned unexpected error: %v", err)
	}

	if insertedLRO == nil {
		t.Fatal("InsertOperation() returned nil LRO")
	}

	// Check if timestamps are updated
	if insertedLRO.CreatedAt.IsZero() || insertedLRO.UpdatedAt.IsZero() {
		t.Errorf("InsertOperation() CreatedAt or UpdatedAt timestamps are zero")
	}
	if !insertedLRO.CreatedAt.Equal(now) || !insertedLRO.UpdatedAt.Equal(now) {
		t.Errorf("InsertOperation() timestamps mismatch: got CreatedAt %v, UpdatedAt %v; want %v", insertedLRO.CreatedAt, insertedLRO.UpdatedAt, now)
	}

	// Ensure other fields are as expected
	if diff := cmp.Diff(lro.OperationID, insertedLRO.OperationID); diff != "" {
		t.Errorf("OperationID mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(lro.Status, insertedLRO.Status); diff != "" {
		t.Errorf("Status mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(lro.Type, insertedLRO.Type); diff != "" {
		t.Errorf("Type mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(lro.RequestJSON, insertedLRO.RequestJSON); diff != "" {
		t.Errorf("RequestJSON mismatch (-want +got):\n%s", diff)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestRegistry_InsertOperation_Failure(t *testing.T) {
	ctx := context.Background()
	requestJSON, _ := json.Marshal(map[string]string{"key": "value"})
	validLRO := &model.LRO{
		OperationID: "test-op-fail",
		Status:      model.LROStatusPending,
		Type:        model.OperationTypeCreateSubscription,
		RequestJSON: requestJSON,
	}

	tests := []struct {
		name      string
		lro       *model.LRO
		mockSetup func(mock sqlmock.Sqlmock, lro *model.LRO)
		wantErr   error
	}{
		{
			name:    "validation failure - nil LRO",
			lro:     nil,
			wantErr: fmt.Errorf("LRO validation failed: %w", ErrLROIsNil),
		},
		{
			name:    "validation failure - missing OperationID",
			lro:     &model.LRO{Type: "create", RequestJSON: requestJSON},
			wantErr: fmt.Errorf("LRO validation failed: %w", ErrLROOperationIDMissing),
		},
		{
			name: "unique constraint violation",
			lro:  validLRO,
			mockSetup: func(mock sqlmock.Sqlmock, lro *model.LRO) {
				pqErr := &pq.Error{Code: "23505", Message: "duplicate key value violates unique constraint"}
				mock.ExpectQuery(regexp.QuoteMeta(insertOperationQuery)).
					WithArgs(lro.OperationID, lro.Status, lro.Type, lro.RequestJSON).
					WillReturnError(pqErr)
			},
			wantErr: fmt.Errorf("%w: %s", ErrOperationAlreadyExists, validLRO.OperationID),
		},
		{
			name: "other database error",
			lro:  validLRO,
			mockSetup: func(mock sqlmock.Sqlmock, lro *model.LRO) {
				mock.ExpectQuery(regexp.QuoteMeta(insertOperationQuery)).
					WithArgs(lro.OperationID, lro.Status, lro.Type, lro.RequestJSON).
					WillReturnError(errors.New("db connection lost"))
			},
			wantErr: fmt.Errorf("failed to insert operation with ID %s: %w", validLRO.OperationID, errors.New("db connection lost")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, mock, db := newMockRegistry(t)
			defer db.Close()

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.lro)
			}

			_, err := r.InsertOperation(ctx, tt.lro)

			if err == nil {
				t.Fatalf("InsertOperation() expected an error, got nil")
			}
			// Use errors.Is for wrapped errors, or check string for specific messages
			if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
				t.Errorf("InsertOperation() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRegistry_updateLRO_Failure_NotFound(t *testing.T) {
	ctx := context.Background()
	r, mock, db := newMockRegistry(t)
	defer db.Close()

	opID := "non-existent-op"
	lro := &model.LRO{OperationID: opID, Status: model.LROStatusApproved}

	// Expect transaction begin (even though we're testing an internal func, it's usually called within a tx)
	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin mock transaction: %v", err)
	}

	// Expect the update query to return sql.ErrNoRows
	mock.ExpectQuery(regexp.QuoteMeta(updateOperationQuery)).
		WithArgs(lro.OperationID, lro.Status, sql.NullString{}, sql.NullString{}, lro.RetryCount).
		WillReturnError(sql.ErrNoRows)

	err = r.updateLRO(ctx, tx, lro)
	if err == nil {
		t.Fatal("updateLRO() expected an error, got nil")
	}

	// Check if the error is the expected wrapped error
	expectedErr := fmt.Errorf("failed to update LRO %s (not found): %w", opID, ErrOperationNotFound)
	if !errors.Is(err, ErrOperationNotFound) || !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Errorf("updateLRO() error = %v, wantErr %v", err, expectedErr)
	}

	// Expect rollback if an error occurred within the transaction
	mock.ExpectRollback()
	_ = tx.Rollback() // Call rollback on the real tx to clear expectations

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestRegistry_InsertSubscription_Success(t *testing.T) {
	ctx := context.Background()
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		sub       *model.Subscription
		mockSetup func(mock sqlmock.Sqlmock, sub *model.Subscription)
	}{
		{
			name: "basic subscription insert",
			sub: &model.Subscription{
				Subscriber: model.Subscriber{
					SubscriberID: "sub-new-1",
					URL:          "http://new.com",
					Type:         model.RoleBAP,
					Domain:       "new.domain",
				},
				KeyID:            "key-new-1",
				SigningPublicKey: "sign-new-1",
				EncrPublicKey:    "encr-new-1",
				ValidFrom:        fixedTime,
				ValidUntil:       fixedTime.Add(time.Hour),
				Status:           "SUBSCRIBED",
				Nonce:            "nonce-new-1",
			},
			mockSetup: func(mock sqlmock.Sqlmock, sub *model.Subscription) {
				rows := sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(fixedTime, fixedTime)
				mock.ExpectQuery(regexp.QuoteMeta(insertOnlySubscriptionQuery)).
					WithArgs(
						sub.SubscriberID, sub.URL, sub.Type, sub.Domain, sql.NullString{}, sub.KeyID,
						sub.SigningPublicKey, sub.EncrPublicKey, sub.ValidFrom, sub.ValidUntil,
						sub.Status, sub.Nonce,
					).
					WillReturnRows(rows)
			},
		},
		{
			name: "subscription with location",
			sub: &model.Subscription{
				Subscriber: model.Subscriber{
					SubscriberID: "sub-new-2",
					URL:          "http://new2.com",
					Type:         model.RoleBPP,
					Domain:       "new2.domain",
					Location:     &model.Location{ID: "loc-1", Address: "123 Main St"},
				},
				KeyID:            "key-new-2",
				SigningPublicKey: "sign-new-2",
				EncrPublicKey:    "encr-new-2",
				ValidFrom:        fixedTime,
				ValidUntil:       fixedTime.Add(time.Hour),
				Status:           "INITIATED",
				Nonce:            "nonce-new-2",
			},
			mockSetup: func(mock sqlmock.Sqlmock, sub *model.Subscription) {
				locBytes, _ := json.Marshal(sub.Location)
				locationJSON := sql.NullString{String: string(locBytes), Valid: true}
				rows := sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(fixedTime, fixedTime)
				mock.ExpectQuery(regexp.QuoteMeta(insertOnlySubscriptionQuery)).
					WithArgs(
						sub.SubscriberID, sub.URL, sub.Type, sub.Domain, locationJSON, sub.KeyID,
						sub.SigningPublicKey, sub.EncrPublicKey, sub.ValidFrom, sub.ValidUntil,
						sub.Status, sub.Nonce,
					).
					WillReturnRows(rows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, mock, db := newMockRegistry(t)
			defer db.Close()

			tt.mockSetup(mock, tt.sub)

			insertedSub, err := r.InsertSubscription(ctx, tt.sub)
			if err != nil {
				t.Fatalf("InsertSubscription() returned unexpected error: %v", err)
			}

			// Check if timestamps are updated
			if insertedSub.Created.IsZero() || insertedSub.Updated.IsZero() {
				t.Errorf("InsertSubscription() Created or Updated timestamps are zero")
			}
			// Ensure the returned object matches the input with updated timestamps
			expectedSub := *tt.sub // Copy
			expectedSub.Created = fixedTime
			expectedSub.Updated = fixedTime

			if diff := cmp.Diff(&expectedSub, insertedSub); diff != "" {
				t.Errorf("InsertSubscription() returned subscription mismatch (-want +got):\n%s", diff)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRegistry_InsertSubscription_Failure(t *testing.T) {
	ctx := context.Background()
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	validSub := &model.Subscription{
		Subscriber: model.Subscriber{
			SubscriberID: "sub-fail",
			URL:          "http://fail.com",
			Type:         model.RoleBAP,
			Domain:       "fail.domain",
		},
		KeyID:            "key-fail",
		SigningPublicKey: "sign-fail",
		EncrPublicKey:    "encr-fail",
		ValidFrom:        fixedTime,
		ValidUntil:       fixedTime.Add(time.Hour),
		Status:           "SUBSCRIBED",
		Nonce:            "nonce-fail",
	}

	tests := []struct {
		name      string
		sub       *model.Subscription
		mockSetup func(mock sqlmock.Sqlmock, sub *model.Subscription)
		wantErr   error
	}{
		{
			name:    "validation failure - nil subscription",
			sub:     nil,
			wantErr: errors.New("subscription cannot be nil for insert"),
		},
		{
			name: "validation failure - missing SubscriberID",
			sub: &model.Subscription{
				Subscriber: model.Subscriber{
					URL:    "http://fail.com",
					Type:   model.RoleBAP,
					Domain: "fail.domain",
				},
				KeyID: "key-fail",
			},
			wantErr: errors.New("subscription SubscriberID is required"),
		},
		{
			name: "validation failure - missing KeyID",
			sub: &model.Subscription{
				Subscriber: model.Subscriber{
					SubscriberID: "sub-fail",
					URL:          "http://fail.com",
					Type:         model.RoleBAP,
					Domain:       "fail.domain",
				},
			},
			wantErr: errors.New("subscription KeyID is required"),
		},
		{
			name: "unique constraint violation",
			sub:  validSub,
			mockSetup: func(mock sqlmock.Sqlmock, sub *model.Subscription) {
				pqErr := &pq.Error{Code: "23505", Message: "duplicate key value violates unique constraint"}
				var locationJSON sql.NullString
				if sub.Location != nil {
					locBytes, _ := json.Marshal(sub.Location)
					locationJSON = sql.NullString{String: string(locBytes), Valid: true}
				} else {
					locationJSON = sql.NullString{}
				}
				mock.ExpectQuery(regexp.QuoteMeta(insertOnlySubscriptionQuery)).
					WithArgs(
						sub.SubscriberID, sub.URL, sub.Type, sub.Domain, locationJSON, sub.KeyID,
						sub.SigningPublicKey, sub.EncrPublicKey, sub.ValidFrom, sub.ValidUntil,
						sub.Status, sub.Nonce,
					).
					WillReturnError(pqErr)
			},
			wantErr: ErrSubscriptionConflict,
		},
		{
			name: "other database error",
			sub:  validSub,
			mockSetup: func(mock sqlmock.Sqlmock, sub *model.Subscription) {
				var locationJSON sql.NullString
				if sub.Location != nil {
					locBytes, _ := json.Marshal(sub.Location)
					locationJSON = sql.NullString{String: string(locBytes), Valid: true}
				} else {
					locationJSON = sql.NullString{}
				}
				mock.ExpectQuery(regexp.QuoteMeta(insertOnlySubscriptionQuery)).
					WithArgs(
						sub.SubscriberID, sub.URL, sub.Type, sub.Domain, locationJSON, sub.KeyID,
						sub.SigningPublicKey, sub.EncrPublicKey, sub.ValidFrom, sub.ValidUntil,
						sub.Status, sub.Nonce,
					).
					WillReturnError(errors.New("db connection lost"))
			},
			wantErr: errors.New("failed to insert subscription"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, mock, db := newMockRegistry(t)
			defer db.Close()

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.sub)
			}

			_, err := r.InsertSubscription(ctx, tt.sub)

			if err == nil {
				t.Fatalf("InsertSubscription() expected an error, got nil")
			}
			if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
				t.Errorf("InsertSubscription() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRegistry_UpdateOperation_Success(t *testing.T) {
	ctx := context.Background()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer mockDB.Close()

	r, err := NewRegistry(mockDB)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	opID := "test-update-op-id"
	now := time.Now().UTC() // Ensure UTC for consistency
	originalRequestJSON, _ := json.Marshal(map[string]string{"req": "original_data"})
	resultJSON, _ := json.Marshal(map[string]string{"res": "updated_result"})
	errorDataJSONInput, _ := json.Marshal(map[string]string{"err": "update_error_detail"})

	lroToUpdate := &model.LRO{
		OperationID:   opID,
		Status:        model.LROStatusApproved,
		ResultJSON:    resultJSON,
		ErrorDataJSON: errorDataJSONInput, // Set ErrorDataJSON for the test
		RetryCount:    1,
		// UpdatedAt will be set by the method
	}

	// These are the fields returned by RETURNING in the query
	expectedCreatedAt := now.Add(-time.Hour) // Simulate it was created earlier
	expectedUpdatedAt := now                 // Simulate DB updated time
	expectedType := model.OperationTypeCreateSubscription
	expectedResultSQLNullString := sql.NullString{String: string(resultJSON), Valid: true}
	expectedErrorSQLNullString := sql.NullString{String: string(errorDataJSONInput), Valid: true}

	mock.ExpectQuery(regexp.QuoteMeta(updateOperationQuery)).
		WithArgs(lroToUpdate.OperationID, lroToUpdate.Status, expectedResultSQLNullString, expectedErrorSQLNullString, lroToUpdate.RetryCount). // Corrected WithArgs
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at", "type", "request_json"}).                                           // Added updated_at
																			AddRow(expectedCreatedAt, expectedUpdatedAt, expectedType, originalRequestJSON))

	updatedLRO, err := r.UpdateOperation(ctx, lroToUpdate)
	if err != nil {
		t.Fatalf("UpdateOperation failed: %v", err)
	}

	if updatedLRO == nil {
		t.Fatal("UpdateOperation returned nil LRO on success")
	}

	// Check fields that should be updated or set by the method/DB
	if updatedLRO.OperationID != opID {
		t.Errorf("UpdateOperation OperationID = %s, want %s", updatedLRO.OperationID, opID)
	}
	if updatedLRO.Status != model.LROStatusApproved {
		t.Errorf("UpdateOperation Status = %s, want %s", updatedLRO.Status, model.LROStatusApproved)
	}
	if diff := cmp.Diff(json.RawMessage(resultJSON), updatedLRO.ResultJSON); diff != "" {
		t.Errorf("UpdateOperation ResultJSON mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(json.RawMessage(errorDataJSONInput), updatedLRO.ErrorDataJSON); diff != "" {
		t.Errorf("UpdateOperation ErrorDataJSON mismatch (-want +got):\n%s", diff)
	}
	if !updatedLRO.UpdatedAt.Equal(expectedUpdatedAt) { // Check against expectedUpdatedAt
		t.Errorf("UpdateOperation UpdatedAt = %v, want %v", updatedLRO.UpdatedAt, expectedUpdatedAt)
	}
	if updatedLRO.RetryCount != 1 {
		t.Errorf("UpdateOperation RetryCount = %d, want %d", updatedLRO.RetryCount, 1)
	}

	// Check fields that are returned by the query
	if !updatedLRO.CreatedAt.Equal(expectedCreatedAt) {
		t.Errorf("UpdateOperation CreatedAt = %v, want %v", updatedLRO.CreatedAt, expectedCreatedAt)
	}
	if updatedLRO.Type != expectedType {
		t.Errorf("UpdateOperation Type = %s, want %s", updatedLRO.Type, expectedType)
	}
	if diff := cmp.Diff(json.RawMessage(originalRequestJSON), updatedLRO.RequestJSON); diff != "" {
		t.Errorf("UpdateOperation RequestJSON mismatch (-want +got):\n%s", diff)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestRegistry_UpdateOperation_Failure(t *testing.T) {
	ctx := context.Background()
	opID := "test-update-op-id-fail"
	dbErr := errors.New("database update error")
	validRequestJSON, _ := json.Marshal(map[string]string{"req": "original_data"})

	tests := []struct {
		name      string
		lro       *model.LRO
		mockSetup func(mock sqlmock.Sqlmock, lro *model.LRO)
		wantErr   error
	}{
		{
			name:    "nil LRO",
			lro:     nil,
			wantErr: errors.New("lro cannot be nil"),
		},
		{
			name:    "empty OperationID",
			lro:     &model.LRO{Status: model.LROStatusApproved},
			wantErr: errors.New("lro OperationID cannot be empty for update"),
		},
		{
			name: "operation not found (sql.ErrNoRows)",
			lro: &model.LRO{
				OperationID: opID,
				Status:      model.LROStatusApproved,
				ResultJSON:  validRequestJSON,
			},
			mockSetup: func(mock sqlmock.Sqlmock, lro *model.LRO) {
				mock.ExpectQuery(regexp.QuoteMeta(updateOperationQuery)).
					WithArgs(lro.OperationID, lro.Status, sqlmock.AnyArg(), sqlmock.AnyArg(), lro.RetryCount).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: ErrOperationNotFound,
		},
		{
			name: "other database error",
			lro: &model.LRO{
				OperationID: opID,
				Status:      model.LROStatusApproved,
				ResultJSON:  validRequestJSON,
			},
			mockSetup: func(mock sqlmock.Sqlmock, lro *model.LRO) {
				mock.ExpectQuery(regexp.QuoteMeta(updateOperationQuery)).
					WithArgs(lro.OperationID, lro.Status, sqlmock.AnyArg(), sqlmock.AnyArg(), lro.RetryCount).
					WillReturnError(dbErr)
			},
			wantErr: fmt.Errorf("failed to update operation %s: %w", opID, dbErr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, mock, db := newMockRegistry(t)
			defer db.Close()

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.lro)
			}

			_, err := r.UpdateOperation(ctx, tt.lro)

			if err == nil {
				t.Fatalf("UpdateOperation() expected an error, got nil")
			}
			if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
				t.Errorf("UpdateOperation() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRegistry_UpsertSubscriptionAndLRO_Success(t *testing.T) {
	ctx := context.Background()
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	sub := &model.Subscription{
		Subscriber: model.Subscriber{
			SubscriberID: "upsert-sub",
			URL:          "http://upsert.com",
			Type:         model.RoleBAP,
			Domain:       "upsert.domain",
		},
		KeyID:            "upsert-key",
		SigningPublicKey: "upsert-sign",
		EncrPublicKey:    "upsert-encr",
		ValidFrom:        fixedTime,
		ValidUntil:       fixedTime.Add(time.Hour),
		Status:           "SUBSCRIBED",
		Nonce:            "upsert-nonce",
	}

	lroRequestJSON, _ := json.Marshal(map[string]string{"req": "upsert_data"})
	lroResultJSON, _ := json.Marshal(map[string]string{"res": "upsert_result"})
	lroErrorDataJSON, _ := json.Marshal(map[string]string{"err": "upsert_error"})

	lro := &model.LRO{
		OperationID:   "upsert-op",
		Status:        model.LROStatusApproved,
		Type:          model.OperationTypeCreateSubscription,
		RequestJSON:   lroRequestJSON,
		ResultJSON:    lroResultJSON,
		ErrorDataJSON: lroErrorDataJSON,
		RetryCount:    0,
	}

	r, mock, db := newMockRegistry(t)
	defer db.Close()

	// Expect transaction begin
	mock.ExpectBegin()

	// Expect upsertSubscription query
	var locationJSON sql.NullString
	if sub.Location != nil {
		locBytes, _ := json.Marshal(sub.Location)
		locationJSON = sql.NullString{String: string(locBytes), Valid: true}
	} else {
		locationJSON = sql.NullString{}
	}
	mock.ExpectQuery(regexp.QuoteMeta(upsertSubscriptionQuery)).
		WithArgs(
			sub.SubscriberID, sub.URL, sub.Type, sub.Domain, locationJSON, sub.KeyID,
			sub.SigningPublicKey, sub.EncrPublicKey, sub.ValidFrom, sub.ValidUntil,
			sub.Status,
		).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(fixedTime, fixedTime))

	// Expect updateLRO query
	mock.ExpectQuery(regexp.QuoteMeta(updateOperationQuery)).
		WithArgs(lro.OperationID, lro.Status, sql.NullString{String: string(lroResultJSON), Valid: true}, sql.NullString{String: string(lroErrorDataJSON), Valid: true}, lro.RetryCount).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at", "type", "request_json"}).AddRow(fixedTime, fixedTime, lro.Type, lro.RequestJSON))

	// Expect transaction commit
	mock.ExpectCommit()

	updatedSub, updatedLRO, err := r.UpsertSubscriptionAndLRO(ctx, sub, lro)
	if err != nil {
		t.Fatalf("UpsertSubscriptionAndLRO() returned unexpected error: %v", err)
	}

	// Verify returned subscription
	expectedSub := *sub
	expectedSub.Created = fixedTime
	expectedSub.Updated = fixedTime
	if diff := cmp.Diff(&expectedSub, updatedSub); diff != "" {
		t.Errorf("UpsertSubscriptionAndLRO() returned subscription mismatch (-want +got):\n%s", diff)
	}

	// Verify returned LRO
	expectedLRO := *lro
	expectedLRO.CreatedAt = fixedTime
	expectedLRO.UpdatedAt = fixedTime
	if diff := cmp.Diff(&expectedLRO, updatedLRO); diff != "" {
		t.Errorf("UpsertSubscriptionAndLRO() returned LRO mismatch (-want +got):\n%s", diff)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestRegistry_UpsertSubscriptionAndLRO_Failure(t *testing.T) {
	ctx := context.Background()
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	validSub := &model.Subscription{
		Subscriber: model.Subscriber{
			SubscriberID: "upsert-fail-sub",
			URL:          "http://upsert-fail.com",
			Type:         model.RoleBAP,
			Domain:       "upsert-fail.domain",
		},
		KeyID:            "upsert-fail-key",
		SigningPublicKey: "upsert-fail-sign",
		EncrPublicKey:    "upsert-fail-encr",
		ValidFrom:        fixedTime,
		ValidUntil:       fixedTime.Add(time.Hour),
		Status:           "SUBSCRIBED",
		Nonce:            "upsert-fail-nonce",
	}

	validLRORequestJSON, _ := json.Marshal(map[string]string{"req": "upsert_fail_data"})
	validLRO := &model.LRO{
		OperationID: "upsert-fail-op",
		Status:      model.LROStatusPending,
		Type:        model.OperationTypeCreateSubscription,
		RequestJSON: validLRORequestJSON,
	}

	tests := []struct {
		name      string
		sub       *model.Subscription
		lro       *model.LRO
		mockSetup func(mock sqlmock.Sqlmock, sub *model.Subscription, lro *model.LRO)
		wantErr   error
	}{
		{
			name:    "validation failure - nil subscription",
			sub:     nil,
			lro:     validLRO,
			wantErr: errors.New("subscription cannot be nil"),
		},
		{
			name:    "validation failure - nil LRO",
			sub:     validSub,
			lro:     nil,
			wantErr: errors.New("lro cannot be nil"),
		},
		{
			name:    "validation failure - LRO missing OperationID",
			sub:     validSub,
			lro:     &model.LRO{Type: "create", RequestJSON: validLRORequestJSON},
			wantErr: fmt.Errorf("LRO validation failed: %w", ErrLROOperationIDMissing),
		},
		{
			name: "validation failure - subscription missing SubscriberID",
			sub: &model.Subscription{
				Subscriber: model.Subscriber{
					URL:    "http://fail.com",
					Type:   model.RoleBAP,
					Domain: "fail.domain",
				},
				KeyID: "key-fail",
			},
			lro:     validLRO,
			wantErr: errors.New("subscriberID and keyID are required for subscription"),
		},
		{
			name: "begin transaction error",
			sub:  validSub,
			lro:  validLRO,
			mockSetup: func(mock sqlmock.Sqlmock, sub *model.Subscription, lro *model.LRO) {
				mock.ExpectBegin().WillReturnError(errors.New("tx begin error"))
			},
			wantErr: errors.New("failed to begin transaction"),
		},
		{
			name: "upsert subscription error",
			sub:  validSub,
			lro:  validLRO,
			mockSetup: func(mock sqlmock.Sqlmock, sub *model.Subscription, lro *model.LRO) {
				mock.ExpectBegin()
				var locationJSON sql.NullString
				if sub.Location != nil {
					locBytes, _ := json.Marshal(sub.Location)
					locationJSON = sql.NullString{String: string(locBytes), Valid: true}
				} else {
					locationJSON = sql.NullString{}
				}
				mock.ExpectQuery(regexp.QuoteMeta(upsertSubscriptionQuery)).
					WithArgs(
						sub.SubscriberID, sub.URL, sub.Type, sub.Domain, locationJSON, sub.KeyID,
						sub.SigningPublicKey, sub.EncrPublicKey, sub.ValidFrom, sub.ValidUntil,
						sub.Status,
					).
					WillReturnError(errors.New("upsert sub error"))
				mock.ExpectRollback() // Expect rollback on error
			},
			wantErr: errors.New("failed to upsert subscription"),
		},
		{
			name: "update LRO error",
			sub:  validSub,
			lro:  validLRO,
			mockSetup: func(mock sqlmock.Sqlmock, sub *model.Subscription, lro *model.LRO) {
				mock.ExpectBegin()
				var locationJSON sql.NullString
				if sub.Location != nil {
					locBytes, _ := json.Marshal(sub.Location)
					locationJSON = sql.NullString{String: string(locBytes), Valid: true}
				} else {
					locationJSON = sql.NullString{}
				}
				mock.ExpectQuery(regexp.QuoteMeta(upsertSubscriptionQuery)).
					WithArgs(
						sub.SubscriberID, sub.URL, sub.Type, sub.Domain, locationJSON, sub.KeyID,
						sub.SigningPublicKey, sub.EncrPublicKey, sub.ValidFrom, sub.ValidUntil,
						sub.Status,
					).
					WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(fixedTime, fixedTime))
				mock.ExpectQuery(regexp.QuoteMeta(updateOperationQuery)).
					WithArgs(lro.OperationID, lro.Status, sqlmock.AnyArg(), sqlmock.AnyArg(), lro.RetryCount).
					WillReturnError(errors.New("update LRO error"))
				mock.ExpectRollback() // Expect rollback on error
			},
			wantErr: errors.New("failed to update LRO"),
		},
		{
			name: "commit transaction error",
			sub:  validSub,
			lro:  validLRO,
			mockSetup: func(mock sqlmock.Sqlmock, sub *model.Subscription, lro *model.LRO) {
				mock.ExpectBegin()
				var locationJSON sql.NullString
				if sub.Location != nil {
					locBytes, _ := json.Marshal(sub.Location)
					locationJSON = sql.NullString{String: string(locBytes), Valid: true}
				} else {
					locationJSON = sql.NullString{}
				}
				mock.ExpectQuery(regexp.QuoteMeta(upsertSubscriptionQuery)).
					WithArgs(
						sub.SubscriberID, sub.URL, sub.Type, sub.Domain, locationJSON, sub.KeyID,
						sub.SigningPublicKey, sub.EncrPublicKey, sub.ValidFrom, sub.ValidUntil,
						sub.Status,
					).
					WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(fixedTime, fixedTime))
				mock.ExpectQuery(regexp.QuoteMeta(updateOperationQuery)).
					WithArgs(lro.OperationID, lro.Status, sqlmock.AnyArg(), sqlmock.AnyArg(), lro.RetryCount).
					WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at", "type", "request_json"}).AddRow(fixedTime, fixedTime, lro.Type, lro.RequestJSON))
				mock.ExpectCommit().WillReturnError(errors.New("commit error"))
			},
			wantErr: errors.New("failed to commit transaction"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, mock, db := newMockRegistry(t)
			defer db.Close()

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.sub, tt.lro)
			}

			_, _, err := r.UpsertSubscriptionAndLRO(ctx, tt.sub, tt.lro)

			if err == nil {
				t.Fatalf("UpsertSubscriptionAndLRO() expected an error, got nil")
			}
			if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
				t.Errorf("UpsertSubscriptionAndLRO() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRegistry_EncryptionKey_Success(t *testing.T) {
	ctx := context.Background()
	subscriberID := "sub-enc-test"
	keyID := "key-enc-test"
	expectedPublicKey := "test-encryption-public-key"

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer mockDB.Close()

	r, err := NewRegistry(mockDB)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	rows := sqlmock.NewRows([]string{"encr_public_key"}).AddRow(expectedPublicKey)
	mock.ExpectQuery(regexp.QuoteMeta(getSubscriberEncryptionKeyQuery)).
		WithArgs(subscriberID, keyID).
		WillReturnRows(rows)

	retrievedKey, err := r.EncryptionKey(ctx, subscriberID, keyID)
	if err != nil {
		t.Fatalf("EncryptionKey failed: %v", err)
	}
	if retrievedKey != expectedPublicKey {
		t.Errorf("Expected encryption key '%s', got '%s'", expectedPublicKey, retrievedKey)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestRegistry_EncryptionKey_Failure(t *testing.T) {
	ctx := context.Background()
	subscriberID := "sub-enc-fail"
	keyID := "key-enc-fail"
	otherDBError := errors.New("some other encryption key query error")

	tests := []struct {
		name            string
		mockSetup       func(mock sqlmock.Sqlmock)
		wantErr         error
		checkErrMessage func(t *testing.T, err error)
	}{
		{
			name: "key not found (sql.ErrNoRows)",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(getSubscriberEncryptionKeyQuery)).
					WithArgs(subscriberID, keyID).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: ErrEncrKeyNotFound, // Changed from ErrSubscriberKeyNotFound to ErrEncrKeyNotFound
			checkErrMessage: func(t *testing.T, err error) {
				expectedMsg := fmt.Sprintf("%s: for subscriber_id '%s', key_id '%s'", ErrEncrKeyNotFound, subscriberID, keyID)
				if err.Error() != expectedMsg {
					t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
				}
			},
		},
		{
			name: "other database error during query",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(getSubscriberEncryptionKeyQuery)).
					WithArgs(subscriberID, keyID).
					WillReturnError(otherDBError)
			},
			wantErr: otherDBError,
			checkErrMessage: func(t *testing.T, err error) {
				expectedMsgPrefix := "failed to query subscriber encryption key"
				if actualErrStr := err.Error(); len(actualErrStr) < len(expectedMsgPrefix) || actualErrStr[:len(expectedMsgPrefix)] != expectedMsgPrefix {
					t.Errorf("Expected error message to start with '%s', got '%s'", expectedMsgPrefix, actualErrStr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB, mock, _ := sqlmock.New()
			defer mockDB.Close()
			r, _ := NewRegistry(mockDB)
			tt.mockSetup(mock)
			_, err := r.EncryptionKey(ctx, subscriberID, keyID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("EncryptionKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.checkErrMessage != nil {
				tt.checkErrMessage(t, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestBuildLookupConditions(t *testing.T) {
	tests := []struct {
		name     string
		filter   *model.Subscription
		expected []goqu.Expression
	}{
		{
			name:     "Empty filter",
			filter:   &model.Subscription{},
			expected: []goqu.Expression{},
		},
		{
			name: "SubscriberID filter",
			filter: &model.Subscription{
				Subscriber: model.Subscriber{SubscriberID: "test_id"},
			},
			expected: []goqu.Expression{
				goqu.C("subscriber_id").Eq("test_id"),
			},
		},
		{
			name: "URL and Type filter",
			filter: &model.Subscription{
				Subscriber: model.Subscriber{URL: "http://example.com", Type: model.RoleBAP},
			},
			expected: []goqu.Expression{
				goqu.C("url").Eq("http://example.com"),
				goqu.C("type").Eq("BAP"),
			},
		},
		{
			name: "All top-level filters (no location)",
			filter: &model.Subscription{
				Subscriber: model.Subscriber{
					SubscriberID: "sub1",
					URL:          "http://all.com",
					Type:         model.RoleBPP,
					Domain:       "domain.all",
				},
				KeyID:  "key_all",
				Status: "SUBSCRIBED",
			},
			expected: []goqu.Expression{
				goqu.C("subscriber_id").Eq("sub1"),
				goqu.C("url").Eq("http://all.com"),
				goqu.C("type").Eq("BPP"),
				goqu.C("domain").Eq("domain.all"),
				goqu.C("status").Eq("SUBSCRIBED"),
				goqu.C("key_id").Eq("key_all"),
			},
		},
		{
			name: "Filter with basic Location fields",
			filter: &model.Subscription{
				Subscriber: model.Subscriber{
					Location: &model.Location{
						ID:       "loc_id",
						MapURL:   "http://map.test",
						Address:  "123 Main St",
						AreaCode: "100001",
					},
				},
			},
			expected: []goqu.Expression{
				goqu.L("location->>'id'").Eq("loc_id"),
				goqu.L("location->>'map_url'").Eq("http://map.test"),
				goqu.L("location->>'address'").Eq("123 Main St"),
				goqu.L("location->>'area_code'").Eq("100001"),
			},
		},
		{
			name: "Filter with nested Location fields (City and State)",
			filter: &model.Subscription{
				Subscriber: model.Subscriber{
					Location: &model.Location{
						City:  &model.City{Name: "Mumbai", Code: "MH"},
						State: &model.State{Name: "Maharashtra"},
					},
				},
			},
			expected: []goqu.Expression{
				goqu.L("location->'city'->>'name'").Eq("Mumbai"),
				goqu.L("location->'city'->>'code'").Eq("MH"),
				goqu.L("location->'state'->>'name'").Eq("Maharashtra"),
			},
		},
		{
			name: "Filter with all possible location fields",
			filter: &model.Subscription{
				Subscriber: model.Subscriber{
					Location: &model.Location{
						ID:          "L1",
						MapURL:      "http://map.com",
						Address:     "addr1",
						District:    "dist1",
						AreaCode:    "code1",
						Polygon:     "poly1",
						ThreeDSpace: "3d1",
						Rating:      "5",
						City:        &model.City{Name: "C1", Code: "CC1"},
						State:       &model.State{Name: "S1", Code: "SC1"},
						Country:     &model.Country{Name: "CO1", Code: "COC1"},
					},
				},
			},
			expected: []goqu.Expression{
				goqu.L("location->>'id'").Eq("L1"),
				goqu.L("location->>'map_url'").Eq("http://map.com"),
				goqu.L("location->>'address'").Eq("addr1"),
				goqu.L("location->>'district'").Eq("dist1"),
				goqu.L("location->>'area_code'").Eq("code1"),
				goqu.L("location->>'polygon'").Eq("poly1"),
				goqu.L("location->>'3dspace'").Eq("3d1"),
				goqu.L("location->>'rating'").Eq("5"),
				goqu.L("location->'city'->>'name'").Eq("C1"),
				goqu.L("location->'city'->>'code'").Eq("CC1"),
				goqu.L("location->'state'->>'name'").Eq("S1"),
				goqu.L("location->'state'->>'code'").Eq("SC1"),
				goqu.L("location->'country'->>'name'").Eq("CO1"),
				goqu.L("location->'country'->>'code'").Eq("COC1"),
			},
		},
		{
			name: "Combined top-level and location filters",
			filter: &model.Subscription{
				Subscriber: model.Subscriber{
					SubscriberID: "hybrid_sub",
					Type:         "BG",
					Location: &model.Location{
						Address: "Hybrid Address",
						City:    &model.City{Name: "Hybrid City"},
					},
				},
				Status: "UNDER_SUBSCRIPTION",
			},
			expected: []goqu.Expression{
				goqu.C("subscriber_id").Eq("hybrid_sub"),
				goqu.C("type").Eq("BG"),
				goqu.C("status").Eq("UNDER_SUBSCRIPTION"),
				goqu.L("location->>'address'").Eq("Hybrid Address"),
				goqu.L("location->'city'->>'name'").Eq("Hybrid City"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualConditions := buildLookupConditions(tt.filter)

			actualExprStrings := make(map[string]struct{})
			for _, expr := range actualConditions {
				sql, _, _ := goqu.From("temp").Where(expr).ToSQL()
				actualExprStrings[extractWhereClause(sql)] = struct{}{}
			}

			expectedExprStrings := make(map[string]struct{})
			for _, expr := range tt.expected {
				sql, _, _ := goqu.From("temp").Where(expr).ToSQL()
				expectedExprStrings[extractWhereClause(sql)] = struct{}{}
			}

			if len(actualExprStrings) != len(expectedExprStrings) {
				t.Errorf("buildLookupConditions() mismatch in number of conditions:\nActual: %v\nExpected: %v", actualExprStrings, expectedExprStrings)
			} else {
				for s := range expectedExprStrings {
					if _, ok := actualExprStrings[s]; !ok {
						t.Errorf("buildLookupConditions() missing expected condition: '%s'\nActual conditions: %v", s, actualExprStrings)
						break
					}
				}
			}
		})
	}
}

func TestBuildLocationConditions(t *testing.T) {
	tests := []struct {
		name     string
		filter   *model.Location
		expected []goqu.Expression
	}{
		{
			name:     "Nil location filter",
			filter:   nil,
			expected: []goqu.Expression{},
		},
		{
			name:     "Empty location filter",
			filter:   &model.Location{},
			expected: []goqu.Expression{},
		},
		{
			name: "Direct fields only",
			filter: &model.Location{
				ID:          "some_id",
				MapURL:      "http://map.test",
				Address:     "Test Address",
				District:    "Test District",
				AreaCode:    "123456",
				Polygon:     "polygon_data",
				ThreeDSpace: "3d_space_data",
				Rating:      "4.5",
			},
			expected: []goqu.Expression{
				goqu.L("location->>'id'").Eq("some_id"),
				goqu.L("location->>'map_url'").Eq("http://map.test"),
				goqu.L("location->>'address'").Eq("Test Address"),
				goqu.L("location->>'district'").Eq("Test District"),
				goqu.L("location->>'area_code'").Eq("123456"),
				goqu.L("location->>'polygon'").Eq("polygon_data"),
				goqu.L("location->>'3dspace'").Eq("3d_space_data"),
				goqu.L("location->>'rating'").Eq("4.5"),
			},
		},
		{
			name: "City filter only",
			filter: &model.Location{
				City: &model.City{Name: "Test City", Code: "TC"},
			},
			expected: []goqu.Expression{
				goqu.L("location->'city'->>'name'").Eq("Test City"),
				goqu.L("location->'city'->>'code'").Eq("TC"),
			},
		},
		{
			name: "State filter only",
			filter: &model.Location{
				State: &model.State{Name: "Test State", Code: "TS"},
			},
			expected: []goqu.Expression{
				goqu.L("location->'state'->>'name'").Eq("Test State"),
				goqu.L("location->'state'->>'code'").Eq("TS"),
			},
		},
		{
			name: "Country filter only",
			filter: &model.Location{
				Country: &model.Country{Name: "Test Country", Code: "TCtry"},
			},
			expected: []goqu.Expression{
				goqu.L("location->'country'->>'name'").Eq("Test Country"),
				goqu.L("location->'country'->>'code'").Eq("TCtry"),
			},
		},
		{
			name: "Combined direct and nested fields",
			filter: &model.Location{
				Address: "Combined Address",
				City:    &model.City{Name: "Combined City"},
				Country: &model.Country{Code: "IN"},
			},
			expected: []goqu.Expression{
				goqu.L("location->>'address'").Eq("Combined Address"),
				goqu.L("location->'city'->>'name'").Eq("Combined City"),
				goqu.L("location->'country'->>'code'").Eq("IN"),
			},
		},
		{
			name: "Nested fields with nil pointers for some fields",
			filter: &model.Location{
				City:    &model.City{Name: "Partial City"},
				State:   nil,
				Country: &model.Country{Code: "US"},
			},
			expected: []goqu.Expression{
				goqu.L("location->'city'->>'name'").Eq("Partial City"),
				goqu.L("location->'country'->>'code'").Eq("US"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualConditions := buildLocationConditions(tt.filter)

			actualExprStrings := make(map[string]struct{})
			for _, expr := range actualConditions {
				sql, _, _ := goqu.From("temp").Where(expr).ToSQL()
				actualExprStrings[extractWhereClause(sql)] = struct{}{}
			}

			expectedExprStrings := make(map[string]struct{})
			for _, expr := range tt.expected {
				sql, _, _ := goqu.From("temp").Where(expr).ToSQL()
				expectedExprStrings[extractWhereClause(sql)] = struct{}{}
			}

			if len(actualExprStrings) != len(expectedExprStrings) {
				t.Errorf("buildLocationConditions() mismatch in number of conditions:\nActual: %v\nExpected: %v", actualExprStrings, expectedExprStrings)
			} else {
				for s := range expectedExprStrings {
					if _, ok := actualExprStrings[s]; !ok {
						t.Errorf("buildLocationConditions() missing expected condition: '%s'\nActual conditions: %v", s, actualExprStrings)
						break
					}
				}
			}
		})
	}
}

func extractWhereClause(sql string) string {
	idx := strings.Index(sql, " WHERE ")
	if idx == -1 {
		return ""
	}
	whereClause := sql[idx+len(" WHERE "):]
	if strings.HasPrefix(whereClause, "(") && strings.HasSuffix(whereClause, ")") {
		return whereClause[1 : len(whereClause)-1]
	}
	return whereClause
}

func TestRegistry_GetSubscriberSigningKey_Success(t *testing.T) {
	ctx := context.Background()
	subscriberID := "sub1"
	domain := "example.com"
	role := model.Role("TEST_ROLE")
	keyID := "key1"
	publicKey := "test-public-key"

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer mockDB.Close()

	r, err := NewRegistry(mockDB)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	rows := sqlmock.NewRows([]string{"signing_public_key"}).AddRow(publicKey)
	mock.ExpectQuery(regexp.QuoteMeta(getSubscriberSigningKeyQuery)).
		WithArgs(subscriberID, domain, role, keyID).
		WillReturnRows(rows)

	retrievedKey, err := r.GetSubscriberSigningKey(ctx, subscriberID, domain, role, keyID)
	if err != nil {
		t.Fatalf("GetSubscriberSigningKey failed: %v", err)
	}
	if retrievedKey != publicKey {
		t.Errorf("Expected key '%s', got '%s'", publicKey, retrievedKey)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestRegistry_GetSubscriberSigningKey_Failure(t *testing.T) {
	ctx := context.Background()
	subscriberID := "sub1"
	domain := "example.com"
	role := model.Role("TEST_ROLE")
	keyID := "key1"
	otherDBError := errors.New("some other query error")

	tests := []struct {
		name            string
		mockSetup       func(mock sqlmock.Sqlmock)
		wantErr         error
		wantKey         string
		checkErrMessage func(t *testing.T, err error)
	}{
		{
			name: "key not found (sql.ErrNoRows)",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(getSubscriberSigningKeyQuery)).
					WithArgs(subscriberID, domain, role, keyID).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: ErrSubscriberKeyNotFound,
			checkErrMessage: func(t *testing.T, err error) {
				expectedMsg := fmt.Sprintf("%s: for subscriber_id '%s', domain '%s', type '%s', key_id '%s'", ErrSubscriberKeyNotFound, subscriberID, domain, role, keyID)
				if err.Error() != expectedMsg {
					t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
				}
			},
		},
		{
			name: "other database error during query",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(getSubscriberSigningKeyQuery)).
					WithArgs(subscriberID, domain, role, keyID).
					WillReturnError(otherDBError)
			},
			wantErr: otherDBError,
			checkErrMessage: func(t *testing.T, err error) {
				expectedMsgPrefix := "failed to query subscriber signing key"
				if actualErrStr := err.Error(); len(actualErrStr) < len(expectedMsgPrefix) || actualErrStr[:len(expectedMsgPrefix)] != expectedMsgPrefix {
					t.Errorf("Expected error message to start with '%s', got '%s'", expectedMsgPrefix, actualErrStr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create sqlmock for subtest '%s': %v", tt.name, err)
			}
			defer mockDB.Close()

			r, err := NewRegistry(mockDB)
			if err != nil {
				t.Fatalf("NewRegistry failed for subtest '%s': %v", tt.name, err)
			}

			tt.mockSetup(mock)

			retrievedKey, err := r.GetSubscriberSigningKey(ctx, subscriberID, domain, role, keyID)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("GetSubscriberSigningKey() error = %v, wantErr %v", err, tt.wantErr)
			}

			// For failure cases, retrievedKey should be empty (or its zero value)
			if tt.wantErr != nil {
				if retrievedKey != "" {
					t.Errorf("GetSubscriberSigningKey() gotKey = %v, want empty string as an error (%v) was expected", retrievedKey, tt.wantErr)
				}
			}

			if tt.checkErrMessage != nil {
				tt.checkErrMessage(t, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRegistry_GetOperation_Success(t *testing.T) {
	ctx := context.Background()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer mockDB.Close()

	r, err := NewRegistry(mockDB)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	opID := "test-op-id"
	now := time.Now()
	requestJSON, _ := json.Marshal(map[string]string{"req": "data"})
	resultJSON, _ := json.Marshal(map[string]string{"res": "data"})
	errorDataJSON, _ := json.Marshal(map[string]string{"err": "detail"})

	expectedLRO := &model.LRO{
		OperationID:   opID,
		Status:        model.LROStatusPending,
		Type:          model.OperationTypeCreateSubscription,
		RequestJSON:   requestJSON,
		ResultJSON:    resultJSON,
		ErrorDataJSON: errorDataJSON,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	rows := sqlmock.NewRows([]string{"operation_id", "status", "type", "request_json", "result_json", "error_data_json", "created_at", "updated_at"}).
		AddRow(expectedLRO.OperationID, expectedLRO.Status, expectedLRO.Type, expectedLRO.RequestJSON, expectedLRO.ResultJSON, expectedLRO.ErrorDataJSON, expectedLRO.CreatedAt, expectedLRO.UpdatedAt)

	mock.ExpectQuery(regexp.QuoteMeta(getOperationQuery)).
		WithArgs(opID).
		WillReturnRows(rows)

	retrievedLRO, err := r.GetOperation(ctx, opID)
	if err != nil {
		t.Fatalf("GetOperation failed: %v", err)
	}

	if diff := cmp.Diff(expectedLRO, retrievedLRO); diff != "" {
		t.Errorf("GetOperation() mismatch (-want +got):\n%s", diff)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}

	// Test case where ErrorDataJSON is NULL in DB
	t.Run("GetOperation_Success_NullErrorData", func(t *testing.T) {
		mockDBNullErr, mockNullErr, errNull := sqlmock.New()
		if errNull != nil {
			t.Fatalf("Failed to create sqlmock for null error test: %v", errNull)
		}
		defer mockDBNullErr.Close()

		rNullErr, _ := NewRegistry(mockDBNullErr)
		opIDNullErr := "test-op-id-null-error"
		expectedLRONullError := &model.LRO{
			OperationID:   opIDNullErr,
			Status:        model.LROStatusPending,
			Type:          model.OperationTypeCreateSubscription,
			RequestJSON:   requestJSON,
			ResultJSON:    resultJSON,
			ErrorDataJSON: nil, // Expect nil or empty []byte
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		rowsNullErr := sqlmock.NewRows([]string{"operation_id", "status", "type", "request_json", "result_json", "error_data_json", "created_at", "updated_at"}).
			AddRow(expectedLRONullError.OperationID, expectedLRONullError.Status, expectedLRONullError.Type, expectedLRONullError.RequestJSON, expectedLRONullError.ResultJSON, nil, expectedLRONullError.CreatedAt, expectedLRONullError.UpdatedAt)

		mockNullErr.ExpectQuery(regexp.QuoteMeta(getOperationQuery)).
			WithArgs(opIDNullErr).
			WillReturnRows(rowsNullErr)

		retrievedLRONullErr, errGetNull := rNullErr.GetOperation(ctx, opIDNullErr)
		if errGetNull != nil {
			t.Fatalf("GetOperation (null error) failed: %v", errGetNull)
		}
		if diff := cmp.Diff(expectedLRONullError, retrievedLRONullErr); diff != "" {
			t.Errorf("GetOperation (null error) mismatch (-want +got):\n%s", diff)
		}
		if err := mockNullErr.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations for null error test: %s", err)
		}
	})
}

func TestRegistry_GetOperation_Failure(t *testing.T) {
	ctx := context.Background()
	opID := "test-op-id-fail"
	otherDBError := errors.New("some other query error during get operation")

	tests := []struct {
		name            string
		opID            string
		mockSetup       func(mock sqlmock.Sqlmock, id string)
		wantErr         error
		checkErrMessage func(t *testing.T, err error, id string)
	}{
		{
			name: "operation not found (sql.ErrNoRows)",
			opID: opID,
			mockSetup: func(mock sqlmock.Sqlmock, id string) {
				mock.ExpectQuery(regexp.QuoteMeta(getOperationQuery)).
					WithArgs(id).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: ErrOperationNotFound,
		},
		{
			name: "other database error during query",
			opID: opID,
			mockSetup: func(mock sqlmock.Sqlmock, id string) {
				mock.ExpectQuery(regexp.QuoteMeta(getOperationQuery)).
					WithArgs(id).
					WillReturnError(otherDBError)
			},
			wantErr: otherDBError,
			checkErrMessage: func(t *testing.T, err error, id string) {
				expectedMsgPrefix := fmt.Sprintf("failed to get operation with ID %s", id)
				if actualErrStr := err.Error(); len(actualErrStr) < len(expectedMsgPrefix) || actualErrStr[:len(expectedMsgPrefix)] != expectedMsgPrefix {
					t.Errorf("Expected error message to start with '%s', got '%s'", expectedMsgPrefix, actualErrStr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB, mock, _ := sqlmock.New() // Error already checked in success test
			defer mockDB.Close()
			r, _ := NewRegistry(mockDB) // Error already checked in NewRegistry tests

			tt.mockSetup(mock, tt.opID)

			_, err := r.GetOperation(ctx, tt.opID)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("GetOperation() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.checkErrMessage != nil {
				tt.checkErrMessage(t, err, tt.opID)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %s", err)
			}
		})
	}
}
