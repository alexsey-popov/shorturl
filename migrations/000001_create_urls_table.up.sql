-- migrations/000001_create_url_table.up.sql
-- Создание таблицы ссылок
CREATE TABLE urls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prefix VARCHAR(255) NOT NULL,
    original_url TEXT NOT NULL
);

-- Индекс уникальности для префикса
CREATE UNIQUE INDEX idx_urls_prefix_unique ON urls(prefix);


-- Индекс уникальности для оригинальной ссылки
CREATE UNIQUE INDEX idx_urls_original_url_unique ON urls(original_url);