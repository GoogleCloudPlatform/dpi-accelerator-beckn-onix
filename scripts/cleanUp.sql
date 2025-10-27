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

-- ONIX Registry Database Cleanup.
-- ====================================================================
-- WARNING: DESTRUCTIVE SCRIPT
-- This script will permanently delete all tables, types, and functions
-- related to the ONIX Registry. ALL DATA WILL BE LOST.
-- Use with extreme caution.
-- ====================================================================

DO $$
BEGIN
   RAISE NOTICE 'Starting ONIX Registry cleanup script...';
END;
$$;


-- Step 1: Remove Triggers from the tables.
-- This must be done before dropping the function or the tables.
RAISE NOTICE 'Dropping triggers...';
DROP TRIGGER IF EXISTS set_updated_at_on_subscriptions ON subscriptions;
DROP TRIGGER IF EXISTS set_updated_at_on_Operations ON Operations;


-- Step 2: Remove the Trigger Function.
-- This can only be done after the triggers that use it are removed.
RAISE NOTICE 'Dropping trigger function...';
DROP FUNCTION IF EXISTS update_updated_at_column();


-- Step 3: Drop the Tables.
-- This will also automatically remove the indexes associated with them.
-- We use CASCADE to remove any other dependent objects automatically.
RAISE NOTICE 'Dropping tables...';
DROP TABLE IF EXISTS subscriptions CASCADE;
DROP TABLE IF EXISTS Operations CASCADE;


-- Step 4: Drop the custom ENUM types.
-- This can only be done after the table columns that use them are dropped.
RAISE NOTICE 'Dropping ENUM types...';
DROP TYPE IF EXISTS subscriber_status_enum CASCADE;
DROP TYPE IF EXISTS operation_status_enum CASCADE;
DROP TYPE IF EXISTS operation_type_enum CASCADE;
DROP TYPE IF EXISTS subscriber_type_enum CASCADE;


DO $$
BEGIN
   RAISE NOTICE 'ONIX Registry cleanup script finished successfully.';
END;
$$;