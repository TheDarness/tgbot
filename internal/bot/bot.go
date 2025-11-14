package bot

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func Run(telegramToken, apiKey string) error {
	b, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		return fmt.Errorf("помилка ініціалізації Telegram бота: %v", err)
	}

	b.Debug = true
	log.Printf("Авторизовано як @%s", b.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.GetUpdatesChan(u)

	LoadHistoryFromFile()

	log.Println("Бот запущено. Очікування повідомлень...")
	handleUpdates(b, updates, apiKey)
	return nil
}
