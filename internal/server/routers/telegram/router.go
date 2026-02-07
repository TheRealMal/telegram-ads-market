package telegram

import (
	"net/http"

	"ads-mrkt/internal/server"
	serverconfig "ads-mrkt/internal/server/config"
)

type telegramMiddleware interface {
	TelegramMiddleware(next http.HandlerFunc) http.HandlerFunc
}

type webhookHandler interface {
	HandleUpdate(w http.ResponseWriter, r *http.Request)
}

type Router struct {
	Config serverconfig.Config

	webhookHandler     webhookHandler
	telegramMiddleware telegramMiddleware
}

func NewRouter(config serverconfig.Config, webhookHandler webhookHandler, telegramMiddleware telegramMiddleware) *Router {
	return &Router{
		Config:             config,
		webhookHandler:     webhookHandler,
		telegramMiddleware: telegramMiddleware,
	}
}

func (r *Router) GetRoutes() http.Handler {
	corsConfig := server.CORSConfig{
		AllowOrigin:  []string{r.Config.ClientDomain},
		AllowMethods: []string{http.MethodPost},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}

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

	return server.MuxWithCORS(mux, &corsConfig)
}
