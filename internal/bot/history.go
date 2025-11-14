package bot

import (
	"encoding/json"
	"os"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserSelection struct {
	Asset  string
	Amount float64
}

var (
	userContext = make(map[int64]UserSelection)
	userHistory = make(map[int64][]string)
	mutex       sync.Mutex
	historyFile = "history.json"
)

const maxHistorySize = 5

func IsSelectingAsset(userID int64) bool {
	s, ok := userContext[userID]
	return ok && s.Asset == ""
}

func InitUserSelection(userID int64) {
	userContext[userID] = UserSelection{Asset: "", Amount: 1.0}
}

func SetUserSelection(userID int64, asset string, amount float64) {
	userContext[userID] = UserSelection{Asset: asset, Amount: amount}
}

func GetUserSelection(userID int64) (UserSelection, bool) {
	s, ok := userContext[userID]
	return s, ok
}

func DeleteUserContext(userID int64) {
	delete(userContext, userID)
}

func SaveToHistory(userID int64, pair string) {
	mutex.Lock()
	defer mutex.Unlock()

	h := userHistory[userID]
	newHistory := []string{pair}
	for _, p := range h {
		if p != pair {
			newHistory = append(newHistory, p)
		}
	}
	if len(newHistory) > maxHistorySize {
		newHistory = newHistory[:maxHistorySize]
	}
	userHistory[userID] = newHistory
	saveHistoryToFile()
}

func GetHistoryKeyboard(userID int64) tgbotapi.InlineKeyboardMarkup {
	h, ok := userHistory[userID]
	if !ok || len(h) == 0 {
		return tgbotapi.InlineKeyboardMarkup{}
	}
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range h {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(p, "history_req:"+p),
		))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func saveHistoryToFile() {
	data, _ := json.MarshalIndent(userHistory, "", "  ")
	_ = os.WriteFile(historyFile, data, 0644)
}

func LoadHistoryFromFile() {
	data, err := os.ReadFile(historyFile)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &userHistory)
}
