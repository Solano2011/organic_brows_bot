package app

import (
	"context"
	"log"
	"strconv"
	"time"

	"hookah-bot/internal/delivery/telegram"
	"hookah-bot/internal/domain"

	tele "gopkg.in/telebot.v3"
)

func StartReminderScheduler(b *tele.Bot, svc domain.BookingService) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	sendDueReminders(b, svc)
	for range ticker.C {
		sendDueReminders(b, svc)
	}
}

func sendDueReminders(b *tele.Bot, svc domain.BookingService) {
	ctx := context.Background()
	bookings, err := svc.GetAllActiveBookings(ctx)
	if err != nil {
		log.Printf("❌ Ошибка выборки записей для напоминаний: %v", err)
		return
	}

	loc, err := time.LoadLocation("Europe/Samara")
	if err != nil {
		log.Printf("⚠️ Не удалось загрузить Europe/Samara, используем UTC: %v", err)
		loc = time.UTC
	}
	now := time.Now().In(loc)
	for _, booking := range bookings {
		start, err := parseBookingTime(booking.Date, booking.TimeSlot, loc)
		if err != nil {
			log.Printf("⚠️ Не удалось разобрать дату записи id=%s: %v", booking.ID, err)
			continue
		}
		until := start.Sub(now)
		id, err := strconv.Atoi(booking.ID)
		if err != nil {
			continue
		}

		switch {
		case !booking.Reminder24hSent && until <= 24*time.Hour && until > 23*time.Hour:
			text := telegram.FormatReminder(
				"Добрый день, у вас назначена запись к специалисту:",
				booking.ServiceName, booking.Date, booking.TimeSlot,
			)
			if _, err := b.Send(tele.ChatID(booking.UserID), text, telegram.BuildReminderMenu(booking.ID), tele.ModeHTML); err != nil {
				log.Printf("❌ Не удалось отправить напоминание за 24ч, id=%d: %v", id, err)
				continue
			}
			if err := svc.MarkReminder24hSent(ctx, id); err != nil {
				log.Printf("❌ Не удалось отметить reminder_24h_sent, id=%d: %v", id, err)
			}
		case !booking.Reminder1hSent && until <= time.Hour && until > 0:
			text := telegram.FormatReminder(
				"Напоминаем что вы записаны к специалисту:",
				booking.ServiceName, booking.Date, booking.TimeSlot,
			)
			if _, err := b.Send(tele.ChatID(booking.UserID), text, telegram.BuildReminderMenu(booking.ID), tele.ModeHTML); err != nil {
				log.Printf("❌ Не удалось отправить напоминание за 1ч, id=%d: %v", id, err)
				continue
			}
			if err := svc.MarkReminder1hSent(ctx, id); err != nil {
				log.Printf("❌ Не удалось отметить reminder_1h_sent, id=%d: %v", id, err)
			}
		}
	}
}

func availableSlotsFor(ctx context.Context, store domain.ScheduleStore, date, serviceName string) ([]string, error) {
	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}
	loc, locErr := time.LoadLocation("Europe/Samara")
	if locErr != nil {
		loc = time.UTC
	}
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	schedule, err := store.GetSchedule(ctx, date)
	if err != nil {
		return nil, err
	}
	settings, err := store.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	busy, err := store.ListBusy(ctx, date)
	if err != nil {
		return nil, err
	}
	// ListBusy уже объединяет подтверждённые записи и перерывы из time_blocks.
	return domain.AvailableSlots(day, schedule, settings, domain.ServiceDuration(serviceName), busy, time.Now().In(loc)), nil
}

func slotIsOpen(slots []string, selected string) bool {
	for _, slot := range slots {
		if slot == selected {
			return true
		}
	}
	return false
}

func parseBookingTime(date, slot string, loc *time.Location) (time.Time, error) {
	return time.ParseInLocation("02.01.2006 15:04", date+" "+slot, loc)
}
