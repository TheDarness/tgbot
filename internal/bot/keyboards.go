package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func GetStartKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Дізнатися курс 💰"),
			tgbotapi.NewKeyboardButton("Останні запити ⏳"),
		),
	)
}

func GetCryptoSelectionKeyboard() tgbotapi.InlineKeyboardMarkup {
	popularCryptos := []string{"BTC", "ETH", "BNB", "XRP", "ADA", "DOGE", "SOL", "DOT", "MATIC", "LTC"}
	var rows []tgbotapi.InlineKeyboardButton
	var keyboardRows [][]tgbotapi.InlineKeyboardButton

	for _, crypto := range popularCryptos {
		rows = append(rows, tgbotapi.NewInlineKeyboardButtonData(crypto, "select_crypto:"+crypto))
		if len(rows) == 5 {
			keyboardRows = append(keyboardRows, rows)
			rows = nil
		}
	}
	if len(rows) > 0 {
		keyboardRows = append(keyboardRows, rows)
	}

	return tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
}

func GetQuoteSelectionKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇺🇦 Гривня (UAH)", "select_quote:UAH"),
			tgbotapi.NewInlineKeyboardButtonData("🇪🇺 Євро (EUR)", "select_quote:EUR"),
			tgbotapi.NewInlineKeyboardButtonData("🇺🇸 Долар (USD)", "select_quote:USD"),
		),
	)
}
