-- =====================================================
-- Remove junyoung user from all databases
-- Run on onlineai-etl instance as admin (cloudsqlsuperuser)
-- =====================================================

-- 1. Revoke privileges on etl database
REVOKE ALL PRIVILEGES ON DATABASE etl FROM junyoung;

-- 2. Connect to etl and revoke schema privileges
\c etl
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA silver FROM junyoung;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA silver FROM junyoung;
REVOKE ALL PRIVILEGES ON SCHEMA silver FROM junyoung;
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM junyoung;
REVOKE ALL PRIVILEGES ON SCHEMA public FROM junyoung;

-- 3. Check for any owned objects
SELECT 
    n.nspname as schema,
    c.relname as name,
    c.relkind as type
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_roles r ON r.oid = c.relowner
WHERE r.rolname = 'junyoung';

-- 4. Reassign owned objects (if any) to onelineai
-- REASSIGN OWNED BY junyoung TO onelineai;

-- 5. Drop owned objects (if any)
-- DROP OWNED BY junyoung;

-- 6. Connect back to postgres and drop user
\c postgres
DROP USER IF EXISTS junyoung;

-- Verify
SELECT rolname FROM pg_roles WHERE rolname = 'junyoung';
