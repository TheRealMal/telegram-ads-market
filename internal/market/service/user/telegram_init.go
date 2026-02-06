package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// TelegramUser represents Telegram user data structure
type telegramUser struct {
	ID              int64  `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name,omitempty"`
	Username        string `json:"username,omitempty"`
	LanguageCode    string `json:"language_code,omitempty"`
	IsPremium       bool   `json:"is_premium,omitempty"`
	AllowsWriteToPM bool   `json:"allows_write_to_pm,omitempty"`
	PhotoURL        string `json:"photo_url,omitempty"`
}

// TelegramInitData represents parsed init data from Telegram Mini App
type telegramInitData struct {
	QueryID      string        `json:"query_id,omitempty"`
	User         *telegramUser `json:"user,omitempty"`
	AuthDate     int64         `json:"auth_date"`
	Hash         string        `json:"hash"`
	ChatInstance string        `json:"chat_instance,omitempty"`
	ChatType     string        `json:"chat_type,omitempty"`
	StartParam   string        `json:"start_param,omitempty"`
}

// parseAndVerifyInitData parses and verifies Telegram init data
// botToken is your Telegram bot's token
// initDataStr is the raw init data string from Telegram Mini App
func parseAndVerifyInitData(botToken, initDataStr string) (*telegramInitData, error) {
	// Parse the URL-encoded string
	values, err := url.ParseQuery(initDataStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse init data: %w", err)
	}

	initData, err := parseInitData(values)
	if err != nil {
		return nil, fmt.Errorf("failed to parse init data: %w", err)
	}

	// Create sorted array of key=value pairs excluding hash
	var dataCheckArr []string
	for key, vals := range values {
		if key == "hash" {
			continue
		}
		if len(vals) > 0 {
			dataCheckArr = append(dataCheckArr, fmt.Sprintf("%s=%s", key, vals[0]))
		}
	}
	sort.Strings(dataCheckArr)
	dataCheckStr := strings.Join(dataCheckArr, "\n")

	// Verify hash
	if !verifySignature(botToken, dataCheckStr, initData.Hash) {
		return nil, fmt.Errorf("invalid init data signature")
	}

	return initData, nil
}

func parseInitData(initDataValues url.Values) (*telegramInitData, error) {
	initData := &telegramInitData{
		Hash:         initDataValues.Get("hash"),
		QueryID:      initDataValues.Get("query_id"),
		User:         &telegramUser{},
		ChatType:     initDataValues.Get("chat_type"),
		ChatInstance: initDataValues.Get("chat_instance"),
		StartParam:   initDataValues.Get("start_param"),
	}
	if authDate := initDataValues.Get("auth_date"); authDate != "" {
		i, err := strconv.ParseInt(authDate, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse auth_date to int64: %w", err)
		}
		initData.AuthDate = i
	}
	if userData := initDataValues.Get("user"); userData != "" {
		if err := json.Unmarshal([]byte(userData), &initData.User); err != nil {
			return nil, fmt.Errorf("failed to parse user data: %w", err)
		}
	}
	return initData, nil
}

// verifySignature verifies the Telegram init data signature
func verifySignature(botToken, dataCheckStr, hash string) bool {
	// 1. Create HMAC-SHA256 with key "WebAppData"
	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))

	// 2. Create HMAC-SHA256 with the result from step 1
	dataKey := hmac.New(sha256.New, secretKey.Sum(nil))
	dataKey.Write([]byte(dataCheckStr))

	// 3. Compare with provided hash
	calculatedHash := hex.EncodeToString(dataKey.Sum(nil))
	return calculatedHash == hash
}
