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

func (r *ScheduleRepo) SaveSchedule(ctx context.Context, item domain.WorkSchedule) error {
	start, end := item.StartTime, item.EndTime
	if start == "" {
		start = "10:00"
	}
	if end == "" {
		end = "20:00"
	}
	_, err := r.db.Conn.Exec(ctx, `
		INSERT INTO work_schedule (date, is_working_day, start_time, end_time)
		VALUES ($1::date, $2, $3, $4)
		ON CONFLICT (date) DO UPDATE
		SET is_working_day = EXCLUDED.is_working_day,
		    start_time = EXCLUDED.start_time,
		    end_time = EXCLUDED.end_time`,
		item.Date, item.IsWorkingDay, start, end)
	return err
}

func (r *ScheduleRepo) ListMonth(ctx context.Context, year, month int) ([]domain.WorkSchedule, error) {
	rows, err := r.db.Conn.Query(ctx, `
		SELECT to_char(date, 'YYYY-MM-DD'), is_working_day, start_time, end_time
		FROM work_schedule
		WHERE date >= make_date($1, $2, 1)
		  AND date < (make_date($1, $2, 1) + INTERVAL '1 month')
		ORDER BY date`, year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.WorkSchedule
	for rows.Next() {
		var item domain.WorkSchedule
		if err := rows.Scan(&item.Date, &item.IsWorkingDay, &item.StartTime, &item.EndTime); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []domain.WorkSchedule{}
	}
	return items, rows.Err()
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

func (r *ScheduleRepo) SaveSlotStep(ctx context.Context, minutes int) error {
	_, err := r.db.Conn.Exec(ctx, `
		INSERT INTO admin_settings (id, slot_step_minutes, min_advance_hours)
		VALUES (1, $1, 3)
		ON CONFLICT (id) DO UPDATE SET slot_step_minutes = EXCLUDED.slot_step_minutes`, minutes)
	return err
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

func (r *ScheduleRepo) ReplaceBlocks(ctx context.Context, date string, blocks []domain.BusyInterval) error {
	tx, err := r.db.Conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM time_blocks WHERE date = $1::date`, date); err != nil {
		return err
	}
	for _, block := range blocks {
		if _, err := tx.Exec(ctx, `
			INSERT INTO time_blocks (date, start_time, end_time) VALUES ($1::date, $2, $3)`,
			date, block.Start, block.End); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *ScheduleRepo) ListBlocksMonth(ctx context.Context, year, month int) (map[string][]domain.BusyInterval, error) {
	rows, err := r.db.Conn.Query(ctx, `
		SELECT to_char(date, 'YYYY-MM-DD'), start_time, end_time
		FROM time_blocks
		WHERE date >= make_date($1, $2, 1)
		  AND date < (make_date($1, $2, 1) + INTERVAL '1 month')
		ORDER BY date, start_time`, year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string][]domain.BusyInterval{}
	for rows.Next() {
		var date string
		var item domain.BusyInterval
		if err := rows.Scan(&date, &item.Start, &item.End); err != nil {
			return nil, err
		}
		out[date] = append(out[date], item)
	}
	return out, rows.Err()
}
