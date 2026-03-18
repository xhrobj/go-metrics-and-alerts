-- 1. удаляем колонку с временем создания
ALTER TABLE metrics
DROP COLUMN created_at;

-- 2.1. возвращаем тип колонки type обратно в TEXT
ALTER TABLE metrics
ALTER COLUMN type TYPE TEXT
USING type::text;

-- 2.2. возвращаем CHECK-ограничение
ALTER TABLE metrics
ADD CONSTRAINT metrics_type_check
CHECK (type IN ('gauge', 'counter'));

-- 2.3. удаляем enum для типа метрики
DROP TYPE metric_type;
