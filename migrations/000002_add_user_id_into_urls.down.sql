-- migrations/000002_add_user_id_into_urls.down.sql
-- Откат создания поля в таблице ссылок
DROP INDEX IF EXISTS idx_urls_user_id;
ALTER TABLE urls DROP COLUMN user_id;