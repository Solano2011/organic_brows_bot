package app

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"hookah-bot/internal/delivery/telegram"
	"hookah-bot/internal/domain"

	tele "gopkg.in/telebot.v3"
)

func StartReminderScheduler(b *tele.Bot, svc domain.BookingService) {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		sendDueReminders(b, svc)
		for range ticker.C {
			sendDueReminders(b, svc)
		}
	}()
}

func sendDueReminders(b *tele.Bot, svc domain.BookingService) {
	ctx := context.Background()
	bookings, err := svc.GetAllActiveBookings(ctx)
	if err != nil {
		log.Printf("❌ Ошибка выборки записей для напоминаний: %v", err)
		return
	}

	now := time.Now()
	for _, booking := range bookings {
		start, err := parseBookingTime(booking.Date, booking.TimeSlot, now.Location())
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

func parseBookingTime(date, slot string, loc *time.Location) (time.Time, error) {
	day, err := time.ParseInLocation("02.01.2006", date, loc)
	if err != nil {
		return time.Time{}, err
	}
	clockPart := strings.TrimSpace(slot)
	if i := strings.Index(clockPart, "-"); i > 0 {
		clockPart = strings.TrimSpace(clockPart[:i])
	}
	clock, err := time.ParseInLocation("15:04", clockPart, loc)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, loc), nil
}
