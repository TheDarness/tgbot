package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type RateResponse struct {
	AssetIDBase  *string  `json:"asset_id_base"`
	AssetIDQuote *string  `json:"asset_id_quote"`
	Rate         *float64 `json:"rate"`
	Time         *string  `json:"time"`
}

type OHLCVResponse struct {
	RateHigh float64 `json:"rate_high"`
	RateLow  float64 `json:"rate_low"`
}

type CoinStats struct {
	Rate       float64
	Time       string
	Min24h     float64
	Max24h     float64
	Aggregator string
}

func GetCryptoRate(apiKey, base, quote string) (*RateResponse, error) {
	url := fmt.Sprintf("https://rest.coinapi.io/v1/exchangerate/%s/%s", strings.ToUpper(base), strings.ToUpper(quote))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("помилка створення запиту: %v", err)
	}

	req.Header.Add("X-CoinAPI-Key", apiKey)

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("помилка мережі при запиті до API: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var result RateResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("помилка обробки json: %v", err)
		}
		if result.AssetIDBase == nil || result.AssetIDQuote == nil || result.Rate == nil {
			return nil, fmt.Errorf("API повернуло неповні дані курсу")
		}
		return &result, nil

	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, fmt.Errorf("помилка авторизації (Код: %d). Перевірте API-ключ", resp.StatusCode)

	case http.StatusNotFound, http.StatusBadRequest:
		var apiError map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&apiError); err == nil {
			if errMsg, ok := apiError["error"].(string); ok {
				return nil, fmt.Errorf("помилка в даних (Код: %d). %s", resp.StatusCode, errMsg)
			}
		}
		return nil, fmt.Errorf("помилка в даних (Код: %d). Перевірте код крипто/валюти", resp.StatusCode)

	default:
		return nil, fmt.Errorf("непередбачена помилка API (Код: %d)", resp.StatusCode)
	}
}

func GetCoinStats(apiKey, base, quote string) (*CoinStats, error) {
	rateResult, err := GetCryptoRate(apiKey, base, quote)
	if err != nil {
		return nil, err
	}
	ohlcvURL := fmt.Sprintf("https://rest.coinapi.io/v1/ohlcv/%s/%s/latest?period_id=1DAY&limit=1", strings.ToUpper(base), strings.ToUpper(quote))

	ohlcvReq, _ := http.NewRequest("GET", ohlcvURL, nil)
	ohlcvReq.Header.Add("X-CoinAPI-Key", apiKey)

	client := http.Client{Timeout: 10 * time.Second}
	ohlcvResp, err := client.Do(ohlcvReq)
	if err != nil {
		return &CoinStats{
			Rate:       *rateResult.Rate,
			Time:       *rateResult.Time,
			Min24h:     0.0,
			Max24h:     0.0,
			Aggregator: "CoinAPI Global",
		}, fmt.Errorf("помилка мережі при запиті OHLCV статистики: %w", err)
	}
	defer ohlcvResp.Body.Close()

	var ohlcvResults []OHLCVResponse
	if ohlcvResp.StatusCode != http.StatusOK || json.NewDecoder(ohlcvResp.Body).Decode(&ohlcvResults) != nil || len(ohlcvResults) == 0 {
		return &CoinStats{
			Rate:       *rateResult.Rate,
			Time:       *rateResult.Time,
			Min24h:     0.0,
			Max24h:     0.0,
			Aggregator: "CoinAPI Global",
		}, nil
	}

	stats := &CoinStats{
		Rate:       *rateResult.Rate,
		Time:       *rateResult.Time,
		Min24h:     ohlcvResults[0].RateLow,
		Max24h:     ohlcvResults[0].RateHigh,
		Aggregator: "CoinAPI Aggregated (OHLCV)",
	}

	return stats, nil
}
