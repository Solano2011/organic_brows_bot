package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"hookah-bot/internal/app"
	"hookah-bot/internal/repository/postgres"

	"github.com/joho/godotenv"
)

func main() {
	// Загружаем переменные из файла .env
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем системные переменные")
	}

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("Укажите BOT_TOKEN в переменных окружения")
	}

	adminIDs := parseAdminIDs(os.Getenv("ADMIN_IDS"))

	webAppURL := getEnvOrDefault("WEBAPP_URL", "https://beauty-bot.example.com")

	// Строим строку подключения из переменных окружения
	dbHost := getEnvOrDefault("DB_HOST", "localhost")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "hookah_user")
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		log.Fatal("Укажите DB_PASSWORD в переменных окружения")
	}
	dbName := getEnvOrDefault("DB_NAME", "hookah_db")
	dbSSLMode := getEnvOrDefault("DB_SSLMODE", "require")

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode)

	db, err := postgres.NewPostgresDB(connString)
	if err != nil {
		log.Fatalf("Не удалось инициализировать базу данных: %v", err)
	}
	// Закрываем соединение при остановке бота
	defer db.Conn.Close(context.Background())

	// Передаем db и webAppURL внутрь app.Run
	app.Run(token, adminIDs, db, webAppURL)
}

func parseAdminIDs(raw string) []int64 {
	var ids []int64
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id == 0 {
			log.Printf("Некорректный ADMIN_IDS: %q", part)
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
