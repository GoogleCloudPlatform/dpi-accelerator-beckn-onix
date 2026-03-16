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
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"cloud.google.com/go/cloudsqlconn/postgres/pgxv5"
	// Import postgres dialect for goqu.
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/doug-martin/goqu/v9"
	"github.com/lib/pq"
	"github.com/jmoiron/sqlx"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// Predefined errors for config validation.
var (
	ErrDBNil                  = errors.New("sql.DB is nil")
	ErrOperationAlreadyExists = errors.New("operation with this ID already exists")
	ErrEncrKeyNotFound        = errors.New("key not found")
	// LRO specific validation errors
	ErrLROIsNil              = errors.New("LRO object is nil")
	ErrLROOperationIDMissing = errors.New("LRO OperationID is missing")
	ErrLROTypeMissing        = errors.New("LRO type is missing")
	ErrLRORequestJSONMissing = errors.New("LRO RequestJSON is missing")
	ErrSubscriberKeyNotFound = errors.New("subscriber signing key not found")
	ErrSubscriptionConflict  = errors.New("subscription already exists or conflicts with an existing one")
	ErrOperationNotFound     = errors.New("operation not found")
)

// subscriptionsTableName defines the name of the database table for subscriptions.
const subscriptionsTableName = "subscriptions"

// registry implements the lookUpRepository interface using PostgreSQL.
type Config struct {
	User            string        `yaml:"user"`
	Name            string        `yaml:"name"`            // Database name.
	ConnectionName  string        `yaml:"connectionName"`  // Cloud SQL connection name.
	MaxOpenConns    int           `yaml:"maxOpenConns"`    // Maximum number of open connections to the database.
	MaxIdleConns    int           `yaml:"maxIdleConns"`    // Maximum number of connections in the idle connection pool.
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime"` // Maximum amount of time a connection may be idle.
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime"` // Maximum amount of time a connection may be reused.
}

type registry struct {
	db *sqlx.DB // Use sqlx.DB for enhanced functionality.
}

// NewRegistry creates a new PostgresSubscriberRepository.
func NewRegistry(db *sql.DB) (*registry, error) {
	if db == nil {
		return nil, ErrDBNil
	}
	// Convert standard *sql.DB to *sqlx.DB.
	return &registry{db: sqlx.NewDb(db, "postgres")}, nil
}

// Lookup retrieves subscriptions based on the provided filter criteria.
func (r *registry) Lookup(ctx context.Context, filter *model.Subscription) ([]model.Subscription, error) {
	slog.Info("Repository: Executing Lookup query", "filter", filter)

	// Create a new goqu dataset for the "subscriptions" table.
	// We'll select all columns, and sqlx will map them to the Subscription struct.
	dataset := goqu.From(subscriptionsTableName).Select(
		"subscriber_id", "url", "type", "domain", "location", "key_id",
		"signing_public_key", "encr_public_key", "valid_from", "valid_until",
		"status", "created_at", "updated_at",
	)

	// Build conditions using a helper function to centralize the logic.
	conditions := buildLookupConditions(filter)

	// Apply all conditions to the dataset.
	if len(conditions) > 0 {
		dataset = dataset.Where(conditions...)
	}

	// Generate SQL and arguments.
	sql, args, err := dataset.ToSQL()
	if err != nil {
		slog.Error("Repository: Failed to build SQL query", "error", err)
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}

	slog.Info("Repository: Generated SQL query", "sql", sql, "args", args)

	subscriptions := []model.Subscription{}
	// Use sqlx.SelectContext to execute the query and unmarshal results into []model.Subscription.
	err = r.db.SelectContext(ctx, &subscriptions, sql, args...)
	if err != nil {
		slog.Error("Repository: Failed to execute lookup query", "error", err)
		return nil, fmt.Errorf("failed to execute lookup query: %w", err)
	}

	slog.Info("Repository: Lookup query successful", "count", len(subscriptions))
	return subscriptions, nil
}

// buildLookupConditions creates a slice of goqu expressions based on the model.Subscription filter.
// This centralizes the logic for building the WHERE clause, making the main Lookup method cleaner.
func buildLookupConditions(filter *model.Subscription) []goqu.Expression {
	var conditions []goqu.Expression

	// Top-level fields.
	if filter.SubscriberID != "" {
		conditions = append(conditions, goqu.C("subscriber_id").Eq(filter.SubscriberID))
	}
	if filter.URL != "" {
		conditions = append(conditions, goqu.C("url").Eq(filter.URL))
	}
	if filter.Type != "" {
		conditions = append(conditions, goqu.C("type").Eq(filter.Type))
	}
	if filter.Domain != "" {
		conditions = append(conditions, goqu.C("domain").Eq(filter.Domain))
	}
	if filter.Status != "" {
		conditions = append(conditions, goqu.C("status").Eq(filter.Status))
	}
	if filter.KeyID != "" {
		conditions = append(conditions, goqu.C("key_id").Eq(filter.KeyID))
	}

	// Append location-specific conditions if a location filter is provided.
	// This delegates the complex location filtering logic to a dedicated helper.
	locationConditions := buildLocationConditions(filter.Location)
	conditions = append(conditions, locationConditions...)

	return conditions
}

// buildLocationConditions creates a slice of goqu expressions for location-related filters.
// This helper method encapsulates the logic for building conditions on the 'location' JSONB column.
// It uses an early return pattern to reduce nesting for the primary nil check.
func buildLocationConditions(locationFilter *model.Location) []goqu.Expression {
	// If no location filter is provided, return early with no conditions.
	if locationFilter == nil {
		return nil // Returning nil is equivalent to an empty slice for goqu.
	}

	var conditions []goqu.Expression

	// Using a slice of anonymous structs to iterate through direct location fields for cleaner code.
	directLocationFilters := []struct {
		Value string
		Path  string
	}{
		{locationFilter.ID, "location->>'id'"},
		{locationFilter.MapURL, "location->>'map_url'"},
		{locationFilter.Address, "location->>'address'"},
		{locationFilter.District, "location->>'district'"},
		{locationFilter.AreaCode, "location->>'area_code'"},
		{locationFilter.Polygon, "location->>'polygon'"},
		{locationFilter.ThreeDSpace, "location->>'3dspace'"},
		{locationFilter.Rating, "location->>'rating'"},
	}

	// Iterate over the defined location filters and add conditions if the value is present.
	for _, lf := range directLocationFilters {
		if lf.Value != "" {
			conditions = append(conditions, goqu.L(lf.Path).Eq(lf.Value))
		}
	}

	// Nested fields: Check if City, State, Country pointers are not nil before accessing their fields.
	// This prevents nil pointer dereferences.
	if locationFilter.City != nil {
		if locationFilter.City.Name != "" {
			conditions = append(conditions, goqu.L("location->'city'->>'name'").Eq(locationFilter.City.Name))
		}
		if locationFilter.City.Code != "" {
			conditions = append(conditions, goqu.L("location->'city'->>'code'").Eq(locationFilter.City.Code))
		}
	}

	if locationFilter.State != nil {
		if locationFilter.State.Name != "" {
			conditions = append(conditions, goqu.L("location->'state'->>'name'").Eq(locationFilter.State.Name))
		}
		if locationFilter.State.Code != "" {
			conditions = append(conditions, goqu.L("location->'state'->>'code'").Eq(locationFilter.State.Code))
		}
	}

	if locationFilter.Country != nil {
		if locationFilter.Country.Name != "" {
			conditions = append(conditions, goqu.L("location->'country'->>'name'").Eq(locationFilter.Country.Name))
		}
		if locationFilter.Country.Code != "" {
			conditions = append(conditions, goqu.L("location->'country'->>'code'").Eq(locationFilter.Country.Code))
		}
	}

	return conditions
}

var pgxv5Registerer = pgxv5.RegisterDriver
var sqlOpen = sql.Open

// NewConnectionPool creates a new database connection pool.
func NewConnectionPool(ctx context.Context, cfg *Config) (*sql.DB, func() error, error) {
	if cfg.ConnectionName == "" {
		return nil, nil, fmt.Errorf("db.connectionName is required in config")
	}
	if cfg.User == "" {
		return nil, nil, fmt.Errorf("db.user is required in config")
	}
	if cfg.Name == "" {
		return nil, nil, fmt.Errorf("db.name is required in config")
	}

	cleanup, err := pgxv5Registerer("cloudsql-iam-postgres", cloudsqlconn.WithIAMAuthN())
	if err != nil {
		return nil, nil, fmt.Errorf("pgxv5.RegisterDriver: %w", err)
	}

	dsn := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable",
		cfg.ConnectionName,
		cfg.User,
		cfg.Name,
	)

	db, err := sqlOpen("cloudsql-iam-postgres", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Configure the Connection Pool.
	// A value of 0 or less for any of these settings means default behavior.
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	// Default is 2.
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	// Default is 0 (to not close idle connections).
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}
	// Default is 0 (that connections are not closed due to their age).
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	// Ping the database to ensure a connection can be made.
	if err = db.Ping(); err != nil {
		if cleanupErr := cleanup(); cleanupErr != nil {
			slog.ErrorContext(ctx, "failed to run pgxv5 cleanup after ping failure", "error", cleanupErr)
		}
		db.Close()
		return nil, nil, fmt.Errorf("db.Ping failed: %w", err)
	}

	fullCleanup := func() error {
		db.Close()
		return cleanup()
	}

	return db, fullCleanup, nil
}

const insertOperationQuery = `
	INSERT INTO Operations (operation_id, status, type, request_json, result_json, error_data_json)
	VALUES ($1, $2, $3, $4, NULL, NULL)
	RETURNING created_at, updated_at`

const updateOperationQuery = `
	UPDATE Operations
	SET status = $2, result_json = $3, error_data_json = $4, retry_count = $5
	WHERE operation_id = $1
	RETURNING created_at, updated_at, type, request_json;`

// upsertSubscriptionQuery lets the DB handle created_at (on insert) and updated_at (on update via trigger).
const upsertSubscriptionQuery = `
	INSERT INTO subscriptions (subscriber_id, url, type, domain, location, key_id, signing_public_key, encr_public_key, valid_from, valid_until, status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	ON CONFLICT (subscriber_id, domain, type) DO UPDATE SET
		url = EXCLUDED.url,
		location = EXCLUDED.location,
		signing_public_key = EXCLUDED.signing_public_key,
		encr_public_key = EXCLUDED.encr_public_key,
		valid_from = EXCLUDED.valid_from,
		valid_until = EXCLUDED.valid_until,
		status = EXCLUDED.status
	RETURNING created_at, updated_at;` // Return DB-generated timestamps

const insertOnlySubscriptionQuery = `
	INSERT INTO subscriptions (
		subscriber_id, url, type, domain, location,
		key_id, signing_public_key, encr_public_key,
		valid_from, valid_until, status, nonce
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	RETURNING created_at, updated_at;`

// validateLRO checks if the LRO object has the minimum required fields for a new operation insertion.
func validateLRO(lro *model.LRO) error {
	if lro == nil {
		return ErrLROIsNil
	}
	if lro.OperationID == "" {
		return ErrLROOperationIDMissing
	}
	if lro.Type == "" {
		return ErrLROTypeMissing
	}
	if lro.RequestJSON == nil {
		return ErrLRORequestJSONMissing
	}
	return nil
}

// InsertOperation inserts a new operation into the Operations table.
func (r *registry) InsertOperation(ctx context.Context, lro *model.LRO) (*model.LRO, error) {
	if err := validateLRO(lro); err != nil {
		return nil, fmt.Errorf("LRO validation failed: %w", err)
	}

	// Scan the database-generated timestamps back into the struct.
	err := r.db.QueryRowContext(ctx, insertOperationQuery, lro.OperationID, lro.Status, lro.Type, lro.RequestJSON).Scan(&lro.CreatedAt, &lro.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			return nil, fmt.Errorf("%w: %s", ErrOperationAlreadyExists, lro.OperationID)
		}
		return nil, fmt.Errorf("failed to insert operation with ID %s: %w", lro.OperationID, err)
	}

	return lro, nil
}

// validateSubscriptionForInsert checks for required fields before inserting a subscription.
func validateSubscriptionForInsert(sub *model.Subscription) error {
	if sub == nil {
		return errors.New("subscription cannot be nil for insert")
	}
	if sub.SubscriberID == "" {
		return errors.New("subscription SubscriberID is required")
	}
	if sub.KeyID == "" {
		return errors.New("subscription KeyID is required")
	}
	// Add other essential field checks as necessary, e.g., Status, Type, Domain.
	return nil
}

// InsertSubscription inserts a new subscription record into the database.
// It expects the database to handle 'created_at' and 'updated_at' timestamps.
func (r *registry) InsertSubscription(ctx context.Context, sub *model.Subscription) (*model.Subscription, error) {
	if err := validateSubscriptionForInsert(sub); err != nil {
		return nil, fmt.Errorf("subscription validation failed: %w", err)
	}
	var locationJSON sql.NullString
	if sub.Location != nil {
		locBytes, err := json.Marshal(sub.Location)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal subscription location for insert: %w", err)
		}
		locationJSON = sql.NullString{String: string(locBytes), Valid: true}
	}

	err := r.db.QueryRowContext(ctx, insertOnlySubscriptionQuery,
		sub.SubscriberID, sub.URL, sub.Type, sub.Domain, locationJSON, sub.KeyID,
		sub.SigningPublicKey, sub.EncrPublicKey, sub.ValidFrom, sub.ValidUntil,
		sub.Status, sub.Nonce,
	).Scan(&sub.Created, &sub.Updated) // Scan back the DB-generated timestamps

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			return nil, fmt.Errorf("%w: subscriber_id '%s', key_id '%s'", ErrSubscriptionConflict, sub.SubscriberID, sub.KeyID)
		}
		return nil, fmt.Errorf("failed to insert subscription for subscriber_id '%s', key_id '%s': %w", sub.SubscriberID, sub.KeyID, err)
	}
	return sub, nil
}

const getSubscriberSigningKeyQuery = `
	SELECT signing_public_key FROM subscriptions
	WHERE subscriber_id = $1 AND domain = $2 AND type = $3 AND key_id = $4 AND status = 'SUBSCRIBED'
`

// GetSubscriberSigningKey fetches the signing public key for a given subscriber_id and key_id.
func (r *registry) GetSubscriberSigningKey(ctx context.Context, subscriberID string, domain string, role model.Role, keyID string) (string, error) {
	var publicKey string
	err := r.db.QueryRowContext(ctx, getSubscriberSigningKeyQuery, subscriberID, domain, role, keyID).Scan(&publicKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%w: for subscriber_id '%s', domain '%s', type '%s', key_id '%s'", ErrSubscriberKeyNotFound, subscriberID, domain, role, keyID)
		}
		return "", fmt.Errorf("failed to query subscriber signing key: %w", err)
	}
	return publicKey, nil
}

const getOperationQuery = `
	SELECT operation_id, status, type, request_json, result_json, error_data_json, created_at, updated_at
	FROM Operations
	WHERE operation_id = $1`

// GetOperation retrieves a specific LRO from the database by its ID. (No changes needed here)
func (r *registry) GetOperation(ctx context.Context, id string) (*model.LRO, error) {
	lro := &model.LRO{}
	var resultJSON, errorDataJSON sql.NullString

	err := r.db.QueryRowContext(ctx, getOperationQuery, id).Scan(
		&lro.OperationID,
		&lro.Status,
		&lro.Type,
		&lro.RequestJSON,
		&resultJSON,
		&errorDataJSON,
		&lro.CreatedAt,
		&lro.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperationNotFound
		}
		return nil, fmt.Errorf("failed to get operation with ID %s: %w", id, err)
	}
	if resultJSON.Valid {
		lro.ResultJSON = []byte(resultJSON.String)
	}
	if errorDataJSON.Valid {
		lro.ErrorDataJSON = []byte(errorDataJSON.String)
	}

	return lro, nil
}

const getSubscriberEncryptionKeyQuery = `
	SELECT encr_public_key FROM subscriptions
	WHERE subscriber_id = $1 AND key_id = $2 AND status = 'SUBSCRIBED'
`

// EncryptionKey fetches the encryption public key for a given subscriber_id and key_id.
func (r *registry) EncryptionKey(ctx context.Context, subscriberID string, keyID string) (string, error) {
	var publicKey string
	err := r.db.QueryRowContext(ctx, getSubscriberEncryptionKeyQuery, subscriberID, keyID).Scan(&publicKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%w: for subscriber_id '%s', key_id '%s'", ErrEncrKeyNotFound, subscriberID, keyID)
		}
		return "", fmt.Errorf("failed to query subscriber encryption key: %w", err)
	}
	return publicKey, nil
}

// UpdateOperation updates an existing LRO record in the database.
func (r *registry) UpdateOperation(ctx context.Context, lro *model.LRO) (*model.LRO, error) {
	if lro == nil {
		return nil, errors.New("lro cannot be nil")
	}
	if lro.OperationID == "" {
		return nil, errors.New("lro OperationID cannot be empty for update")
	}

	var resultJSON, errorDataJSON sql.NullString
	if lro.ResultJSON != nil {
		resultJSON = sql.NullString{String: string(lro.ResultJSON), Valid: true}
	}
	if lro.ErrorDataJSON != nil {
		errorDataJSON = sql.NullString{String: string(lro.ErrorDataJSON), Valid: true}
	}

	err := r.db.QueryRowContext(ctx, updateOperationQuery,
		lro.OperationID, lro.Status, resultJSON, errorDataJSON, lro.RetryCount,
	).Scan(&lro.CreatedAt, &lro.UpdatedAt, &lro.Type, &lro.RequestJSON) // Scan back all returned fields

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperationNotFound
		}
		return nil, fmt.Errorf("failed to update operation %s: %w", lro.OperationID, err)
	}
	return lro, nil
}

// UpsertSubscriptionAndLRO performs an upsert on the subscriptions table and an update on the Operations table
// within the same database transaction. Timestamps are handled by the database.
func (r *registry) UpsertSubscriptionAndLRO(ctx context.Context, sub *model.Subscription, lro *model.LRO) (*model.Subscription, *model.LRO, error) {
	if err := r.validateUpsertInputs(sub, lro); err != nil {
		return nil, nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.ErrorContext(ctx, "transaction rollback failed", "error", err)
		}
	}()

	if err := r.upsertSubscription(ctx, tx, sub); err != nil {
		return nil, nil, err
	}

	if err := r.updateLRO(ctx, tx, lro); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return sub, lro, nil
}

// validateUpsertInputs checks the validity of subscription and LRO inputs.
func (r *registry) validateUpsertInputs(sub *model.Subscription, lro *model.LRO) error {
	if sub == nil {
		return errors.New("subscription cannot be nil")
	}
	if lro == nil {
		return errors.New("lro cannot be nil")
	}
	if err := validateLRO(lro); err != nil {
		return fmt.Errorf("LRO validation failed: %w", err)
	}
	if sub.SubscriberID == "" || sub.KeyID == "" {
		return errors.New("subscriberID and keyID are required for subscription")
	}
	return nil
}

// upsertSubscription handles the database upsert operation for a subscription within a transaction.
func (r *registry) upsertSubscription(ctx context.Context, tx *sql.Tx, sub *model.Subscription) error {
	var locationJSON sql.NullString
	if sub.Location != nil {
		locBytes, err := json.Marshal(sub.Location)
		if err != nil {
			return fmt.Errorf("failed to marshal subscription location: %w", err)
		}
		locationJSON = sql.NullString{String: string(locBytes), Valid: true}
	}

	err := tx.QueryRowContext(ctx, upsertSubscriptionQuery,
		sub.SubscriberID, sub.URL, sub.Type, sub.Domain, locationJSON, sub.KeyID,
		sub.SigningPublicKey, sub.EncrPublicKey, sub.ValidFrom, sub.ValidUntil,
		sub.Status,
	).Scan(&sub.Created, &sub.Updated) // Scan back the DB-generated timestamps

	if err != nil {
		return fmt.Errorf("failed to upsert subscription: %w", err)
	}
	return nil
}

func (r *registry) updateLRO(ctx context.Context, tx *sql.Tx, lro *model.LRO) error {
	var resultJSON, errorDataJSON sql.NullString
	if lro.ResultJSON != nil {
		resultJSON = sql.NullString{String: string(lro.ResultJSON), Valid: true}
	}
	if lro.ErrorDataJSON != nil {
		errorDataJSON = sql.NullString{String: string(lro.ErrorDataJSON), Valid: true}
	}

	err := tx.QueryRowContext(ctx, updateOperationQuery,
		lro.OperationID, lro.Status, resultJSON, errorDataJSON, lro.RetryCount,
	).Scan(&lro.CreatedAt, &lro.UpdatedAt, &lro.Type, &lro.RequestJSON) // Scan back all returned fields

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to update LRO %s (not found): %w", lro.OperationID, ErrOperationNotFound)
		}
		return fmt.Errorf("failed to update LRO %s in transaction: %w", lro.OperationID, err)
	}
	return nil
}
