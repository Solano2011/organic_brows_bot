# 🎯 Обновление услуг: Брови и ресницы

## ✅ Что сделано

### 1. Обновлены услуги в Telegram-боте
**Файл:** `internal/delivery/telegram/markups.go`

Добавлены inline-кнопки для всех 16 услуг по категориям:
- **Organic brow** (2 услуги)
- **Брови** (5 услуг)
- **Ресницы** (2 услуги)
- **Дополнительно** (1 услуга)
- **Комбо** (3 услуги)

### 2. Обновлён WebApp (карточки услуг)
**Файл:** `webapp/templates/base.html`

Заменены все карточки услуг с ногтей на брови/ресницы. Добавлены заголовки категорий.

### 3. Добавлены стили для категорий
**Файл:** `static/css/style.css`

Добавлен класс `.category-header` для визуального разделения категорий.

---

## 📋 Полный список услуг

### Категория: Organic brow
1. **Organic brow 🍈 + коррекция** — 1500 ₽, 60 мин
2. **Organic brow 🍈 + натуральное оформление бровей** — 2000 ₽, 90 мин

### Категория: брови
3. **Коррекция бровей** — 1000 ₽, 60 мин
4. **Ламинирование бровей** — 1600 ₽, 60 мин
5. **Натуральное оформление бровей 🐚** — 1600 ₽, 60 мин
6. **Осветление бровей** — 1800 ₽, 80 мин
7. **Полный комплекс ламинирования бровей** — 2000 ₽, 60 мин

### Категория: ресницы
8. **Ламинирование ресниц + снятие наращенных ресниц** — 2000 ₽, 80 мин
9. **Ламинирование ресниц 🐚** — 1800 ₽, 60 мин

### Категория: Дополнительно
10. **Удаление пушка на губой** — 300 ₽, 10 мин

### Категория: комбо
11. **Organic brow 🍈 + оформление бровей + ламинирование ресниц** — 3500 ₽, 120 мин
12. **Ламинирование ресниц + ламинирование бровей** — 3400 ₽, 120 мин
13. **Ламинирование ресниц + натуральное оформление бровей 🪞** — 3000 ₽, 120 мин

---

## 📸 Требуются изображения

Создайте папку `/static/img/` и добавьте изображения для каждой услуги:

```
/static/img/
├── organic_brow.jpg
├── organic_brow_styling.jpg
├── brow_correction.jpg
├── brow_lamination.jpg
├── brow_styling.jpg
├── brow_bleaching.jpg
├── brow_full_complex.jpg
├── lash_lamination.jpg
├── lash_lamination_removal.jpg
├── lip_hair_removal.jpg
├── combo_organic_lash.jpg
├── combo_lash_brow.jpg
└── combo_lash_styling.jpg
```

**Временное решение:** Если изображений нет, можно использовать одну заглушку для всех услуг:
```html
<img src="/static/img/placeholder.jpg" alt="..." class="service-image">
```

---

## 🗄️ База данных

**Важно:** В этом проекте **НЕТ** таблиц `services` и `categories` в БД.

Услуги захардкожены в коде:
- **Telegram-бот:** `markups.go` — inline-кнопки
- **WebApp:** `base.html` — HTML-карточки

Бронирования сохраняются в таблицу `bookings` с полем `service_name` (VARCHAR), которое хранит название услуги как текст.

### Очистка старых записей (опционально)

Если нужно удалить все старые бронирования с услугами "Маникюр", "Педикюр" и т.д.:

```sql
-- Посмотреть, что есть
SELECT service_name, COUNT(*) FROM bookings GROUP BY service_name;

-- Удалить все старые записи
DELETE FROM bookings 
WHERE service_name IN (
    'Маникюр', 
    'Педикюр', 
    'Наращивание ногтей', 
    'Покрытие гель-лак'
);

-- Или очистить таблицу полностью
TRUNCATE TABLE bookings;
```

---

## 🚀 Запуск обновлённого бота

```bash
# 1. Скомпилировать Go-код
go build -o bot.exe cmd/bot/main.go

# 2. Запустить бота
./bot.exe

# 3. Проверить WebApp в Telegram
# Откройте бота, нажмите "💅 Записаться"
```

---

## ✏️ Как добавить/изменить услугу в будущем

### 1. Telegram-бот (inline-кнопки)
Файл: `internal/delivery/telegram/markups.go`

```go
btnNewService := m.Data("🌟 Новая услуга", "service", "Полное название услуги")
m.Inline(
    m.Row(btnNewService),
    // ...
)
```

### 2. WebApp (карточка)
Файл: `webapp/templates/base.html`

```html
<div class="service-card" onclick="selectServiceFromMenu('Полное название услуги')">
    <img src="/static/img/new_service.jpg" alt="Новая услуга" class="service-image">
    <div class="service-info">
        <div class="service-name">Новая услуга</div>
        <div class="service-price">2000 ₽ · 90 мин</div>
    </div>
</div>
```

**Важно:** Название услуги в `onclick="selectServiceFromMenu('...')` должно **точно совпадать** с названием в кнопке Telegram!

---

## 🎉 Готово!

Все услуги обновлены. База данных не требует изменений — `bookings.service_name` автоматически сохранит новые названия услуг.
