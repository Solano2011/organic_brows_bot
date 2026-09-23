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
	adminID        int64
	bot            *tele.Bot
	webAppBaseURL  string
}

func NewHandlers(bs domain.BookingService, adminID int64, bot *tele.Bot, webAppURL string) *Handlers {
	return &Handlers{
		bookingService: bs,
		adminID:        adminID,
		bot:            bot,
		webAppBaseURL:  webAppURL,
	}
}

func (h *Handlers) InitRoutes(b *tele.Bot) {
	b.Handle("/start", h.handleStart)
	b.Handle("/admin", h.handleAdmin)

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
				"💅 Услуга: `%s`\n"+
				"📅 Дата: `%s`\n"+
				"⏰ Время: `%s`\n"+
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
			"🔔 *НОВАЯ ЗАПИСЬ В СИСТЕМЕ*\n"+
				"━━━━━━━━━━━━━━━\n"+
				"👤 *Имя:* %s\n"+
				"📞 *Телефон:* %s\n"+
				"🆔 Гость: %s\n"+
				"💅 *Услуга:* %s\n"+
				"📅 *Дата:* %s\n"+
				"⏰ *Время:* %s\n"+
				"💬 *Комментарий:* %s",
			booking.UserName, booking.Phone, usernameStr, booking.ServiceName, booking.Date, booking.TimeSlot, booking.Comment,
		)
		go func(msg string) { _, _ = h.bot.Send(tele.ChatID(h.adminID), msg, tele.ModeMarkdown) }(notifyText)
	}

	// Подтверждаем пользователю
	text := fmt.Sprintf(
		"✅ *Запись успешно подтверждена!*\n"+
			"━━━━━━━━━━━━━━━\n"+
			"Услуга: `%s`\n"+
			"Дата: `%s`\n"+
			"Время: `%s`\n"+
			"Имя: `%s`\n"+
			"Телефон: `%s`\n"+
			"💬 Комментарий: `%s`\n\n"+
			"Жду вас! 💖",
		booking.ServiceName, booking.Date, booking.TimeSlot, booking.UserName, booking.Phone, booking.Comment,
	)

	return c.Send(text, BuildInlineMainMenu(h.webAppBaseURL), tele.ModeMarkdown)
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

	text := fmt.Sprintf(
		"📋 *Ваша активная запись:*\n"+
			"━━━━━━━━━━━━━━━\n"+
			"Услуга: `%s`\n"+
			"Дата: `%s`\n"+
			"Время: `%s`\n"+
			"Имя: `%s`\n"+
			"Телефон: `%s`\n"+
			"Комментарий: `%s`\n\n"+
			"Для отмены записи используйте кнопку ниже.",
		b.ServiceName, b.Date, b.TimeSlot, b.UserName, b.Phone, b.Comment,
	)

	m := &tele.ReplyMarkup{}
	m.Inline(m.Row(BtnCancelBooking))

	return c.Send(text, m, tele.ModeMarkdown)
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

	text := fmt.Sprintf(
		"📋 *Ваша запись:*\n"+
			"━━━━━━━━━━━━━━━\n"+
			"Услуга: `%s`\n"+
			"Дата: `%s`\n"+
			"Время: `%s`\n"+
			"Имя: `%s`\n"+
			"Телефон: `%s`\n"+
			"Комментарий: `%s`",
		b.ServiceName, b.Date, b.TimeSlot, b.UserName, b.Phone, b.Comment,
	)
	return c.Send(text, m, tele.ModeMarkdown)
}

func (h *Handlers) handleCancelBooking(c tele.Context) error {
	ctx := context.Background()
	_ = h.bookingService.CancelBooking(ctx, c.Sender().ID)
	_ = c.Delete()

	// Отвечаем на callback
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
	if strings.HasPrefix(data, "del_book:") {
		return h.handleDeleteBooking(c)
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

	// 3. Открываем WebApp через REPLY-клавиатуру (не Inline!)
	webAppURL := fmt.Sprintf("%s?service=%s", h.webAppBaseURL, strings.ReplaceAll(draft.ServiceName, " ", "+"))

	text := fmt.Sprintf("✅ Старая запись отменена.\n\n💅 *Выбрана услуга:*\n`%s`\n\nНажмите кнопку ниже, чтобы выбрать дату и время:", draft.ServiceName)
	return c.Send(text, BuildWebAppReplyKeyboard(webAppURL), tele.ModeMarkdown)
}

func (h *Handlers) handleKeepOldBooking(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	// Удаляем черновик, так как пользователь передумал
	_ = h.bookingService.CancelDraftBooking(ctx, userID)

	_ = c.Delete()
	return c.Send("👌 Вы отменили замену. Ваша старая запись остаётся в силе!", BuildInlineMainMenu(h.webAppBaseURL), tele.ModeMarkdown)
}
