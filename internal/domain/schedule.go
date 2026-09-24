package domain

import (
	"context"
	"time"
)

type Service struct {
	Name     string
	Duration int
}

type WorkSchedule struct {
	Date         string
	IsWorkingDay bool
	StartTime    string
	EndTime      string
}

type AdminSettings struct {
	SlotStepMinutes int
	MinAdvanceHours int
}

type BusyInterval struct {
	Start string
	End   string
}

type ScheduleStore interface {
	SetDayOff(ctx context.Context, date string) error
	SetWorkDay(ctx context.Context, date, start, end string) error
	SaveSchedule(ctx context.Context, item WorkSchedule) error
	ListMonth(ctx context.Context, year, month int) ([]WorkSchedule, error)
	BlockTime(ctx context.Context, date, start, end string) error
	GetSchedule(ctx context.Context, date string) (*WorkSchedule, error)
	GetSettings(ctx context.Context) (AdminSettings, error)
	ListBusy(ctx context.Context, date string) ([]BusyInterval, error)
}

var serviceDurations = map[string]int{
	"Organic brow 🍈 + оформление бровей + ламинирование ресниц": 120,
	"Ламинирование ресниц + ламинирование бровей":               120,
	"Ламинирование ресниц + натуральное оформление бровей 🪞":    120,
	"Ламинирование ресниц + снятие наращенных ресниц":           80,
	"Ламинирование ресниц 🐚":                                    60,
	"Коррекция бровей":                                          60,
	"Ламинирование бровей":                                      60,
	"Натуральное оформление бровей 🐚":                           60,
	"Осветление бровей":                                         80,
	"Полный комплекс ламинирования бровей":                      60,
	"Organic brow 🍈 + коррекция":                                60,
	"Organic brow 🍈 + натуральное оформление бровей":            90,
	"Удаление пушка на губой":                                   10,
}

func ServiceDuration(name string) int {
	if d, ok := serviceDurations[name]; ok && d > 0 {
		return d
	}
	return 60
}

func AvailableSlots(day time.Time, schedule *WorkSchedule, settings AdminSettings, serviceDuration int, busy []BusyInterval, now time.Time) []string {
	if schedule != nil && !schedule.IsWorkingDay {
		return nil
	}
	startText, endText := "10:00", "20:00"
	if schedule != nil && schedule.StartTime != "" && schedule.EndTime != "" {
		startText, endText = schedule.StartTime, schedule.EndTime
	}
	step := settings.SlotStepMinutes
	if step <= 0 {
		step = 30
	}
	advance := settings.MinAdvanceHours
	if advance < 0 {
		advance = 3
	}
	if serviceDuration <= 0 {
		serviceDuration = 60
	}

	startMin, okStart := ClockMinutes(startText)
	endMin, okEnd := ClockMinutes(endText)
	if !okStart || !okEnd || endMin <= startMin {
		return nil
	}

	var occupied [][2]int
	for _, item := range busy {
		a, okA := ClockMinutes(item.Start)
		b, okB := ClockMinutes(item.End)
		if okA && okB && b > a {
			occupied = append(occupied, [2]int{a, b})
		}
	}

	sameDay := day.Year() == now.Year() && day.Month() == now.Month() && day.Day() == now.Day()
	nowMin := now.Hour()*60 + now.Minute()

	var slots []string
	for cursor := startMin; cursor+serviceDuration <= endMin; cursor += step {
		if sameDay && cursor-nowMin < advance*60 {
			continue
		}
		slotEnd := cursor + serviceDuration
		overlaps := false
		for _, span := range occupied {
			if cursor < span[1] && span[0] < slotEnd {
				overlaps = true
				break
			}
		}
		if overlaps {
			continue
		}
		slots = append(slots, FormatClock(cursor))
	}
	return slots
}

func ClockMinutes(value string) (int, bool) {
	t, err := time.Parse("15:04", value)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

func FormatClock(minutes int) string {
	return time.Date(0, 1, 1, minutes/60, minutes%60, 0, 0, time.UTC).Format("15:04")
}
