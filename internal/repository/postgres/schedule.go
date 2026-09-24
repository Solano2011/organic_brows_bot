package postgres

import (
	"context"
	"errors"

	"hookah-bot/internal/domain"

	"github.com/jackc/pgx/v5"
)

type ScheduleRepo struct {
	db *DB
}

func NewScheduleRepo(db *DB) *ScheduleRepo {
	return &ScheduleRepo{db: db}
}

func (r *ScheduleRepo) SetDayOff(ctx context.Context, date string) error {
	_, err := r.db.Conn.Exec(ctx, `
		INSERT INTO work_schedule (date, is_working_day, start_time, end_time)
		VALUES ($1::date, FALSE, '10:00', '20:00')
		ON CONFLICT (date) DO UPDATE SET is_working_day = FALSE`, date)
	return err
}

func (r *ScheduleRepo) SetWorkDay(ctx context.Context, date, start, end string) error {
	_, err := r.db.Conn.Exec(ctx, `
		INSERT INTO work_schedule (date, is_working_day, start_time, end_time)
		VALUES ($1::date, TRUE, $2, $3)
		ON CONFLICT (date) DO UPDATE SET is_working_day = TRUE, start_time = $2, end_time = $3`,
		date, start, end)
	return err
}

func (r *ScheduleRepo) BlockTime(ctx context.Context, date, start, end string) error {
	_, err := r.db.Conn.Exec(ctx, `
		INSERT INTO time_blocks (date, start_time, end_time) VALUES ($1::date, $2, $3)`,
		date, start, end)
	return err
}

func (r *ScheduleRepo) GetSchedule(ctx context.Context, date string) (*domain.WorkSchedule, error) {
	var item domain.WorkSchedule
	err := r.db.Conn.QueryRow(ctx, `
		SELECT to_char(date, 'YYYY-MM-DD'), is_working_day, start_time, end_time
		FROM work_schedule WHERE date = $1::date`, date).Scan(&item.Date, &item.IsWorkingDay, &item.StartTime, &item.EndTime)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ScheduleRepo) GetSettings(ctx context.Context) (domain.AdminSettings, error) {
	var settings domain.AdminSettings
	err := r.db.Conn.QueryRow(ctx, `
		SELECT slot_step_minutes, min_advance_hours FROM admin_settings WHERE id = 1`,
	).Scan(&settings.SlotStepMinutes, &settings.MinAdvanceHours)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AdminSettings{SlotStepMinutes: 30, MinAdvanceHours: 3}, nil
	}
	return settings, err
}

func (r *ScheduleRepo) ListBusy(ctx context.Context, date string) ([]domain.BusyInterval, error) {
	rows, err := r.db.Conn.Query(ctx, `
		SELECT time_slot, service_name FROM bookings
		WHERE date = $1::date AND status = 'confirmed'`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var busy []domain.BusyInterval
	for rows.Next() {
		var start, service string
		if err := rows.Scan(&start, &service); err != nil {
			return nil, err
		}
		startMin, ok := domain.ClockMinutes(start)
		if !ok {
			continue
		}
		busy = append(busy, domain.BusyInterval{
			Start: start,
			End:   domain.FormatClock(startMin + domain.ServiceDuration(service)),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	blocks, err := r.db.Conn.Query(ctx, `
		SELECT start_time, end_time FROM time_blocks WHERE date = $1::date`, date)
	if err != nil {
		return nil, err
	}
	defer blocks.Close()
	for blocks.Next() {
		var item domain.BusyInterval
		if err := blocks.Scan(&item.Start, &item.End); err != nil {
			return nil, err
		}
		busy = append(busy, item)
	}
	return busy, blocks.Err()
}
