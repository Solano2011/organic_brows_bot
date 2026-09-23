package app

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"hookah-bot/internal/delivery/telegram"
	"hookah-bot/internal/repository/postgres"
	"hookah-bot/internal/service"
	"hookah-bot/internal/validation"

	tele "gopkg.in/telebot.v3"
)

// Добавили db *postgres.DB третьим параметром и webAppURL четвертым
func Run(token string, adminID int64, db *postgres.DB, webAppURL string) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	// Устанавливаем меню команд (только /start, без /admin)
	commands := []tele.Command{
		{Text: "start", Description: "🌟 Главное меню"},
	}
	if err := b.SetCommands(commands); err != nil {
		log.Printf("⚠️ Не удалось установить команды бота: %v", err)
	} else {
		log.Println("✅ Меню команд бота успешно установлено")
	}

	// Заменяем кнопку меню (слева от поля ввода) на прямую кнопку WebApp
	menuButton := &tele.MenuButton{
		Type:   tele.MenuButtonWebApp,
		Text:   "Запись",
		WebApp: &tele.WebApp{URL: webAppURL},
	}
	if err := b.SetMenuButton(&tele.User{}, menuButton); err != nil {
		log.Printf("⚠️ Не удалось установить кнопку меню WebApp: %v", err)
	} else {
		log.Println("✅ Кнопка меню WebApp успешно установлена")
	}

	repo := postgres.NewBookingRepo(db)

	bookingService := service.NewBookingService(repo)
	handlers := telegram.NewHandlers(bookingService, adminID, b, webAppURL)
	handlers.InitRoutes(b)

	log.Printf("Бот @%s успешно запущен! Admin ID: %d", b.Me.Username, adminID)

	// --- ЗАПУСК ВЕБ-СЕРВЕРА ---
	go func() {
		// Rate limiting map (потокобезопасная реализация)
		var rateLimitMap sync.Map

		http.HandleFunc("/api/book", func(w http.ResponseWriter, r *http.Request) {
			// CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			if r.Method != "POST" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var data struct {
				ServiceName string `json:"serviceName"`
				Time        string `json:"time"`
				Date        string `json:"date"`
				UserID      int64  `json:"userId"`
				Name        string `json:"name"`
				Phone       string `json:"phone"`
				Comment     string `json:"comment"`
				InitData    string `json:"initData"`
			}

			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				http.Error(w, "Invalid request format", http.StatusBadRequest)
				return
			}

			// Очищаем все строковые поля от лишних пробелов
			data.ServiceName = strings.TrimSpace(data.ServiceName)
			data.Time = strings.TrimSpace(data.Time)
			data.Date = strings.TrimSpace(data.Date)
			data.Name = strings.TrimSpace(data.Name)
			data.Phone = strings.TrimSpace(data.Phone)
			data.Comment = strings.TrimSpace(data.Comment)

			// КРИТИЧНО: Проверка подлинности запроса от Telegram WebApp
			if data.InitData != "" {
				if err := validation.VerifyTelegramWebAppData(data.InitData, token); err != nil {
					log.Printf("⚠️ Неверная подпись WebApp: %v", err)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}

			// Rate limiting: не более 1 запроса в 5 секунд от одного пользователя
			if val, ok := rateLimitMap.Load(data.UserID); ok {
				lastReq := val.(time.Time)
				if time.Since(lastReq) < 5*time.Second {
					http.Error(w, "Too many requests", http.StatusTooManyRequests)
					return
				}
			}
			rateLimitMap.Store(data.UserID, time.Now())

			// Валидация входных данных
			if err := validation.ValidateName(data.Name); err != nil {
				http.Error(w, fmt.Sprintf("Invalid name: %v", err), http.StatusBadRequest)
				return
			}

			if err := validation.ValidatePhone(data.Phone); err != nil {
				http.Error(w, fmt.Sprintf("Invalid phone: %v", err), http.StatusBadRequest)
				return
			}

			if err := validation.ValidateTimeSlot(data.Time); err != nil {
				http.Error(w, fmt.Sprintf("Invalid time slot: %v", err), http.StatusBadRequest)
				return
			}

			if err := validation.ValidateDate(data.Date); err != nil {
				http.Error(w, fmt.Sprintf("Invalid date: %v", err), http.StatusBadRequest)
				return
			}

			ctx := context.Background()

			log.Printf("🌐 [HTTP API] Получен запрос на запись от userID=%d: услуга=%s, дата=%s, время=%s",
				data.UserID, data.ServiceName, data.Date, data.Time)

			// Проверяем, есть ли у пользователя существующая запись
			existingBooking, err := bookingService.GetUserBooking(ctx, data.UserID)
			if err == nil && existingBooking.TimeSlot != "" {
				log.Printf("⚠️ [HTTP API] У userID=%d найдена существующая запись: услуга=%s, дата=%s, время=%s",
					data.UserID, existingBooking.ServiceName, existingBooking.Date, existingBooking.TimeSlot)

				// Сохраняем новые данные в черновик
				if err := bookingService.StartBookingDraft(ctx, data.UserID, data.ServiceName); err != nil {
					log.Printf("❌ [HTTP API] Ошибка создания черновика для userID=%d: %v", data.UserID, err)
					http.Error(w, "Failed to create booking draft", http.StatusInternalServerError)
					return
				}

				if err := bookingService.SetBookingDate(ctx, data.UserID, data.Date); err != nil {
					log.Printf("❌ [HTTP API] Ошибка сохранения даты в черновик для userID=%d: %v", data.UserID, err)
					http.Error(w, "Failed to save date to draft", http.StatusInternalServerError)
					return
				}

				if err := bookingService.SetDraftTimeAndContacts(ctx, data.UserID, data.Time, data.Name, data.Phone, data.Comment); err != nil {
					log.Printf("❌ [HTTP API] Ошибка сохранения времени и контактов в черновик для userID=%d: %v", data.UserID, err)
					http.Error(w, "Failed to save time and contacts to draft", http.StatusInternalServerError)
					return
				}

				log.Printf("✅ [HTTP API] Черновик сохранён для userID=%d, ожидаем подтверждения", data.UserID)

				// Отправляем пользователю сообщение с выбором
				user := &tele.User{ID: data.UserID}
				text := fmt.Sprintf(
					"⚠️ *У вас уже есть активная запись:*\n\n"+
						"💅 Услуга: `%s`\n"+
						"📅 Дата: `%s`\n"+
						"⏰ Время: `%s`\n\n"+
						"Хотите отменить предыдущую запись и создать новую на `%s` (`%s`) в `%s`?",
					existingBooking.ServiceName, existingBooking.Date, existingBooking.TimeSlot,
					data.ServiceName, data.Date, data.Time,
				)

				_, sendErr := b.Send(user, text, telegram.BuildReplaceConfirmMenu(), tele.ModeMarkdown)
				if sendErr != nil {
					log.Printf("⚠️ Ошибка при отправке сообщения выбора для userID=%d: %v", data.UserID, sendErr)
				}

				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"status": "pending_confirmation"})
				return
			}

			log.Printf("ℹ️ [HTTP API] У userID=%d нет существующих записей, создаем новую", data.UserID)

			// Создаем новую запись
			if err := bookingService.StartBookingDraft(ctx, data.UserID, data.ServiceName); err != nil {
				log.Printf("❌ [HTTP API] Ошибка создания черновика для userID=%d: %v", data.UserID, err)
				http.Error(w, "Failed to create booking draft", http.StatusInternalServerError)
				return
			}

			if err := bookingService.SetBookingDate(ctx, data.UserID, data.Date); err != nil {
				log.Printf("❌ [HTTP API] Ошибка сохранения даты для userID=%d: %v", data.UserID, err)
				http.Error(w, "Failed to save date", http.StatusInternalServerError)
				return
			}

			booking, err := bookingService.CompleteBookingDraft(ctx, data.UserID, data.Time, data.Name, data.Phone, data.Comment)
			if err != nil {
				log.Printf("❌ [HTTP API] Ошибка завершения записи для userID=%d: %v", data.UserID, err)
				http.Error(w, "Booking conflict or error", http.StatusConflict)
				return
			}

			log.Printf("✅ [HTTP API] Запись успешно создана для userID=%d: услуга=%s, дата=%s, время=%s",
				data.UserID, booking.ServiceName, booking.Date, booking.TimeSlot)

			// Отправляем сообщение пользователю
			user := &tele.User{ID: data.UserID}
			confirmText := fmt.Sprintf(
				"✅ *Запись успешно подтверждена!*\n"+
					"━━━━━━━━━━━━━━━\n"+
					"Услуга: `%s`\n"+
					"Дата: `%s`\n"+
					"Время: `%s`\n"+
					"Имя: `%s`\n"+
					"Телефон: `%s`\n",
				booking.ServiceName, booking.Date, booking.TimeSlot, booking.UserName, booking.Phone,
			)

			if booking.Comment != "" {
				confirmText += fmt.Sprintf("💬 Комментарий: `%s`\n", booking.Comment)
			}

			confirmText += "\nСтатус: *Подтверждено*\n\nЖдем вас!"

			_, err = b.Send(user, confirmText, telegram.BuildInlineMainMenu(webAppURL), tele.ModeMarkdown)
			if err != nil {
				log.Printf("⚠️ Ошибка при отправке сообщения: %v", err)
			}

			// Уведомляем админа
			if adminID != 0 {
				adminText := fmt.Sprintf(
					"🔔 *НОВАЯ ЗАПИСЬ*\n"+
						"━━━━━━━━━━━━━━━\n"+
						"👤 *Имя:* %s\n"+
						"📞 *Телефон:* %s\n"+
						"🆔 ID: `%d`\n"+
						"💅 Услуга: *%s*\n"+
						"📅 Дата: *%s*\n"+
						"⏰ Время: *%s*\n",
					booking.UserName, booking.Phone, booking.UserID,
					booking.ServiceName, booking.Date, booking.TimeSlot,
				)

				if booking.Comment != "" {
					adminText += fmt.Sprintf("💬 Комментарий: *%s*\n", booking.Comment)
				}

				admin := &tele.User{ID: adminID}
				_, adminErr := b.Send(admin, adminText, tele.ModeMarkdown)
				if adminErr != nil {
					log.Printf("⚠️ Ошибка при отправке уведомления админу: %v", adminErr)
				}
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "confirmed"})
		})

		// --- ЭНДПОИНТ ДЛЯ СОЗДАНИЯ ЧЕРНОВИКА ---
		http.HandleFunc("/api/start-draft", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			if r.Method != "POST" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var data struct {
				UserID      int64  `json:"user_id"`
				ServiceName string `json:"service_name"`
			}

			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				http.Error(w, "Invalid request format", http.StatusBadRequest)
				return
			}

			if data.UserID == 0 || data.ServiceName == "" {
				http.Error(w, "Missing user_id or service_name", http.StatusBadRequest)
				return
			}

			ctx := context.Background()

			// Создаем черновик записи для пользователя
			if err := bookingService.StartBookingDraft(ctx, data.UserID, data.ServiceName); err != nil {
				log.Printf("❌ [API] Ошибка создания черновика для userID=%d, услуга=%s: %v", data.UserID, data.ServiceName, err)
				http.Error(w, "Failed to create booking draft", http.StatusInternalServerError)
				return
			}

			log.Printf("✅ [API] Черновик создан для userID=%d, услуга=%s", data.UserID, data.ServiceName)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
		})

		// --- ЭНДПОИНТ ДЛЯ ПРОВЕРКИ ЗАНЯТОСТИ ---
		http.HandleFunc("/api/availability", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			if r.Method != "GET" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			date := r.URL.Query().Get("date")

			if date == "" {
				http.Error(w, "Missing date parameter", http.StatusBadRequest)
				return
			}

			ctx := context.Background()
			takenSlots, err := repo.GetTakenTimeSlots(ctx, date, "")
			if err != nil {
				log.Printf("❌ Ошибка получения занятых слотов для даты %s: %v", date, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string][]string{"takenSlots": takenSlots})
		})
		// ---------------------------------------------

		// 1. Отдаем статические файлы (картинки) по пути /img/
		http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("./webapp/img"))))

		// 2. Отдаем статические файлы (CSS, JS) по пути /static/
		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

		// 3. Обрабатываем главную страницу через шаблонизатор
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}

			tmpl, err := template.ParseGlob("webapp/templates/*.html")
			if err != nil {
				log.Printf("Ошибка загрузки шаблонов: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		log.Println("Локальный сервер Web App запущен на http://localhost:8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal("Ошибка запуска сервера: ", err)
		}
	}()
	// --------------------------

	b.Start()
}
