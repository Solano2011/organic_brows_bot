package postgres

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
)

type DB struct {
	Conn *pgx.Conn
}

func NewPostgresDB(connString string) (*DB, error) {
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к PostgreSQL: %w", err)
	}

	if err = conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ошибка пинга PostgreSQL: %w", err)
	}

	db := &DB{Conn: conn}

	// Запускаем миграции
	if err := runMigrations(connString); err != nil {
		return nil, fmt.Errorf("ошибка миграций: %w", err)
	}

	if _, err := conn.Exec(context.Background(), `
		ALTER TABLE bookings ADD COLUMN IF NOT EXISTS reminder_24h_sent BOOLEAN DEFAULT FALSE;
		ALTER TABLE bookings ADD COLUMN IF NOT EXISTS reminder_1h_sent BOOLEAN DEFAULT FALSE;
		CREATE TABLE IF NOT EXISTS work_schedule (
			date DATE PRIMARY KEY,
			is_working_day BOOLEAN NOT NULL DEFAULT TRUE,
			start_time VARCHAR(5) NOT NULL DEFAULT '10:00',
			end_time VARCHAR(5) NOT NULL DEFAULT '20:00'
		);
		CREATE TABLE IF NOT EXISTS admin_settings (
			id INT PRIMARY KEY,
			slot_step_minutes INT NOT NULL DEFAULT 30,
			min_advance_hours INT NOT NULL DEFAULT 3
		);
		INSERT INTO admin_settings (id, slot_step_minutes, min_advance_hours)
		VALUES (1, 30, 3)
		ON CONFLICT (id) DO NOTHING;
		CREATE TABLE IF NOT EXISTS time_blocks (
			id SERIAL PRIMARY KEY,
			date DATE NOT NULL,
			start_time VARCHAR(5) NOT NULL,
			end_time VARCHAR(5) NOT NULL
		);
	`); err != nil {
		return nil, fmt.Errorf("ошибка миграции напоминаний: %w", err)
	}

	log.Println("✅ Подключение к БД успешно, миграции применены!")
	return db, nil
}

func runMigrations(connString string) error {
	// Драйвер миграций ожидает префикс postgres:// или postgresql://
	// Но pgx понимает оба, на всякий случай убедимся, что формат верный
	if !strings.HasPrefix(connString, "postgres://") && !strings.HasPrefix(connString, "postgresql://") {
		return fmt.Errorf("строка подключения должна начинаться с postgres:// или postgresql://")
	}

	// Указываем путь к папке с миграциями (file://migrations)
	m, err := migrate.New("file://migrations", connString)
	if err != nil {
		return fmt.Errorf("ошибка инициализации migrate: %w", err)
	}

	// Пытаемся накатить миграции
	err = m.Up()
	if err != nil {
		if err == migrate.ErrNoChange {
			log.Println("ℹ️ Структура БД актуальна, новых миграций нет.")
			return nil
		}
		return err
	}

	log.Println("🔥 Миграции успешно применены к базе данных!")
	return nil
}
