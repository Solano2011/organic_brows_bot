package domain

import (
	"testing"
	"time"
)

func TestAvailableSlotsOverlapAndAdvance(t *testing.T) {
	day := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	settings := AdminSettings{SlotStepMinutes: 30, MinAdvanceHours: 3}
	busy := []BusyInterval{{Start: "12:00", End: "14:00"}}

	slots := AvailableSlots(day, nil, settings, 60, busy, now)
	for _, slot := range slots {
		if slot == "11:30" || slot == "12:00" || slot == "12:30" || slot == "13:00" || slot == "13:30" {
			t.Fatalf("слот %s не должен быть доступен", slot)
		}
	}
	found := false
	for _, slot := range slots {
		if slot == "14:00" {
			found = true
		}
	}
	if !found {
		t.Fatal("слот 14:00 должен остаться свободным")
	}
}

func TestLongServiceDoesNotPassClosingTime(t *testing.T) {
	day := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	settings := AdminSettings{SlotStepMinutes: 30, MinAdvanceHours: 3}
	slots := AvailableSlots(day, nil, settings, 120, nil, now)

	has1630 := false
	for _, slot := range slots {
		if slot == "16:30" {
			has1630 = true
		}
		if slot == "19:00" || slot == "18:30" {
			t.Fatalf("слот %s не помещается в день до 20:00 при услуге 120 минут", slot)
		}
	}
	if !has1630 {
		t.Fatal("16:30 должно быть доступно: услуга заканчивается в 18:30")
	}
}

func TestDayOffHasNoSlots(t *testing.T) {
	day := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	schedule := &WorkSchedule{IsWorkingDay: false}
	slots := AvailableSlots(day, schedule, AdminSettings{SlotStepMinutes: 30, MinAdvanceHours: 3}, 60, nil, now)
	if len(slots) != 0 {
		t.Fatalf("в выходной слотов быть не должно, получено %v", slots)
	}
}
