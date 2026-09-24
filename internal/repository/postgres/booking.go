package postgres

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"hookah-bot/internal/domain"

	"github.com/jackc/pgx/v5"
)

type BookingRepo struct {
	db *DB
}

func NewBookingRepo(db *DB) *BookingRepo {
	return &BookingRepo{db: db}
}

func (r *BookingRepo) SaveDraft(ctx context.Context, userID int64, serviceName string) error {
	// Гарантируем, что пользователь существует в таблице users
	_, err := r.db.Conn.Exec(ctx, `INSERT INTO users (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, userID)
	if err != nil {
		return err
	}

	// Удаляем старый черновик, если он был
	_, err = r.db.Conn.Exec(ctx, `DELETE FROM bookings WHERE user_id = $1 AND status = 'draft'`, userID)
	if err != nil {
		return err
	}

	// Создаем новый черновик (используем дату-заглушку '1970-01-01' для черновиков)
	_, err = r.db.Conn.Exec(ctx, `
        INSERT INTO bookings (user_id, service_name, time_slot, date, status)
        VALUES ($1, $2, '', '1970-01-01', 'draft')`,
		userID, serviceName,
	)
	return err
}

func (r *BookingRepo) SetDraftDate(ctx context.Context, userID int64, date string) error {
	// Преобразуем дату из формата YYYY-MM-DD (ISO 8601 от WebApp) в DATE
	cmdTag, err := r.db.Conn.Exec(ctx, `
        UPDATE bookings SET date = TO_DATE($1, 'YYYY-MM-DD')
        WHERE user_id = $2 AND status = 'draft'`,
		date, userID,
	)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrBookingNotFound
	}
	return nil
}

func (r *BookingRepo) SetDraftTimeAndContacts(ctx context.Context, userID int64, timeSlot string, name string, phone string, comment string) error {
	cmdTag, err := r.db.Conn.Exec(ctx, `
        UPDATE bookings SET time_slot = $1, user_name = $2, phone = $3, comment = $4
        WHERE user_id = $5 AND status = 'draft'`,
		timeSlot, name, phone, comment, userID,
	)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrBookingNotFound
	}
	return nil
}

func (r *BookingRepo) CompleteBooking(ctx context.Context, userID int64, timeSlot string, name string, phone string, comment string) (*domain.Booking, error) {
	tx, err := r.db.Conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 1. Получаем текущий черновик
	var serviceName string
	var dateVal time.Time
	err = tx.QueryRow(ctx, `
        SELECT service_name, date FROM bookings
        WHERE user_id = $1 AND status = 'draft'`, userID).Scan(&serviceName, &dateVal)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBookingNotFound
	} else if err != nil {
		return nil, err
	}

	// 2. Проверяем, не занято ли это время для данной услуги и даты
	var conflictID int
	err = tx.QueryRow(ctx, `
        SELECT id FROM bookings
        WHERE service_name = $1 AND time_slot = $2 AND date = $3 AND status = 'confirmed'
        FOR UPDATE`,
		serviceName, timeSlot, dateVal).Scan(&conflictID)

	if err == nil {
		return nil, domain.ErrTimeSlotTaken
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// 3. Обновляем статус черновика на confirmed
	var b domain.Booking
	var dateValResult time.Time
	err = tx.QueryRow(ctx, `
        UPDATE bookings
        SET time_slot = $1, user_name = $2, phone = $3, comment = $4, status = 'confirmed', created_at = $5
        WHERE user_id = $6 AND status = 'draft'
        RETURNING user_id, service_name, time_slot, date, user_name, phone, comment, created_at`,
		timeSlot, name, phone, comment, time.Now(), userID,
	).Scan(&b.UserID, &b.ServiceName, &b.TimeSlot, &dateValResult, &b.UserName, &b.Phone, &b.Comment, &b.CreatedAt)

	if err != nil {
		return nil, err
	}

	// Преобразуем DATE обратно в формат DD.MM.YYYY для отображения
	b.Date = dateValResult.Format("02.01.2006")

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &b, nil
}

func (r *BookingRepo) GetByUserID(ctx context.Context, userID int64) (*domain.Booking, error) {
	var b domain.Booking
	var id int
	var dateVal time.Time
	err := r.db.Conn.QueryRow(ctx, `
        SELECT id, user_id, service_name, time_slot, date, COALESCE(user_name, ''), COALESCE(phone, ''), COALESCE(comment, ''), created_at
        FROM bookings
        WHERE user_id = $1 AND status = 'confirmed'
        ORDER BY created_at DESC LIMIT 1`,
		userID,
	).Scan(&id, &b.UserID, &b.ServiceName, &b.TimeSlot, &dateVal, &b.UserName, &b.Phone, &b.Comment, &b.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBookingNotFound
	} else if err != nil {
		return nil, err
	}

	b.ID = strconv.Itoa(id)
	// Преобразуем DATE обратно в формат DD.MM.YYYY для отображения
	b.Date = dateVal.Format("02.01.2006")
	return &b, nil
}

func (r *BookingRepo) GetDraftByUserID(ctx context.Context, userID int64) (*domain.Booking, error) {
	var b domain.Booking
	var dateVal time.Time
	err := r.db.Conn.QueryRow(ctx, `
        SELECT user_id, service_name, COALESCE(time_slot, ''), date, COALESCE(user_name, ''), COALESCE(phone, ''), COALESCE(comment, ''), COALESCE(created_at, NOW())
        FROM bookings
        WHERE user_id = $1 AND status = 'draft'
        ORDER BY created_at DESC LIMIT 1`,
		userID,
	).Scan(&b.UserID, &b.ServiceName, &b.TimeSlot, &dateVal, &b.UserName, &b.Phone, &b.Comment, &b.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBookingNotFound
	} else if err != nil {
		return nil, err
	}

	// Преобразуем DATE обратно в формат DD.MM.YYYY
	// Для черновиков с датой-заглушкой '1970-01-01' оставляем пустую строку
	if dateVal.Year() == 1970 {
		b.Date = ""
	} else {
		b.Date = dateVal.Format("02.01.2006")
	}
	return &b, nil
}

func (r *BookingRepo) Delete(ctx context.Context, userID int64) error {
	_, err := r.db.Conn.Exec(ctx, `DELETE FROM bookings WHERE user_id = $1`, userID)
	return err
}

func (r *BookingRepo) DeleteConfirmed(ctx context.Context, userID int64) error {
	log.Printf("🗑️ Попытка удалить подтвержденные брони для userID=%d", userID)
	cmdTag, err := r.db.Conn.Exec(ctx, `DELETE FROM bookings WHERE user_id = $1 AND status = 'confirmed'`, userID)
	if err != nil {
		log.Printf("❌ Ошибка при удалении подтвержденных броней для userID=%d: %v", userID, err)
		return err
	}
	rowsAffected := cmdTag.RowsAffected()
	log.Printf("✅ Удалено подтвержденных броней для userID=%d: %d записей", userID, rowsAffected)
	if rowsAffected == 0 {
		log.Printf("⚠️ Не найдено подтвержденных броней для удаления у userID=%d", userID)
		return nil
	}
	return nil
}

func (r *BookingRepo) DeleteDraft(ctx context.Context, userID int64) error {
	_, err := r.db.Conn.Exec(ctx, `DELETE FROM bookings WHERE user_id = $1 AND status = 'draft'`, userID)
	return err
}

func (r *BookingRepo) GetAllActive(ctx context.Context) ([]domain.Booking, error) {
	rows, err := r.db.Conn.Query(ctx, `
        SELECT id, user_id, service_name, time_slot, date, COALESCE(user_name, ''), COALESCE(phone, ''), COALESCE(comment, ''), created_at,
               COALESCE(reminder_24h_sent, FALSE), COALESCE(reminder_1h_sent, FALSE)
        FROM bookings
        WHERE status = 'confirmed'
        ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Booking
	for rows.Next() {
		var b domain.Booking
		var id int
		var dateVal time.Time
		if err := rows.Scan(&id, &b.UserID, &b.ServiceName, &b.TimeSlot, &dateVal, &b.UserName, &b.Phone, &b.Comment, &b.CreatedAt, &b.Reminder24hSent, &b.Reminder1hSent); err != nil {
			return nil, err
		}
		b.ID = strconv.Itoa(id)
		b.Date = dateVal.Format("02.01.2006")
		result = append(result, b)
	}
	return result, nil
}

func (r *BookingRepo) DeleteBookingByID(ctx context.Context, id int) error {
	_, err := r.db.Conn.Exec(ctx, `DELETE FROM bookings WHERE id = $1`, id)
	return err
}

func (r *BookingRepo) GetByID(ctx context.Context, id int) (*domain.Booking, error) {
	var b domain.Booking
	var rowID int
	var dateVal time.Time
	err := r.db.Conn.QueryRow(ctx, `
        SELECT id, user_id, service_name, time_slot, date, COALESCE(user_name, ''), COALESCE(phone, ''), COALESCE(comment, ''), created_at,
               COALESCE(reminder_24h_sent, FALSE), COALESCE(reminder_1h_sent, FALSE)
        FROM bookings
        WHERE id = $1`, id,
	).Scan(&rowID, &b.UserID, &b.ServiceName, &b.TimeSlot, &dateVal, &b.UserName, &b.Phone, &b.Comment, &b.CreatedAt, &b.Reminder24hSent, &b.Reminder1hSent)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBookingNotFound
	} else if err != nil {
		return nil, err
	}
	b.ID = strconv.Itoa(rowID)
	b.Date = dateVal.Format("02.01.2006")
	return &b, nil
}

func (r *BookingRepo) MarkReminder24hSent(ctx context.Context, id int) error {
	_, err := r.db.Conn.Exec(ctx, `UPDATE bookings SET reminder_24h_sent = TRUE WHERE id = $1`, id)
	return err
}

func (r *BookingRepo) MarkReminder1hSent(ctx context.Context, id int) error {
	_, err := r.db.Conn.Exec(ctx, `UPDATE bookings SET reminder_1h_sent = TRUE WHERE id = $1`, id)
	return err
}

func (r *BookingRepo) ResetAll(ctx context.Context) error {
	_, err := r.db.Conn.Exec(ctx, `TRUNCATE TABLE bookings`)
	return err
}

func (r *BookingRepo) GetTakenTimeSlots(ctx context.Context, date string, serviceName string) ([]string, error) {
	// Преобразуем дату из формата YYYY-MM-DD (от WebApp) в DATE для запроса
	rows, err := r.db.Conn.Query(ctx, `
        SELECT time_slot, service_name
        FROM bookings
        WHERE date = TO_DATE($1, 'YYYY-MM-DD') AND status = 'confirmed'
        ORDER BY time_slot`,
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var timeSlot, service string
		if err := rows.Scan(&timeSlot, &service); err != nil {
			return nil, err
		}
		log.Printf("🔍 [GetTakenTimeSlots] Дата=%s, Занят слот=%s, Услуга=%s", date, timeSlot, service)
		result = append(result, timeSlot)
	}
	log.Printf("✅ [GetTakenTimeSlots] Итого занятых слотов на %s: %v", date, result)
	return result, nil
}
