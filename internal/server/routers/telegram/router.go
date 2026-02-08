package telegram

import (
	"net/http"

	"ads-mrkt/internal/server"
)

type telegramMiddleware interface {
	TelegramMiddleware(next http.HandlerFunc) http.HandlerFunc
}

type webhookHandler interface {
	HandleUpdate(w http.ResponseWriter, r *http.Request)
}

type Router struct {
	webhookHandler     webhookHandler
	telegramMiddleware telegramMiddleware
}

func NewRouter(webhookHandler webhookHandler, telegramMiddleware telegramMiddleware) *Router {
	return &Router{
		webhookHandler:     webhookHandler,
		telegramMiddleware: telegramMiddleware,
	}
}

func (r *Router) GetRoutes() http.Handler {
	mux := http.NewServeMux()

	// Telegram updates webhook
	mux.HandleFunc("POST /api/v1/telegram/webhook", server.WithMetrics(
		r.telegramMiddleware.TelegramMiddleware(
			server.WithMethod(
				r.webhookHandler.HandleUpdate,
				http.MethodPost,
			),
		),
		"/api/v1",
	))

	return mux
}
