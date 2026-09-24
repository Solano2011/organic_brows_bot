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

CREATE INDEX IF NOT EXISTS idx_time_blocks_date ON time_blocks(date);
