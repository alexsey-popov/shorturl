-- migrations/000001_create_url_table.up.sql
-- Откат создания таблицы ссылок
DROP INDEX IF EXISTS idx_urls_original_url;
DROP INDEX IF EXISTS idx_urls_prefix;
DROP TABLE IF EXISTS urls;