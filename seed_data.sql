-- ============================================================
-- НАЧАЛЬНЫЕ ДАННЫЕ ДЛЯ BEAUTY BOT
-- ============================================================
-- Описание: Стартовый прайс услуг для бьюти-мастера
--
-- ВАЖНО: Услуги сейчас захардкожены в коде (markups.go:42-50)
-- Этот файл — для справки и будущего расширения функционала
-- ============================================================

-- ============================================
-- СПРАВОЧНАЯ ИНФОРМАЦИЯ: ТЕКУЩИЕ УСЛУГИ
-- ============================================

-- На данный момент в боте доступны следующие услуги (захардкожены):
--
-- 1. Маникюр
-- 2. Педикюр
-- 3. Наращивание ногтей
-- 4. Покрытие гель-лак
--
-- Чтобы изменить список услуг, редактируйте файл:
-- internal/delivery/telegram/markups.go -> функция BuildServicesMenu()

-- ============================================
-- ПРИМЕР: ТАБЛИЦА УСЛУГ (ДЛЯ БУДУЩЕГО)
-- ============================================

-- Если в будущем понадобится хранить услуги в БД, используйте:

-- CREATE TABLE IF NOT EXISTS services (
--     id SERIAL PRIMARY KEY,
--     name VARCHAR(255) NOT NULL UNIQUE,
--     description TEXT,
--     price_min INTEGER,
--     price_max INTEGER,
--     duration_minutes INTEGER DEFAULT 60,
--     is_active BOOLEAN DEFAULT true,
--     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
--     updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );

-- INSERT INTO services (name, description, price_min, price_max, duration_minutes) VALUES
--     ('Маникюр', 'Классический маникюр с обработкой кутикулы', 1500, 2000, 60),
--     ('Педикюр', 'Аппаратный педикюр с покрытием', 2000, 2500, 90),
--     ('Наращивание ногтей', 'Наращивание ногтей гелем', 3000, 4000, 120),
--     ('Покрытие гель-лак', 'Покрытие ногтей гель-лаком', 1200, 1800, 45);

-- ============================================
-- ТЕСТОВЫЕ ДАННЫЕ (ДЛЯ РАЗРАБОТКИ)
-- ============================================

-- Раскомментируйте для создания тестовых записей:

-- -- Тестовый пользователь
-- INSERT INTO users (id, username) VALUES
--     (123456789, 'test_user')
-- ON CONFLICT (id) DO NOTHING;

-- -- Тестовая бронь
-- INSERT INTO bookings (user_id, service_name, user_name, phone, date, time_slot, comment, status)
-- VALUES
--     (123456789, 'Маникюр', 'Анна Иванова', '+79991234567', '2026-09-25', '14:00-15:00', 'Хочу красный лак', 'confirmed')
-- ON CONFLICT DO NOTHING;

-- ============================================
-- ПРОВЕРКА ДАННЫХ
-- ============================================

-- Количество пользователей
SELECT COUNT(*) as total_users FROM users;

-- Количество активных броней
SELECT COUNT(*) as active_bookings FROM bookings WHERE status = 'confirmed';

-- Все подтвержденные бронирования
SELECT
    b.id,
    b.user_name,
    b.phone,
    b.service_name,
    TO_CHAR(b.date, 'DD.MM.YYYY') as date,
    b.time_slot,
    b.status,
    b.created_at
FROM bookings b
WHERE b.status = 'confirmed'
ORDER BY b.date DESC, b.time_slot;
