package memory_test

import (
	"context"
	"testing"

	"hookah-bot/internal/repository/memory"
)

func TestMemoryBookingRepo_CompleteBooking(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewBookingRepo()

	userID := int64(123)
	serviceName := "Наращивание ногтей"
	date := "2024-01-15"
	timeSlot := "14:00"
	name := "Мария"
	phone := "+79991112233"
	comment := "Хочу френч"

	// 1. Создаем черновик
	if err := repo.SaveDraft(ctx, userID, serviceName); err != nil {
		t.Fatalf("ошибка сохранения драфта: %v", err)
	}

	// 2. Устанавливаем дату
	if err := repo.SetDraftDate(ctx, userID, date); err != nil {
		t.Fatalf("ошибка установки даты: %v", err)
	}

	// 3. Завершаем бронь с новыми параметрами
	booking, err := repo.CompleteBooking(ctx, userID, timeSlot, name, phone, comment)
	if err != nil {
		t.Fatalf("ошибка завершения брони: %v", err)
	}

	if booking.TimeSlot != timeSlot || booking.UserName != name || booking.Phone != phone || booking.Comment != comment {
		t.Errorf("данные брони не совпали: %+v", booking)
	}
	
	if booking.ServiceName != serviceName {
		t.Errorf("услуга не совпала: ожидалось %s, получено %s", serviceName, booking.ServiceName)
	}
	
	if booking.Date != date {
		t.Errorf("дата не совпала: ожидалось %s, получено %s", date, booking.Date)
	}
}
