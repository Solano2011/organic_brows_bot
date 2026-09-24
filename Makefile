.PHONY: deploy

deploy:
	@echo "⬇️  Скачиваем обновления с GitHub..."
	git pull
	@echo "🔨 Собираем Go-приложение..."
	go build -o bot_app ./cmd/bot
	@echo "🔄 Перезапускаем сервис бота..."
	systemctl restart organic_bot
	@echo "✅ Обновление успешно завершено!"
