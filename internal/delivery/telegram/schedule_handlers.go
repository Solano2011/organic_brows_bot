package telegram

import (
	"context"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (h *Handlers) handleDayOff(c tele.Context) error {
	if !h.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет прав администратора.")
	}
	parts := strings.Fields(c.Message().Payload)
	if len(parts) != 1 {
		return c.Send("Формат: /dayoff ДД.ММ.ГГГГ")
	}
	date, err := commandDate(parts[0])
	if err != nil {
		return c.Send("Дата должна быть в формате ДД.ММ.ГГГГ")
	}
	if err := h.schedule.SetDayOff(context.Background(), date); err != nil {
		return c.Send("Не удалось сохранить выходной.")
	}
	return c.Send("День " + parts[0] + " отмечен как выходной.")
}

func (h *Handlers) handleWorkDay(c tele.Context) error {
	if !h.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет прав администратора.")
	}
	parts := strings.Fields(c.Message().Payload)
	if len(parts) != 3 {
		return c.Send("Формат: /workday ДД.ММ.ГГГГ 10:00 20:00")
	}
	date, err := commandDate(parts[0])
	if err != nil || !validClock(parts[1]) || !validClock(parts[2]) {
		return c.Send("Проверьте дату и время. Формат: /workday ДД.ММ.ГГГГ 10:00 20:00")
	}
	if err := h.schedule.SetWorkDay(context.Background(), date, parts[1], parts[2]); err != nil {
		return c.Send("Не удалось сохранить рабочие часы.")
	}
	return c.Send("Рабочий день " + parts[0] + ": " + parts[1] + "–" + parts[2])
}

func (h *Handlers) handleBlockTime(c tele.Context) error {
	if !h.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет прав администратора.")
	}
	parts := strings.Fields(c.Message().Payload)
	if len(parts) != 3 {
		return c.Send("Формат: /block ЧЧ:ММ ЧЧ:ММ ДД.ММ.ГГГГ")
	}
	date, err := commandDate(parts[2])
	if err != nil || !validClock(parts[0]) || !validClock(parts[1]) {
		return c.Send("Проверьте время и дату. Формат: /block ЧЧ:ММ ЧЧ:ММ ДД.ММ.ГГГГ")
	}
	if err := h.schedule.BlockTime(context.Background(), date, parts[0], parts[1]); err != nil {
		return c.Send("Не удалось заблокировать время.")
	}
	return c.Send("Интервал " + parts[0] + "–" + parts[1] + " на " + parts[2] + " закрыт.")
}

func (h *Handlers) scheduleURL() string {
	return strings.TrimRight(h.webAppBaseURL, "/") + "/admin/schedule"
}

func commandDate(value string) (string, error) {
	parsed, err := time.Parse("02.01.2006", value)
	if err != nil {
		return "", err
	}
	return parsed.Format("2006-01-02"), nil
}

func validClock(value string) bool {
	_, err := time.Parse("15:04", value)
	return err == nil
}
