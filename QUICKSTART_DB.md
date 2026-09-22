# 🚀 Быстрый старт — База данных

## Минимальные шаги для запуска

### 1. Создайте БД и пользователя
```bash
psql -U postgres
```
```sql
CREATE USER beauty_user WITH PASSWORD 'your_password';
CREATE DATABASE client_beauty_db OWNER beauty_user;
GRANT ALL PRIVILEGES ON DATABASE client_beauty_db TO beauty_user;
\q
```

### 2. Инициализируйте структуру
```bash
psql -U beauty_user -d client_beauty_db -f init_database.sql
```

### 3. Настройте .env
```env
BOT_TOKEN=ваш_токен
ADMIN_ID=ваш_id
DB_HOST=localhost
DB_PORT=5432
DB_USER=beauty_user
DB_PASSWORD=your_password
DB_NAME=client_beauty_db
DB_SSLMODE=disable  # для локальной разработки
WEBAPP_URL=https://your-domain.com
```

### 4. Запустите бота
```bash
go run cmd/bot/main.go
```

---

## 📊 Ключевые особенности БД

### Типы данных
- ✅ `date` — тип **DATE** (не VARCHAR!)
- ✅ `status` — с CHECK constraint ('draft', 'confirmed', 'cancelled')
- ✅ `updated_at` — автообновление через триггер

### Индексы (оптимизированы для production)
- `idx_bookings_user_status` — поиск броней пользователя
- `idx_bookings_availability` — проверка занятых слотов по дате
- `idx_bookings_date` — выборка броней по датам
- `idx_bookings_user_id` — все брони пользователя

### Формат дат
- **В БД:** тип `DATE` (2026-09-25)
- **В коде Go:** строка `DD.MM.YYYY` (25.09.2026)
- **Преобразование:** `TO_DATE($1, 'DD.MM.YYYY')` в SQL

### Услуги
Захардкожены в `internal/delivery/telegram/markups.go`:
- 💅 Маникюр
- 🦶 Педикюр
- 💎 Наращивание ногтей
- ✨ Покрытие гель-лак

---

## 🔍 Полезные команды

### Проверка структуры
```sql
\d bookings
```

### Посмотреть все записи
```sql
SELECT 
    user_name, 
    service_name, 
    TO_CHAR(date, 'DD.MM.YYYY') as date, 
    time_slot 
FROM bookings 
WHERE status = 'confirmed';
```

### Очистить тестовые данные
```sql
TRUNCATE TABLE bookings, users CASCADE;
```

---

## 📖 Полная документация

См. `DATABASE_SETUP.md` для детальной инструкции по развертыванию, безопасности и обслуживанию.
