-- migrations/000003_add_is_deleted_into_urls.up.up.sql
-- Создание поля в is_deleted таблице ссылок
ALTER TABLE urls ADD COLUMN is_deleted BOOLEAN DEFAULT false;