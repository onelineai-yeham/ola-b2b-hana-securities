-- =====================================================
-- Gold DB Setup Script for ola-b2b instance
-- Run this on ola-b2b PostgreSQL (10.35.64.2) as admin
-- =====================================================

-- 1. Create database
CREATE DATABASE hana_securities
    WITH OWNER = cloudsqlsuperuser
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.UTF-8'
    LC_CTYPE = 'en_US.UTF-8';

-- 2. Connect to hana_securities database
\c hana_securities

-- 3. Create user (if not exists)
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'ola_b2b_hana_securities') THEN
        CREATE USER ola_b2b_hana_securities WITH PASSWORD 'QeRkTzMpaLhWnSxB';
    END IF;
END
$$;

-- 4. Create gold schema
CREATE SCHEMA IF NOT EXISTS gold;

-- 5. Grant permissions
GRANT CONNECT ON DATABASE hana_securities TO ola_b2b_hana_securities;
GRANT USAGE ON SCHEMA gold TO ola_b2b_hana_securities;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA gold TO ola_b2b_hana_securities;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA gold TO ola_b2b_hana_securities;
ALTER DEFAULT PRIVILEGES IN SCHEMA gold GRANT ALL ON TABLES TO ola_b2b_hana_securities;
ALTER DEFAULT PRIVILEGES IN SCHEMA gold GRANT ALL ON SEQUENCES TO ola_b2b_hana_securities;

-- 6. Run the migration (creates unified translated_news table)
\i migrations/001_create_gold_tables.sql

-- 7. Verify setup
SELECT schemaname, tablename FROM pg_tables WHERE schemaname = 'gold';
