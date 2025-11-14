package bot

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"crypto-rate-bot/internal/api"
	"crypto-rate-bot/pkg/format"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var coinRateLimit = make(map[int64]time.Time)
var coinRateLimitMutex sync.Mutex

const limitDuration = 10 * time.Second

func FormatToKyivTime(timeStr string) string {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return timeStr
	}

	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		return t.Local().Format("02.01.2006 15:04:05 (MST)")
	}

	return t.In(loc).Format("02.01.2006 15:04:05 (EEST/EET)")
}

func ParseAssetInput(input string) (amount float64, cryptoCode string, ok bool) {
	input = strings.TrimSpace(input)
	re := regexp.MustCompile(`^(\d*\.?\d+)?\s*([a-zA-Z]{2,5})$`)
	matches := re.FindStringSubmatch(input)

	if len(matches) == 3 && (matches[1] != "" || matches[2] != "") {
		cryptoCode = strings.ToUpper(matches[2])
		if matches[1] != "" {
			amount, _ = strconv.ParseFloat(matches[1], 64)
		} else {
			amount = 1.0
		}
	} else {
		cryptoCode = strings.ToUpper(input)
		amount = 1.0
	}

	if len(cryptoCode) < 2 || len(cryptoCode) > 5 || amount == 0 {
		return 0, "", false
	}
	return amount, cryptoCode, true
}

func handleUpdates(bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel, apiKey string) {
	for update := range updates {
		if update.Message != nil {
			handleMessage(bot, update, apiKey)
		} else if update.CallbackQuery != nil {
			handleCallback(bot, update.CallbackQuery, apiKey)
		}
	}
}

func handleMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update, apiKey string) {
	msg := update.Message

	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			welcome := tgbotapi.NewMessage(msg.Chat.ID, "Вітаю! Я допоможу вам дізнатися курс криптовалюти! 👋")
			welcome.ReplyMarkup = GetStartKeyboard()
			bot.Send(welcome)
			DeleteUserContext(msg.Chat.ID)
			return
		case "coin":
			HandleCoinCommand(bot, update, apiKey)
			return
		}
	}

	switch msg.Text {
	case "Дізнатися курс 💰":
		response := tgbotapi.NewMessage(msg.Chat.ID, "Оберіть криптовалюту або введіть свою (наприклад, **0.5 ETH**):")
		response.ParseMode = "Markdown"
		response.ReplyMarkup = GetCryptoSelectionKeyboard()
		InitUserSelection(msg.Chat.ID)
		bot.Send(response)
		return

	case "Останні запити ⏳":
		keyboard := GetHistoryKeyboard(msg.Chat.ID)
		if len(keyboard.InlineKeyboard) == 0 {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ваша історія запитів порожня."))
			return
		}
		msg := tgbotapi.NewMessage(msg.Chat.ID, "Оберіть запит з історії:")
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
		return
	}

	if IsSelectingAsset(msg.Chat.ID) {
		amount, cryptoCode, valid := ParseAssetInput(msg.Text)
		if !valid {
			errorMsg := tgbotapi.NewMessage(msg.Chat.ID, "Введіть коректну валюту (наприклад, **BTC**) або суму та код (наприклад, **0.5 ETH**).")
			errorMsg.ParseMode = "Markdown"
			bot.Send(errorMsg)
			return
		}

		SetUserSelection(msg.Chat.ID, cryptoCode, amount)
		text := fmt.Sprintf("Ви обрали **%.2f %s**. Тепер оберіть валюту:", amount, cryptoCode)
		reply := tgbotapi.NewMessage(msg.Chat.ID, text)
		reply.ParseMode = "Markdown"
		reply.ReplyMarkup = GetQuoteSelectionKeyboard()
		bot.Send(reply)
	}
}

func handleCallback(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, apiKey string) {
	data := strings.Split(cb.Data, ":")
	if len(data) != 2 {
		return
	}
	action, value := data[0], data[1]
	chatID := cb.Message.Chat.ID
	msgID := cb.Message.MessageID

	switch action {
	case "select_crypto":
		SetUserSelection(chatID, value, 1.0)
		text := fmt.Sprintf("Ви обрали **%s**. Тепер оберіть валюту:", value)
		edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
		edit.ParseMode = "Markdown"
		keyb := GetQuoteSelectionKeyboard()
		edit.ReplyMarkup = &keyb
		bot.Send(edit)

	case "select_quote":
		selection, ok := GetUserSelection(chatID)
		if !ok || selection.Asset == "" {
			bot.Send(tgbotapi.NewMessage(chatID, "Помилка: спершу оберіть криптовалюту."))
			return
		}
		go sendRateResult(bot, chatID, msgID, apiKey, selection.Asset, value, selection.Amount, cb.ID)

	case "history_req":
		parts := strings.Split(value, "/")
		if len(parts) != 2 {
			return
		}
		base, quote := parts[0], parts[1]
		go sendRateResult(bot, chatID, msgID, apiKey, base, quote, 1.0, cb.ID)
	}
}

func sendRateResult(bot *tgbotapi.BotAPI, chatID int64, messageID int, apiKey, base, quote string, amount float64, cbID string) {
	rate, err := api.GetCryptoRate(apiKey, base, quote)
	var text string

	if err != nil {
		log.Printf("Помилка отримання курсу: %v", err)
		text = fmt.Sprintf("❌ Помилка при отриманні курсу **%s/%s**:\n%s", base, quote, err.Error())
	} else {
		finalRate := *rate.Rate * amount
		pair := fmt.Sprintf("%s/%s", *rate.AssetIDBase, *rate.AssetIDQuote)
		SaveToHistory(chatID, pair)
		text = fmt.Sprintf("📊 Курс **%.2f %s** до **%s**:\n\n**%.2f %s = %.4f %s**\n\n_Дані актуальні станом на %s_",
			amount, *rate.AssetIDBase, *rate.AssetIDQuote,
			amount, *rate.AssetIDBase, finalRate, *rate.AssetIDQuote,
			FormatToKyivTime(*rate.Time))
	}

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = "Markdown"
	bot.Send(edit)
	bot.Send(tgbotapi.NewCallback(cbID, "✅ Курс отримано!"))
}

func HandleCoinCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update, apiKey string) {
	chatID := update.Message.Chat.ID
	msg := update.Message

	coinRateLimitMutex.Lock()
	lastRequestTime, exists := coinRateLimit[chatID]
	if exists && time.Since(lastRequestTime) < limitDuration {
		coinRateLimitMutex.Unlock()
		remaining := (limitDuration - time.Since(lastRequestTime)).Seconds()
		text := fmt.Sprintf("🛑 **Обмеження запитів**: Спробуйте знову через **%.1f сек.**", remaining)
		sendCoinResponse(bot, chatID, text, true)
		return
	}
	coinRateLimit[chatID] = time.Now()
	coinRateLimitMutex.Unlock()

	args := msg.CommandArguments()
	parts := strings.Fields(args)

	symbol := ""
	fiat := os.Getenv("DEFAULT_FIAT")
	if fiat == "" {
		fiat = "UAH"
	}

	if len(parts) >= 1 {
		symbol = strings.ToUpper(parts[0])
	}
	if len(parts) >= 2 {
		fiat = strings.ToUpper(parts[1])
	}

	if symbol == "" {
		text := fmt.Sprintf("Будь ласка, вкажіть символ. Наприклад: `/coin BTC %s`", fiat)
		sendCoinResponse(bot, chatID, text, true)
		return
	}

	stats, err := api.GetCoinStats(apiKey, symbol, fiat)

	var responseText string

	if err != nil {
		log.Printf("Помилка отримання статистики для %s/%s: %v", symbol, fiat, err)
		responseText = fmt.Sprintf("❌ **Помилка отримання курсу %s/%s**:\n%s", symbol, fiat, err.Error())
	} else {
		rateFormatted := format.FormatNumber(stats.Rate, 6)
		minFormatted := format.FormatNumber(stats.Min24h, 6)
		maxFormatted := format.FormatNumber(stats.Max24h, 6)
		timeFormatted := FormatToKyivTime(stats.Time)

		responseText = fmt.Sprintf(
			"📊 **Курс %s/%s**\n\n"+
				"💰 Ціна зараз: **%s %s**\n"+
				"📉 24h MIN: %s %s\n"+
				"📈 24h MAX: %s %s\n\n"+
				"🌐 Агрегатор: %s\n"+
				"⏰ Оновлено: %s",
			symbol, fiat,
			rateFormatted, fiat,
			minFormatted, fiat,
			maxFormatted, fiat,
			stats.Aggregator,
			timeFormatted,
		)
	}

	sendCoinResponse(bot, chatID, responseText, true)
}

func sendCoinResponse(bot *tgbotapi.BotAPI, chatID int64, text string, useMarkdown bool) {
	msg := tgbotapi.NewMessage(chatID, text)
	if useMarkdown {
		msg.ParseMode = "Markdown"
	}
	bot.Send(msg)
}
