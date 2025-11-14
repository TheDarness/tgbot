package main

import (
	"log"
	"os"

	"crypto-rate-bot/internal/bot"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Помилка завантаження .env файлу: %v", err)
	}

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	coinAPIKey := os.Getenv("COINAPI_KEY")

	if telegramToken == "" || coinAPIKey == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN або COINAPI_KEY не знайдено у .env файлі")
	}

	if err := bot.Run(telegramToken, coinAPIKey); err != nil {
		log.Fatalf("Помилка запуску бота: %v", err)
	}
}
