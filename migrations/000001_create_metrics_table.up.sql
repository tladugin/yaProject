-- Создание таблицы для метрик gauge
CREATE TABLE IF NOT EXISTS gauge_metrics (
                                             id SERIAL PRIMARY KEY,
                                             name VARCHAR(255) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    UNIQUE (name)
    );

-- Создание таблицы для метрик counter
CREATE TABLE IF NOT EXISTS counter_metrics (
                                               id SERIAL PRIMARY KEY,
                                               name VARCHAR(255) NOT NULL,
    value BIGINT NOT NULL,
    UNIQUE (name)
    );

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_gauge_name ON gauge_metrics(name);
CREATE INDEX IF NOT EXISTS idx_counter_name ON counter_metrics(name);