# 🗄️ Инструкция по развертыванию базы данных

## 📋 Предварительные требования

- PostgreSQL 12 или выше
- Права на создание базы данных и пользователей
- Клиент `psql` или любой другой инструмент для работы с PostgreSQL

---

## 🚀 Шаг 1: Создание базы данных и пользователя

Подключитесь к PostgreSQL от имени суперпользователя (обычно `postgres`):

```bash
psql -U postgres
```

Выполните следующие команды:

```sql
-- Создаем пользователя для бота
CREATE USER beauty_user WITH PASSWORD 'your_secure_password_here';

-- Создаем базу данных
CREATE DATABASE client_beauty_db OWNER beauty_user;

-- Даем все права пользователю
GRANT ALL PRIVILEGES ON DATABASE client_beauty_db TO beauty_user;

-- Выходим
\q
```

---

## 🏗️ Шаг 2: Инициализация структуры БД

Подключитесь к новой базе данных:

```bash
psql -U beauty_user -d client_beauty_db
```

Или запустите SQL-скрипт напрямую:

```bash
psql -U beauty_user -d client_beauty_db -f init_database.sql
```

**Что создается:**
- ✅ Таблица `users` (пользователи Telegram)
- ✅ Таблица `bookings` (бронирования)
- ✅ 4 индекса для производительности
- ✅ Триггеры автообновления `updated_at`
- ✅ CHECK constraint для статусов броней

---

## 🔧 Шаг 3: Настройка переменных окружения

Отредактируйте файл `.env` в корне проекта:

```bash
# Telegram Bot Configuration
BOT_TOKEN=ваш_токен_бота
ADMIN_ID=ваш_telegram_id

# Database Configuration
DB_HOST=localhost                    # или IP вашего сервера PostgreSQL
DB_PORT=5432
DB_USER=beauty_user
DB_PASSWORD=your_secure_password_here
DB_NAME=client_beauty_db
DB_SSLMODE=require                   # или disable для локальной разработки

# Web Application
WEBAPP_URL=https://your-domain.com   # URL вашего WebApp
WEBAPP_PORT=8080
```

**⚠️ Важно:**
- Используйте **сильный пароль** для продакшена
- Для production сервера **обязательно** используйте `DB_SSLMODE=require`
- Не коммитьте `.env` в git (он уже в `.gitignore`)

---

## 🧪 Шаг 4: Проверка структуры

Выполните проверочные запросы:

```sql
-- Проверка таблиц
\dt

-- Проверка индексов
\di

-- Проверка триггеров
\dS update_*

-- Проверка структуры bookings
\d bookings
```

Ожидаемый результат для `bookings`:
```
                                          Table "public.bookings"
    Column    |            Type             | Collation | Nullable |                Default                
--------------+-----------------------------+-----------+----------+---------------------------------------
 id           | integer                     |           | not null | nextval('bookings_id_seq'::regclass)
 user_id      | bigint                      |           | not null | 
 service_name | character varying(255)      |           | not null | 
 user_name    | character varying(255)      |           |          | 
 phone        | character varying(50)       |           |          | 
 date         | date                        |           | not null | 
 time_slot    | character varying(50)       |           | not null | 
 comment      | text                        |           |          | ''::text
 status       | character varying(50)       |           |          | 'confirmed'::character varying
 created_at   | timestamp without time zone |           |          | CURRENT_TIMESTAMP
 updated_at   | timestamp without time zone |           |          | CURRENT_TIMESTAMP
```

---

## ✅ Шаг 5: Миграция (если есть старая БД)

Если у вас уже была старая версия базы с `date VARCHAR(10)`, используйте миграцию:

```bash
# Создайте резервную копию
pg_dump -U beauty_user client_beauty_db > backup_before_migration.sql

# Примените миграцию (если используете migrate tool)
migrate -path migrations -database "postgres://beauty_user:password@localhost:5432/client_beauty_db?sslmode=require" up
```

**Внимание:** Текущая миграция `000001_init.up.sql` предназначена для **чистой** базы.

---

## 🎯 Запуск бота

После настройки БД запустите бота:

```bash
go run cmd/bot/main.go
```

Проверьте логи на наличие ошибок подключения к БД. Успешное подключение не выводит ошибок.

---

## 📊 Полезные SQL-запросы

### Количество пользователей
```sql
SELECT COUNT(*) as total_users FROM users;
```

### Все активные бронирования
```sql
SELECT 
    b.user_name,
    b.phone,
    b.service_name,
    TO_CHAR(b.date, 'DD.MM.YYYY') as date,
    b.time_slot,
    b.created_at
FROM bookings b
WHERE b.status = 'confirmed'
ORDER BY b.date, b.time_slot;
```

### Занятые слоты на конкретную дату
```sql
SELECT time_slot 
FROM bookings 
WHERE date = '2026-09-25' AND status = 'confirmed'
ORDER BY time_slot;
```

### Статистика по услугам
```sql
SELECT 
    service_name,
    COUNT(*) as total_bookings
FROM bookings
WHERE status = 'confirmed'
GROUP BY service_name
ORDER BY total_bookings DESC;
```

---

## 🔒 Безопасность

### Рекомендации для продакшена:

1. **Ограничьте доступ к БД по IP**
   ```
   # В pg_hba.conf
   host    client_beauty_db    beauty_user    ваш_ip/32    scram-sha-256
   ```

2. **Используйте SSL-сертификаты**
   - Настройте PostgreSQL для работы с SSL
   - Установите `DB_SSLMODE=verify-full` в `.env`

3. **Регулярные бэкапы**
   ```bash
   # Ежедневный бэкап через cron
   pg_dump -U beauty_user client_beauty_db | gzip > backup_$(date +%Y%m%d).sql.gz
   ```

4. **Мониторинг**
   - Следите за размером БД
   - Настройте алерты на ошибки подключения
   - Проверяйте производительность запросов

---

## 🛠️ Обслуживание

### Очистка старых черновиков (рекомендуется запускать еженедельно)
```sql
DELETE FROM bookings 
WHERE status = 'draft' 
  AND created_at < NOW() - INTERVAL '7 days';
```

### Архивация старых записей (старше 1 года)
```sql
-- Создайте архивную таблицу
CREATE TABLE bookings_archive (LIKE bookings INCLUDING ALL);

-- Переместите старые записи
INSERT INTO bookings_archive 
SELECT * FROM bookings 
WHERE created_at < NOW() - INTERVAL '1 year';

DELETE FROM bookings 
WHERE created_at < NOW() - INTERVAL '1 year';
```

### Оптимизация индексов
```sql
REINDEX DATABASE client_beauty_db;
VACUUM ANALYZE;
```

---

## 📝 Изменение списка услуг

Услуги сейчас **захардкожены в коде**. Для изменения отредактируйте:

**Файл:** `internal/delivery/telegram/markups.go`

```go
func BuildServicesMenu() *tele.ReplyMarkup {
    m := &tele.ReplyMarkup{}

    // Добавьте свои услуги здесь
    btnManicure := m.Data("💅 Маникюр", "service", "Маникюр")
    btnPedicure := m.Data("🦶 Педикюр", "service", "Педикюр")
    btnExtension := m.Data("💎 Наращивание", "service", "Наращивание ногтей")
    btnGelPolish := m.Data("✨ Гель-лак", "service", "Покрытие гель-лак")

    m.Inline(
        m.Row(btnManicure, btnPedicure),
        m.Row(btnExtension, btnGelPolish),
        m.Row(BtnBackToMain),
    )
    return m
}
```

После изменений пересоберите и перезапустите бота:
```bash
go build -o bot cmd/bot/main.go
./bot
```

---

## 🆘 Проблемы и решения

### Ошибка: "password authentication failed"
- Проверьте пароль в `.env`
- Убедитесь, что пользователь создан: `psql -U postgres -c "\du"`

### Ошибка: "database does not exist"
- Создайте БД: `createdb -U postgres client_beauty_db -O beauty_user`

### Ошибка: "no pg_hba.conf entry"
- Добавьте правило в `pg_hba.conf`
- Перезапустите PostgreSQL: `sudo systemctl restart postgresql`

### Бот не подключается к БД
- Проверьте, что PostgreSQL запущен: `systemctl status postgresql`
- Проверьте файрвол: `telnet localhost 5432`
- Проверьте логи PostgreSQL: `/var/log/postgresql/postgresql-*.log`

---

## ✨ Готово!

База данных настроена и готова к работе. Успехов! 🎉
