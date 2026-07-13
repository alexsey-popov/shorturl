-- migrations/000003_add_is_deleted_into_urls.up.up.sql
-- Откат создания поля в is_deleted таблице ссылок
ALTER TABLE urls DROP COLUMN is_deleted;