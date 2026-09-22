-- ============================================================
-- ИНИЦИАЛИЗАЦИЯ БАЗЫ ДАННЫХ ДЛЯ BEAUTY BOT (PRODUCTION)
-- ============================================================
-- База: client_beauty_db
-- Дата создания: 2026-09-22
-- Описание: Полная структура БД для бота записи к бьюти-мастеру
-- ============================================================

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

-- ============================================
-- КОММЕНТАРИИ К ТАБЛИЦАМ
-- ============================================

COMMENT ON TABLE users IS 'Пользователи Telegram-бота';
COMMENT ON TABLE bookings IS 'Бронирования услуг у мастера';

COMMENT ON COLUMN bookings.status IS 'Статус бронирования: draft (черновик), confirmed (подтверждено), cancelled (отменено)';
COMMENT ON COLUMN bookings.date IS 'Дата записи в формате DATE (преобразуется из DD.MM.YYYY)';
COMMENT ON COLUMN bookings.time_slot IS 'Временной слот (например, "10:00-11:00")';

-- ============================================
-- СТАТИСТИКА И ПРОВЕРКА
-- ============================================

-- Запрос для проверки структуры
SELECT
    table_name,
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name IN ('users', 'bookings')
ORDER BY table_name, ordinal_position;

-- Запрос для проверки индексов
SELECT
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
  AND tablename IN ('users', 'bookings')
ORDER BY tablename, indexname;
