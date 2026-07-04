-- migrations/000002_add_user_id_into_urls.up.sql
-- Создание поля в user_id таблице ссылок
ALTER TABLE urls ADD COLUMN user_id UUID NULL;

-- Индекс для быстрой фильтрации по значению
CREATE INDEX idx_urls_user_id ON urls(user_id);