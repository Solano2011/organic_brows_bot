package telegram

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"strconv"
	"strings"

	"hookah-bot/internal/domain"

	tele "gopkg.in/telebot.v3"
)

const (
	ImgHeroUrl = "https://images.unsplash.com/photo-1604654894610-df63bc536371?w=900&auto=format&fit=crop&q=80"
)

type Handlers struct {
	bookingService domain.BookingService
	schedule       domain.ScheduleStore
	adminID        int64
	bot            *tele.Bot
	webAppBaseURL  string
}

func NewHandlers(bs domain.BookingService, schedule domain.ScheduleStore, adminID int64, bot *tele.Bot, webAppURL string) *Handlers {
	return &Handlers{
		bookingService: bs,
		schedule:       schedule,
		adminID:        adminID,
		bot:            bot,
		webAppBaseURL:  webAppURL,
	}
}

func (h *Handlers) InitRoutes(b *tele.Bot) {
	b.Handle("/start", h.handleStart)
	b.Handle("/admin", h.handleAdmin)
	b.Handle("/dayoff", h.handleDayOff)
	b.Handle("/workday", h.handleWorkDay)
	b.Handle("/block", h.handleBlockTime)

	b.Handle(&BtnBackToMain, h.handleBackToMain)
	b.Handle(&BtnMyBooking, h.handleMyBookingBtn)
	b.Handle(&BtnCancelBooking, h.handleCancelBooking)
	b.Handle(&BtnContacts, h.handleContactsBtn)

	b.Handle(&BtnService, h.handleServiceSelect)

	// Подтверждение замены записи
	b.Handle(&BtnConfirmReplace, h.handleConfirmReplace)
	b.Handle(&BtnKeepOldBooking, h.handleKeepOldBooking)

	// Обработчик данных из Web App
	b.Handle(tele.OnWebApp, h.handleWebApp)

	// Обработчик кнопки "Моя бронь"
	b.Handle(&BtnMyBookings, h.handleMyBookings)
	b.Handle(&BtnHowToGet, h.handleHowToGet)

	b.Handle(&BtnAdminRefresh, h.handleAdminRefresh)
	b.Handle(&BtnAdminResetAll, h.handleAdminResetAll)
	b.Handle(tele.OnCallback, h.handleCallback)
}

func (h *Handlers) isAdmin(userID int64) bool {
	return h.adminID != 0 && userID == h.adminID
}

func (h *Handlers) handleStart(c tele.Context) error {
	// Теперь это просто переменная с текстом (назовем ее text вместо caption)
	text := fmt.Sprintf(
		"Добро пожаловать, *%s*! 🌿\n\n"+
			"Я помогу вам записаться к мастеру.\n"+
			"Выберите действие из меню ниже:",
		c.Sender().FirstName,
	)

	// Отправляем текст напрямую, без привязки к фото
	return c.Send(text, BuildInlineMainMenu(h.webAppBaseURL), tele.ModeMarkdown)
}

func (h *Handlers) handleBackToMain(c tele.Context) error {
	_ = c.Delete()
	caption := "Выберите услугу для записи:"
	photo := &tele.Photo{File: tele.FromURL(ImgHeroUrl), Caption: caption}
	return c.Send(photo, BuildServicesMenu(), tele.ModeMarkdown)
}

func (h *Handlers) handleServiceSelect(c tele.Context) error {
	serviceName := c.Data()
	ctx := context.Background()

	// Проверяем, есть ли уже активная запись
	existingBooking, err := h.bookingService.GetUserBooking(ctx, c.Sender().ID)
	if err == nil && existingBooking.TimeSlot != "" {
		// У пользователя уже есть активная запись
		_ = c.Delete()
		text := fmt.Sprintf(
			"⚠️ *У вас уже есть активная запись:*\n\n"+
				"Услуга: `%s`\n"+
				"Дата: `%s`\n"+
				"Время: `%s`\n"+
				"👤 Имя: `%s`\n"+
				"📞 Телефон: `%s`\n\n"+
				"Хотите отменить предыдущую запись и создать новую?",
			existingBooking.ServiceName, existingBooking.Date, existingBooking.TimeSlot,
			existingBooking.UserName, existingBooking.Phone,
		)

		// Сохраняем выбранную услугу в черновик для последующего использования
		_ = h.bookingService.StartBookingDraft(ctx, c.Sender().ID, serviceName)

		return c.Send(text, BuildReplaceConfirmMenu(), tele.ModeMarkdown)
	}

	// Сохраняем черновик с выбранной услугой
	if err := h.bookingService.StartBookingDraft(ctx, c.Sender().ID, serviceName); err != nil {
		return c.Send("❌ Ошибка сохранения. Попробуйте еще раз.")
	}

	_ = c.Delete()

	// Открываем WebApp через REPLY-клавиатуру (не Inline!)
	// Это критично для работы tg.sendData()
	webAppURL := fmt.Sprintf("%s?service=%s", h.webAppBaseURL, strings.ReplaceAll(serviceName, " ", "+"))

	text := fmt.Sprintf("💅 *Выбрана услуга:*\n`%s`\n\nНажмите кнопку ниже, чтобы выбрать дату и время:", serviceName)
	return c.Send(text, BuildWebAppReplyKeyboard(webAppURL), tele.ModeMarkdown)
}

// Принимаем данные из Web App
func (h *Handlers) handleWebApp(c tele.Context) error {
	if c.Message().WebAppData == nil {
		return nil
	}

	rawData := c.Message().WebAppData.Data

	// Ожидаем строку вида "date|timeSlot|name|phone|comment"
	parts := strings.Split(rawData, "|")
	if len(parts) != 5 {
		log.Printf("❌ [Web App] Неверный формат данных: %s", rawData)
		return c.Send("❌ Ошибка: Неверный формат данных от Web App.")
	}

	date := parts[0]
	timeSlot := parts[1]
	name := parts[2]
	phone := parts[3]
	comment := parts[4]

	ctx := context.Background()
	userID := c.Sender().ID

	log.Printf("🌐 [Web App] Получены данные от userID=%d: дата=%s, время=%s, имя=%s, телефон=%s, комментарий=%s",
		userID, date, timeSlot, name, phone, comment)

	// Проверяем и удаляем старую подтверждённую запись перед созданием новой
	existingBooking, err := h.bookingService.GetUserBooking(ctx, userID)
	if err == nil && existingBooking.TimeSlot != "" {
		log.Printf("⚠️ [Web App] У userID=%d найдена существующая запись: услуга=%s, дата=%s, время=%s",
			userID, existingBooking.ServiceName, existingBooking.Date, existingBooking.TimeSlot)

		if err := h.bookingService.CancelConfirmedBooking(ctx, userID); err != nil {
			log.Printf("❌ [Web App] Ошибка удаления старой записи для userID=%d: %v", userID, err)
			return c.Send("❌ Ошибка при удалении старой записи. Попробуйте позже.")
		}
		log.Printf("✅ [Web App] Старая запись userID=%d успешно удалена", userID)
	}

	// Сохраняем дату в черновик
	if err := h.bookingService.SetBookingDate(ctx, userID, date); err != nil {
		log.Printf("❌ [Web App] Ошибка сохранения даты для userID=%d: %v", userID, err)
		return c.Send("❌ Ошибка сохранения даты.")
	}

	// Финализируем запись
	booking, err := h.bookingService.CompleteBookingDraft(ctx, userID, timeSlot, name, phone, comment)
	if err != nil {
		if errors.Is(err, domain.ErrTimeSlotTaken) {
			log.Printf("⚠️ [Web App] Слот занят для userID=%d: %s на %s", userID, timeSlot, date)
			return c.Send("❌ Это время уже занято! Начните запись заново.")
		}
		log.Printf("❌ [Web App] Ошибка завершения записи для userID=%d: %v", userID, err)
		return c.Send("❌ Сессия истекла или произошла ошибка. Начните заново.")
	}

	log.Printf("✅ [Web App] Запись успешно создана для userID=%d: услуга=%s, дата=%s, время=%s",
		userID, booking.ServiceName, booking.Date, booking.TimeSlot)

	// Удаляем сообщение с кнопкой Web App
	_ = h.bot.Delete(c.Message())

	// Уведомляем админа
	if h.adminID != 0 && h.bot != nil {
		user := c.Sender()
		usernameStr := "@" + user.Username
		if user.Username == "" {
			usernameStr = "без username"
		}
		notifyText := fmt.Sprintf(
			"🔔 <b>НОВАЯ ЗАПИСЬ В СИСТЕМЕ</b>\n"+
				"━━━━━━━━━━━━━━━\n"+
				"<b>Имя:</b> %s\n"+
				"<b>Телефон:</b> <a href='tel:%s'>%s</a>\n"+
				"Гость: %s\n"+
				"<b>Услуга:</b> %s\n"+
				"<b>Дата:</b> %s\n"+
				"<b>Время:</b> %s\n"+
				"<b>Комментарий:</b> %s",
			html.EscapeString(booking.UserName), booking.Phone, booking.Phone, html.EscapeString(usernameStr),
			html.EscapeString(booking.ServiceName), html.EscapeString(booking.Date), html.EscapeString(booking.TimeSlot), html.EscapeString(booking.Comment),
		)
		go func(msg string) { _, _ = h.bot.Send(tele.ChatID(h.adminID), msg, tele.ModeHTML) }(notifyText)
	}

	// Подтверждаем пользователю
	text := FormatClientBooking(booking.ServiceName, booking.Date, booking.TimeSlot)

	return c.Send(text, BuildBookingSuccessMenu(h.webAppBaseURL), tele.ModeHTML)
}

func (h *Handlers) handleHowToGet(c tele.Context) error {
	if err := c.Respond(); err != nil {
		return err
	}
	photo := &tele.Photo{
		File:    tele.FromDisk("static/img/door.jpg"),
		Caption: "Фото входа \n\nДалее нужно подняться на второй этаж, повернуть налево и идти до 10-го кабинета",
	}
	return c.Send(photo)
}

// handleMyBookings обрабатывает нажатие на кнопку "📅 Моя бронь"
func (h *Handlers) handleMyBookings(c tele.Context) error {
	// Обязательно отвечаем на callback, чтобы убрать "часики" в Telegram
	if err := c.Respond(); err != nil {
		return err
	}

	ctx := context.Background()
	b, err := h.bookingService.GetUserBooking(ctx, c.Sender().ID)

	if err != nil || b.TimeSlot == "" {
		return c.Send(
			"У вас пока нет активных записей.",
			BuildInlineMainMenu(h.webAppBaseURL),
		)
	}

	text := FormatClientBooking(b.ServiceName, b.Date, b.TimeSlot)

	m := &tele.ReplyMarkup{}
	m.Inline(m.Row(BtnCancelBooking))

	return c.Send(text, m, tele.ModeHTML)
}

func (h *Handlers) handleMyBookingBtn(c tele.Context) error {
	ctx := context.Background()
	b, err := h.bookingService.GetUserBooking(ctx, c.Sender().ID)

	_ = c.Delete()
	if err != nil || b.TimeSlot == "" {
		m := &tele.ReplyMarkup{}
		m.Inline(m.Row(BtnBackToMain))
		return c.Send("У вас пока нет активных записей.", m)
	}

	m := &tele.ReplyMarkup{}
	m.Inline(m.Row(BtnCancelBooking), m.Row(BtnBackToMain))

	text := FormatClientBooking(b.ServiceName, b.Date, b.TimeSlot)
	return c.Send(text, m, tele.ModeHTML)
}

func (h *Handlers) handleCancelBooking(c tele.Context) error {
	ctx := context.Background()
	existing, err := h.bookingService.GetUserBooking(ctx, c.Sender().ID)
	if err != nil || existing == nil || existing.TimeSlot == "" {
		_ = c.Respond()
		_ = c.Delete()
		return c.Send("У вас пока нет активных записей.", BuildInlineMainMenu(h.webAppBaseURL))
	}

	id, err := strconv.Atoi(existing.ID)
	if err != nil {
		_ = c.Respond(&tele.CallbackResponse{Text: "Не удалось найти запись", ShowAlert: true})
		return nil
	}
	booking, err := h.bookingService.GetBookingByID(ctx, id)
	if err != nil {
		_ = c.Respond(&tele.CallbackResponse{Text: "Не удалось найти запись", ShowAlert: true})
		return nil
	}

	if err := h.bookingService.CancelBooking(ctx, c.Sender().ID); err != nil {
		log.Printf("❌ Ошибка отмены записи userID=%d: %v", c.Sender().ID, err)
		return c.Respond(&tele.CallbackResponse{Text: "Не удалось отменить запись", ShowAlert: true})
	}

	if h.adminID != 0 && h.bot != nil {
		adminText := fmt.Sprintf(
			"⚠️ <b>Внимание! Клиент отменил запись.</b>\nУслуга: %s\nДата: %s\nВремя: %s\nИмя: %s\nТелефон: <a href='tel:%s'>%s</a>",
			html.EscapeString(booking.ServiceName),
			html.EscapeString(booking.Date),
			html.EscapeString(booking.TimeSlot),
			html.EscapeString(booking.UserName),
			booking.Phone, booking.Phone,
		)
		_, _ = h.bot.Send(tele.ChatID(h.adminID), adminText, tele.ModeHTML)
	}

	_ = c.Delete()
	_ = c.Respond()

	return c.Send("✅ Запись успешно отменена.", BuildInlineMainMenu(h.webAppBaseURL))
}

func (h *Handlers) handleContactsBtn(c tele.Context) error {
	_ = c.Delete()
	text := "📍 *Контакты*\n\n📍 *Адрес:* ул. Примерная, д. 1\n📞 *Телефон:* `+7 (999) 123-45-67`"
	return c.Send(text, BuildContactsMenu(), tele.ModeMarkdown)
}

func (h *Handlers) renderAdminDashboard(ctx context.Context) (string, []domain.Booking, error) {
	bookings, err := h.bookingService.GetAllActiveBookings(ctx)
	if err != nil {
		return "", nil, err
	}
	if len(bookings) == 0 {
		return "🛠 <b>Панель администратора</b>\n\nНа сегодня активных записей нет.", bookings, nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "🛠 <b>Панель администратора</b>\nВсего активных записей: <b>%d</b>\n━━━━━━━━━━━━━━━\n", len(bookings))
	for i, b := range bookings {
		fmt.Fprintf(&sb, "<b>%d.</b> %s | <b>%s</b> | <b>%s</b>\n   👤 %s | 📞 %s\n   💬 %s\n   🆔 Гость: <code>%d</code>\n\n",
			i+1,
			html.EscapeString(b.ServiceName),
			html.EscapeString(b.Date),
			html.EscapeString(b.TimeSlot),
			html.EscapeString(b.UserName),
			html.EscapeString(b.Phone),
			html.EscapeString(b.Comment),
			b.UserID)
	}
	return sb.String(), bookings, nil
}

func (h *Handlers) handleAdmin(c tele.Context) error {
	if !h.isAdmin(c.Sender().ID) {
		return c.Send("❌ У вас нет прав администратора.")
	}
	text, bookings, _ := h.renderAdminDashboard(context.Background())
	return c.Send(text, BuildAdminMenu(bookings), tele.ModeHTML)
}

func (h *Handlers) handleAdminRefresh(c tele.Context) error {
	if !h.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "Доступ запрещен", ShowAlert: true})
	}
	text, bookings, _ := h.renderAdminDashboard(context.Background())
	_ = c.Delete()
	return c.Send(text, BuildAdminMenu(bookings), tele.ModeHTML)
}

func (h *Handlers) handleAdminResetAll(c tele.Context) error {
	if !h.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "Доступ запрещен", ShowAlert: true})
	}
	ctx := context.Background()
	_ = h.bookingService.ResetAllBookings(ctx)
	_ = c.Delete()
	return c.Send("✅ *Все записи успешно отменены.*", BuildAdminMenu(nil), tele.ModeMarkdown)
}

func (h *Handlers) handleCallback(c tele.Context) error {
	data := c.Data()
	switch {
	case strings.HasPrefix(data, "del_book:"):
		return h.handleDeleteBooking(c)
	case strings.HasPrefix(data, "confirm_remind_"):
		return h.handleConfirmReminder(c)
	case strings.HasPrefix(data, "cancel_remind_"):
		return h.handleCancelReminder(c)
	}
	return nil
}

func (h *Handlers) handleDeleteBooking(c tele.Context) error {
	if !h.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "Доступ запрещен", ShowAlert: true})
	}

	idStr := strings.TrimPrefix(c.Data(), "del_book:")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Некорректный ID", ShowAlert: true})
	}

	ctx := context.Background()
	if err := h.bookingService.DeleteBookingByID(ctx, id); err != nil {
		log.Printf("❌ Ошибка удаления записи id=%d: %v", id, err)
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка удаления", ShowAlert: true})
	}

	text, bookings, _ := h.renderAdminDashboard(ctx)
	_ = c.Edit(text, tele.ModeHTML)
	_, _ = h.bot.EditReplyMarkup(c.Callback(), BuildAdminMenu(bookings))
	return c.Respond(&tele.CallbackResponse{Text: fmt.Sprintf("Запись #%d удалена", id)})
}

func (h *Handlers) handleConfirmReminder(c tele.Context) error {
	id, booking, err := h.reminderBooking(c, "confirm_remind_")
	if err != nil {
		return err
	}
	if _, err := h.bot.EditReplyMarkup(c.Callback(), &tele.ReplyMarkup{}); err != nil {
		log.Printf("⚠️ Не удалось убрать кнопки напоминания id=%d: %v", id, err)
	}
	if h.adminID != 0 {
		adminText := fmt.Sprintf(
			"🔔 Клиент подтвердил запись!\nУслуга: %s\nДата: %s\nВремя: %s\nТелефон: %s",
			html.EscapeString(booking.ServiceName),
			html.EscapeString(booking.Date),
			html.EscapeString(booking.TimeSlot),
			html.EscapeString(booking.Phone),
		)
		_, _ = h.bot.Send(tele.ChatID(h.adminID), adminText, tele.ModeHTML)
	}
	_ = c.Respond()
	return c.Send("✅ Спасибо, мы ждем вас!")
}

func (h *Handlers) handleCancelReminder(c tele.Context) error {
	id, booking, err := h.reminderBooking(c, "cancel_remind_")
	if err != nil {
		return err
	}
	if err := h.bookingService.DeleteBookingByID(context.Background(), id); err != nil {
		log.Printf("❌ Ошибка отмены записи из напоминания id=%d: %v", id, err)
		return c.Respond(&tele.CallbackResponse{Text: "Не удалось отменить запись", ShowAlert: true})
	}
	if _, err := h.bot.EditReplyMarkup(c.Callback(), &tele.ReplyMarkup{}); err != nil {
		log.Printf("⚠️ Не удалось убрать кнопки напоминания id=%d: %v", id, err)
	}
	if h.adminID != 0 {
		adminText := fmt.Sprintf(
			"⚠️ Внимание! Клиент ОТМЕНИЛ запись!\nУслуга: %s\nДата: %s\nВремя: %s\nТелефон: %s",
			html.EscapeString(booking.ServiceName),
			html.EscapeString(booking.Date),
			html.EscapeString(booking.TimeSlot),
			html.EscapeString(booking.Phone),
		)
		_, _ = h.bot.Send(tele.ChatID(h.adminID), adminText, tele.ModeHTML)
	}
	_ = c.Respond()
	return c.Send("❌ Ваша запись отменена.")
}

func (h *Handlers) reminderBooking(c tele.Context, prefix string) (int, *domain.Booking, error) {
	id, err := strconv.Atoi(strings.TrimPrefix(c.Data(), prefix))
	if err != nil {
		_ = c.Respond(&tele.CallbackResponse{Text: "Некорректный ID", ShowAlert: true})
		return 0, nil, fmt.Errorf("bad id")
	}
	booking, err := h.bookingService.GetBookingByID(context.Background(), id)
	if err != nil {
		_ = c.Respond(&tele.CallbackResponse{Text: "Запись не найдена", ShowAlert: true})
		return 0, nil, err
	}
	if booking.UserID != c.Sender().ID {
		_ = c.Respond(&tele.CallbackResponse{Text: "Это не ваша запись", ShowAlert: true})
		return 0, nil, fmt.Errorf("forbidden")
	}
	return id, booking, nil
}

// --- ОБРАБОТКА ЗАМЕНЫ ЗАПИСИ ---

func (h *Handlers) handleConfirmReplace(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	// 1. Отменяем старую подтверждённую запись
	err := h.bookingService.CancelConfirmedBooking(ctx, userID)
	if err != nil {
		log.Printf("❌ Ошибка отмены старой записи для userID=%d: %v", userID, err)
		return c.Send("❌ Произошла ошибка при отмене старой записи. Обратитесь к администратору.")
	}

	log.Printf("✅ Старая запись отменена для userID=%d, открываем WebApp для новой", userID)

	// 2. Получаем черновик, чтобы узнать выбранную услугу
	draft, err := h.bookingService.GetUserDraft(ctx, userID)
	if err != nil {
		log.Printf("❌ Ошибка получения черновика для userID=%d: %v", userID, err)
		return c.Send("❌ Произошла ошибка. Попробуйте начать запись заново.")
	}

	_ = c.Delete()

	text := fmt.Sprintf("✅ Старая запись отменена.\n\n*Выбрана услуга:*\n`%s`\n\nНажмите на кнопку «Запись» слева внизу экрана, чтобы выбрать новую дату и время.", draft.ServiceName)
	return c.Send(text, tele.ModeMarkdown)
}

func (h *Handlers) handleKeepOldBooking(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	// Удаляем черновик, так как пользователь передумал
	_ = h.bookingService.CancelDraftBooking(ctx, userID)

	_ = c.Delete()
	return c.Send("👌 Вы отменили замену. Ваша старая запись остаётся в силе!", BuildInlineMainMenu(h.webAppBaseURL), tele.ModeMarkdown)
}
