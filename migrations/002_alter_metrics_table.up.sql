-- 1.1. создаём enum для типа метрики
CREATE TYPE metric_type AS ENUM ('gauge', 'counter');

-- 1.2. удаляем старое ограничение CHECK
ALTER TABLE metrics
DROP CONSTRAINT metrics_type_check;

-- 1.3. меняем тип колонки type на enum
ALTER TABLE metrics
ALTER COLUMN type TYPE metric_type
USING type::metric_type;

-- 2. добавляем колонку с временем создания
ALTER TABLE metrics
ADD COLUMN created_at TIMESTAMPTZ DEFAULT now();
