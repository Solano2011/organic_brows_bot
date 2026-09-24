package telegram

import (
	"fmt"

	"hookah-bot/internal/domain"

	tele "gopkg.in/telebot.v3"
)

var (
	Menu = &tele.ReplyMarkup{}

	// Главное меню (Inline-кнопки под сообщением)
	BtnMyBooking  = Menu.Data("📅 Моя запись", "btn_my_booking")
	BtnMyBookings = Menu.Data("📅 Моя бронь", "my_bookings")
	BtnHowToGet   = Menu.Data("📍 Как пройти", "how_to_get")
	BtnContacts   = Menu.Data("📍 Контакты", "btn_contacts")

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

	btnWebApp := m.WebApp("✨ Записаться", &tele.WebApp{URL: webAppURL})

	m.Inline(
		m.Row(btnWebApp),
		m.Row(BtnMyBookings),
	)
	return m
}

func BuildBookingSuccessMenu(webAppURL string) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	btnWebApp := m.WebApp("✨ Записаться", &tele.WebApp{URL: webAppURL})
	m.Inline(
		m.Row(btnWebApp),
		m.Row(BtnMyBookings),
		m.Row(BtnHowToGet),
	)
	return m
}

func BuildServicesMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}

	// Organic brow
	btnOrganicBrow := m.Data("🍈 Organic brow", "service", "Organic brow 🍈 + коррекция")
	btnOrganicBrowFull := m.Data("🍈 Organic + оформление", "service", "Organic brow 🍈 + натуральное оформление бровей")

	// Брови
	btnBrowCorrection := m.Data("✨ Коррекция бровей", "service", "Коррекция бровей")
	btnBrowLamination := m.Data("💫 Ламинирование бровей", "service", "Ламинирование бровей")
	btnBrowStyling := m.Data("🐚 Оформление бровей", "service", "Натуральное оформление бровей 🐚")
	btnBrowBleaching := m.Data("🌟 Осветление бровей", "service", "Осветление бровей")
	btnBrowFull := m.Data("💎 Полный комплекс бровей", "service", "Полный комплекс ламинирования бровей")

	// Ресницы
	btnLashLamination := m.Data("🐚 Ламинирование ресниц", "service", "Ламинирование ресниц 🐚")
	btnLashRemoval := m.Data("🔄 Ламин. + снятие", "service", "Ламинирование ресниц + снятие наращенных ресниц")

	// Дополнительно
	btnLipHair := m.Data("➕ Удаление пушка", "service", "Удаление пушка на губой")

	// Комбо
	btnCombo1 := m.Data("🎁 Organic + ламин. ресниц", "service", "Organic brow 🍈 + оформление бровей + ламинирование ресниц")
	btnCombo2 := m.Data("🎁 Ламин. ресниц + бровей", "service", "Ламинирование ресниц + ламинирование бровей")
	btnCombo3 := m.Data("🪞 Ламин. + оформление", "service", "Ламинирование ресниц + натуральное оформление бровей 🪞")

	m.Inline(
		m.Row(btnOrganicBrow, btnOrganicBrowFull),
		m.Row(btnBrowCorrection, btnBrowLamination),
		m.Row(btnBrowStyling, btnBrowBleaching),
		m.Row(btnBrowFull),
		m.Row(btnLashLamination, btnLashRemoval),
		m.Row(btnLipHair),
		m.Row(btnCombo1),
		m.Row(btnCombo2),
		m.Row(btnCombo3),
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

func BuildAdminMenu(bookings []domain.Booking, scheduleURL string) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(bookings)+3)
	for i, b := range bookings {
		btn := tele.Btn{
			Text: fmt.Sprintf("❌ Удалить №%d", i+1),
			Data: fmt.Sprintf("del_book:%s", b.ID),
		}
		rows = append(rows, m.Row(btn))
	}
	if scheduleURL != "" {
		rows = append(rows, m.Row(m.WebApp("⚙️ Настроить график", &tele.WebApp{URL: scheduleURL})))
	}
	rows = append(rows, m.Row(BtnAdminRefresh), m.Row(BtnAdminResetAll))
	m.Inline(rows...)
	return m
}

func BuildReminderMenu(bookingID string) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	confirm := tele.Btn{Text: "✅ Подтвердить запись", Data: "confirm_remind_" + bookingID}
	cancel := tele.Btn{Text: "❌ Отменить запись", Data: "cancel_remind_" + bookingID}
	m.Inline(m.Row(confirm), m.Row(cancel))
	return m
}
