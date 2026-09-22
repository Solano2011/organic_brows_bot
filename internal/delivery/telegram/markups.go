package telegram

import tele "gopkg.in/telebot.v3"

var (
	Menu = &tele.ReplyMarkup{}

	// Главное меню (Inline-кнопки под сообщением)
	BtnMyBooking     = Menu.Data("📅 Моя запись", "btn_my_booking")
	BtnMyBookings    = Menu.Data("📅 Моя бронь", "my_bookings")
	BtnContacts      = Menu.Data("📍 Контакты", "btn_contacts")

	// Навигация
	BtnBackToMain    = Menu.Data("◀️ Назад в меню", "btn_back_main")
	BtnCancelBooking = Menu.Data("❌ Отменить запись", "btn_cancel_booking")

	// Эндпоинты
	BtnService = Menu.Data("", "service")

	// Подтверждение замены записи
	BtnConfirmReplace = Menu.Data("✅ Да, отменить старую", "confirm_replace")
	BtnKeepOldBooking = Menu.Data("❌ Нет, оставить", "keep_old")

	// Админка
	BtnAdminRefresh  = Menu.Data("🔄 Обновить сводку", "admin_refresh")
	BtnAdminResetAll = Menu.Data("🗑 Сбросить все записи", "admin_reset_all")
)

// BuildInlineMainMenu создаёт Inline-клавиатуру главного меню с WebApp кнопкой
func BuildInlineMainMenu(webAppURL string) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}

	btnWebApp := m.WebApp("💅 Записаться", &tele.WebApp{URL: webAppURL})

	m.Inline(
		m.Row(btnWebApp),
		m.Row(BtnMyBookings),
	)
	return m
}

func BuildServicesMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}

	// Услуги бьюти-мастера
	btnManicure := m.Data("💅 Маникюр", "service", "Маникюр")
	btnPedicure := m.Data("🦶 Педикюр", "service", "Педикюр")
	btnExtension := m.Data("💎 Наращивание ногтей", "service", "Наращивание ногтей")
	btnGelPolish := m.Data("✨ Покрытие гель-лак", "service", "Покрытие гель-лак")

	m.Inline(
		m.Row(btnManicure, btnPedicure),
		m.Row(btnExtension, btnGelPolish),
		m.Row(BtnBackToMain),
	)
	return m
}

// BuildWebAppReplyKeyboard создаёт Reply-клавиатуру с кнопкой WebApp
// ВАЖНО: Reply-клавиатура нужна для работы tg.sendData() в WebApp!
func BuildWebAppReplyKeyboard(webAppURL string) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{
		ResizeKeyboard: true, // Компактная клавиатура
	}

	btnWebApp := m.WebApp("📅 Выбрать дату и время", &tele.WebApp{URL: webAppURL})
	btnBack := m.Text("◀️ Назад в меню")

	m.Reply(
		m.Row(btnWebApp),
		m.Row(btnBack),
	)
	return m
}

func BuildContactsMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	btnMap := m.URL("🗺 Открыть на Яндекс.Картах", "https://yandex.ru/maps")
	m.Inline(
		m.Row(btnMap),
		m.Row(BtnBackToMain),
	)
	return m
}

func BuildReplaceConfirmMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(BtnConfirmReplace),
		m.Row(BtnKeepOldBooking),
	)
	return m
}

func BuildAdminMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(BtnAdminRefresh),
		m.Row(BtnAdminResetAll),
	)
	return m
}
