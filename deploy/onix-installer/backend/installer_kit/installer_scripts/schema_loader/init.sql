-- Copyright 2025 Google LLC
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--     http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

-- ONIX Registry Database Initialization - Final Version

-- Define ENUM types for statuses to ensure data integrity and efficiency.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'subscriber_status_enum') THEN
        CREATE TYPE subscriber_status_enum AS ENUM ('INITIATED', 'UNDER_SUBSCRIPTION', 'SUBSCRIBED', 'INVALID_SSL', 'UNSUBSCRIBED');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'operation_status_enum') THEN
        CREATE TYPE operation_status_enum AS ENUM ('PENDING', 'APPROVED', 'REJECTED', 'FAILURE');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'operation_type_enum') THEN
        CREATE TYPE operation_type_enum AS ENUM ('CREATE_SUBSCRIPTION', 'UPDATE_SUBSCRIPTION');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'subscriber_type_enum') THEN
        CREATE TYPE subscriber_type_enum AS ENUM ('BAP', 'BPP', 'BG', 'REGISTRY');
    END IF;
END$$;

-- Subscribers Table:
CREATE TABLE IF NOT EXISTS subscriptions (
    subscriber_id VARCHAR(255) NOT NULL,
    type subscriber_type_enum NOT NULL,
    domain VARCHAR(255) NOT NULL,
    location JSONB,
    signing_public_key TEXT NOT NULL,
    encr_public_key TEXT NOT NULL,
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL,
    valid_until TIMESTAMP WITH TIME ZONE NOT NULL,
    status subscriber_status_enum NOT NULL,
    url VARCHAR(2048) NOT NULL,
    key_id VARCHAR(255) NOT NULL,
    -- This DEFAULT value handles the creation timestamp automatically on INSERT.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    nonce VARCHAR(255),
    extended_attributes JSONB,
    PRIMARY KEY (subscriber_id, domain, type)
);

-- Indexes for subscriptions table:
CREATE INDEX IF NOT EXISTS idx_subscribers_key_id ON subscriptions (key_id);
CREATE INDEX IF NOT EXISTS idx_subscribers_status ON subscriptions (status);
CREATE INDEX IF NOT EXISTS Idx_subscribers_location_city_country ON subscriptions USING BTREE ((location ->> 'city'), (location ->> 'country'));


-- Operations Table:
CREATE TABLE IF NOT EXISTS Operations (
    operation_id VARCHAR(255) PRIMARY KEY,
    status operation_status_enum NOT NULL,
    type operation_type_enum NOT NULL,
    request_json JSONB NOT NULL,
    result_json JSONB,
    error_data_json JSONB,
    retry_count INTEGER DEFAULT 0,
    -- This DEFAULT value handles the creation timestamp automatically on INSERT.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for Operations table:
CREATE INDEX IF NOT EXISTS Idx_operations_status ON Operations (status);
CREATE INDEX IF NOT EXISTS Idx_operations_updated_at ON Operations (updated_at);

--------------------------------------------------------------------------------
-- AUTO-UPDATE TIMESTAMP LOGIC
--------------------------------------------------------------------------------

-- This function is for the 'updated_at' column ONLY.
-- It runs on UPDATE operations.
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = NOW();
   RETURN NEW;
END;
$$ language 'plpgsql';

-- Attach the trigger to the 'subscriptions' table for UPDATEs.
DROP TRIGGER IF EXISTS set_updated_at_on_subscriptions ON subscriptions;
CREATE TRIGGER set_updated_at_on_subscriptions
BEFORE UPDATE ON subscriptions
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Attach the trigger to the 'Operations' table for UPDATEs.
DROP TRIGGER IF EXISTS set_updated_at_on_Operations ON Operations;
CREATE TRIGGER set_updated_at_on_Operations
BEFORE UPDATE ON Operations
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();