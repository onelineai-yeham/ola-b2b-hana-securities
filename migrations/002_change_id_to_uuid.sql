-- Migration: Change id from BIGSERIAL to UUID
-- Run on gold database (hana_securities)

-- 1. Add new UUID column
ALTER TABLE gold.translated_news 
ADD COLUMN IF NOT EXISTS uuid_id UUID DEFAULT gen_random_uuid();

-- 2. Generate UUIDs for existing records
UPDATE gold.translated_news SET uuid_id = gen_random_uuid() WHERE uuid_id IS NULL;

-- 3. Drop existing primary key constraint
ALTER TABLE gold.translated_news DROP CONSTRAINT IF EXISTS translated_news_pkey;

-- 4. Drop old id column
ALTER TABLE gold.translated_news DROP COLUMN IF EXISTS id;

-- 5. Rename uuid_id to id
ALTER TABLE gold.translated_news RENAME COLUMN uuid_id TO id;

-- 6. Add primary key on new id
ALTER TABLE gold.translated_news ADD PRIMARY KEY (id);

-- 7. Set NOT NULL constraint
ALTER TABLE gold.translated_news ALTER COLUMN id SET NOT NULL;

-- 8. Set DEFAULT for new inserts
ALTER TABLE gold.translated_news ALTER COLUMN id SET DEFAULT gen_random_uuid();

-- Verify
-- SELECT id, source, source_news_id FROM gold.translated_news LIMIT 5;
