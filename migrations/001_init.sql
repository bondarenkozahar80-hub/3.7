-- Создание таблицы пользователей
CREATE TABLE IF NOT EXISTS users (
    username VARCHAR(50) PRIMARY KEY,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'manager', 'viewer', 'auditor')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы товаров
CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    quantity INTEGER NOT NULL CHECK (quantity >= 0),
    price DECIMAL(10, 2) NOT NULL CHECK (price >= 0),
    location VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(50) REFERENCES users(username)
);

-- Создание таблицы истории изменений
CREATE TABLE IF NOT EXISTS item_history (
    id SERIAL PRIMARY KEY,
    item_id INTEGER REFERENCES items(id) ON DELETE CASCADE,
    action VARCHAR(20) NOT NULL CHECK (action IN ('CREATE', 'UPDATE', 'DELETE')),
    changed_by VARCHAR(50) REFERENCES users(username),
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    old_data JSONB,
    new_data JSONB,
    changes JSONB
);

-- Создание индексов для оптимизации
CREATE INDEX idx_items_name ON items(name);
CREATE INDEX idx_items_location ON items(location);
CREATE INDEX idx_item_history_item_id ON item_history(item_id);
CREATE INDEX idx_item_history_changed_at ON item_history(changed_at);
CREATE INDEX idx_item_history_changed_by ON item_history(changed_by);

-- Вставка тестовых пользователей
INSERT INTO users (username, password_hash, role) VALUES
    ('admin', '$2a$14$5Y5G5h5j5k5l5m5n5o5p5q5r5s5t5u5v5w5x5y5z5A5B5C5D5E5F5G5H5I', 'admin'),
    ('manager', '$2a$14$5Y5G5h5j5k5l5m5n5o5p5q5r5s5t5u5v5w5x5y5z5A5B5C5D5E5F5G5H5I', 'manager'),
    ('viewer', '$2a$14$5Y5G5h5j5k5l5m5n5o5p5q5r5s5t5u5v5w5x5y5z5A5B5C5D5E5F5G5H5I', 'viewer'),
    ('auditor', '$2a$14$5Y5G5h5j5k5l5m5n5o5p5q5r5s5t5u5v5w5x5y5z5A5B5C5D5E5F5G5H5I', 'auditor')
ON CONFLICT (username) DO NOTHING;
