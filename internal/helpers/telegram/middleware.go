package telegram

import "net/http"

func (c *APIClient) TelegramMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != c.secretToken {
			http.Error(w, "invalid secret token", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}
