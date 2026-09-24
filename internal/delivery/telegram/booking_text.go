package telegram

import (
	"fmt"
	"html"
)

const masterContact = "Александра Смелая, <a href='tel:+79963051404'>+7 (996) 305-14-04</a>"

var servicePrices = map[string]string{
	"Organic brow 🍈 + оформление бровей + ламинирование ресниц": "3500 ₽",
	"Ламинирование ресниц + ламинирование бровей":               "3400 ₽",
	"Ламинирование ресниц + натуральное оформление бровей 🪞":    "3000 ₽",
	"Ламинирование ресниц + снятие наращенных ресниц":           "2000 ₽",
	"Ламинирование ресниц 🐚":                                    "1800 ₽",
	"Коррекция бровей":                                          "1000 ₽",
	"Ламинирование бровей":                                      "1600 ₽",
	"Натуральное оформление бровей 🐚":                           "1600 ₽",
	"Осветление бровей":                                         "1800 ₽",
	"Полный комплекс ламинирования бровей":                      "2000 ₽",
	"Organic brow 🍈 + коррекция":                                "1500 ₽",
	"Organic brow 🍈 + натуральное оформление бровей":            "2000 ₽",
	"Удаление пушка на губой":                                   "300 ₽",
}

func servicePrice(name string) string {
	if price, ok := servicePrices[name]; ok {
		return price
	}
	return name
}

func FormatClientBooking(serviceName, date, timeSlot string) string {
	return fmt.Sprintf(
		"Вы записались к специалисту:\n"+
			"%s\n\n"+
			"Услуга: %s\n"+
			"Стоимость услуги: %s\n\n"+
			"Дата: %s\n"+
			"Время: %s\n"+
			"Адрес: Советская улица, 7/6, 2 этаж, 10 кабинет",
		masterContact,
		html.EscapeString(serviceName),
		html.EscapeString(servicePrice(serviceName)),
		html.EscapeString(date),
		html.EscapeString(timeSlot),
	)
}

func FormatReminder(intro, serviceName, date, timeSlot string) string {
	return fmt.Sprintf(
		"%s %s\nАдрес: Советская улица, 7/6, 2 этаж, 10 кабинет\nУслуга: %s\nДата: %s\nВремя: %s\nСтоимость услуги: %s",
		intro,
		masterContact,
		html.EscapeString(serviceName),
		html.EscapeString(date),
		html.EscapeString(timeSlot),
		html.EscapeString(servicePrice(serviceName)),
	)
}
