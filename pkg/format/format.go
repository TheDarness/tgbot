package format

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func FormatNumber(num float64, precision int) string {
	formatStr := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(formatStr, num)
}

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
