-- ============================================
-- ТАБЛИЦА ПОЛЬЗОВАТЕЛЕЙ
-- ============================================
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    username VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- ТАБЛИЦА БРОНИРОВАНИЙ
-- ============================================
CREATE TABLE IF NOT EXISTS bookings (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    service_name VARCHAR(255) NOT NULL,
    user_name VARCHAR(255),
    phone VARCHAR(50),
    date DATE NOT NULL,
    time_slot VARCHAR(50) NOT NULL,
    comment TEXT DEFAULT '',
    status VARCHAR(50) DEFAULT 'confirmed' CHECK (status IN ('draft', 'confirmed', 'cancelled')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- ИНДЕКСЫ ДЛЯ ПРОИЗВОДИТЕЛЬНОСТИ
-- ============================================

-- Для быстрого поиска броней пользователя по статусу
CREATE INDEX idx_bookings_user_status ON bookings(user_id, status, created_at DESC);

-- Для проверки доступности слотов (глобально по дате)
CREATE INDEX idx_bookings_availability ON bookings(date, status, time_slot);

-- Для выборки подтвержденных броней по датам
CREATE INDEX idx_bookings_date ON bookings(date DESC) WHERE status = 'confirmed';

-- Для быстрого поиска всех броней пользователя
CREATE INDEX idx_bookings_user_id ON bookings(user_id);

-- ============================================
-- ТРИГГЕРЫ ДЛЯ АВТООБНОВЛЕНИЯ updated_at
-- ============================================

-- Функция для обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Триггер для users
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Триггер для bookings
CREATE TRIGGER update_bookings_updated_at
    BEFORE UPDATE ON bookings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
