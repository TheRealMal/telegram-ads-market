package market

import (
	"net/http"

	"ads-mrkt/internal/server"
	serverconfig "ads-mrkt/internal/server/config"
)

type handler interface {
	AuthUser(w http.ResponseWriter, r *http.Request) (interface{}, error)
	CreateListing(w http.ResponseWriter, r *http.Request) (interface{}, error)
	GetListing(w http.ResponseWriter, r *http.Request) (interface{}, error)
	ListListings(w http.ResponseWriter, r *http.Request) (interface{}, error)
	ListMyListings(w http.ResponseWriter, r *http.Request) (interface{}, error)
	UpdateListing(w http.ResponseWriter, r *http.Request) (interface{}, error)
	DeleteListing(w http.ResponseWriter, r *http.Request) (interface{}, error)
	ListMyChannels(w http.ResponseWriter, r *http.Request) (interface{}, error)
	RefreshChannel(w http.ResponseWriter, r *http.Request) (interface{}, error)
	GetChannelStats(w http.ResponseWriter, r *http.Request) (interface{}, error)
	CreateDeal(w http.ResponseWriter, r *http.Request) (interface{}, error)
	GetDeal(w http.ResponseWriter, r *http.Request) (interface{}, error)
	ListDealsByListingID(w http.ResponseWriter, r *http.Request) (interface{}, error)
	ListMyDeals(w http.ResponseWriter, r *http.Request) (interface{}, error)
	UpdateDealDraft(w http.ResponseWriter, r *http.Request) (interface{}, error)
	SignDeal(w http.ResponseWriter, r *http.Request) (interface{}, error)
	SendDealChatMessage(w http.ResponseWriter, r *http.Request) (interface{}, error)
	ListDealMessages(w http.ResponseWriter, r *http.Request) (interface{}, error)
}

type authMiddleware interface {
	WithAuth(next http.HandlerFunc) http.HandlerFunc
}

type Router struct {
	Config serverconfig.Config

	handler        handler
	authMiddleware authMiddleware
}

func NewRouter(config serverconfig.Config, handler handler, authMiddleware authMiddleware) *Router {
	return &Router{
		Config:         config,
		handler:        handler,
		authMiddleware: authMiddleware,
	}
}

func (r *Router) GetRoutes() http.Handler {
	corsConfig := server.CORSConfig{
		AllowOrigin:  []string{r.Config.ClientDomain},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch},
		AllowHeaders: []string{"Content-Type", "Authorization", "X-Telegram-InitData"},
	}

	mux := http.NewServeMux()

	// Auth (no JWT required)
	mux.HandleFunc("POST /api/v1/market/auth", server.WithMetrics(
		server.WithMethod(
			server.WithJSONResponse(r.handler.AuthUser),
			http.MethodPost,
		),
		"/api/v1",
	))

	// Listings: all (public) and my (auth)
	mux.HandleFunc("GET /api/v1/market/listings", server.WithMetrics(
		server.WithMethod(
			server.WithJSONResponse(r.handler.ListListings),
			http.MethodGet,
		),
		"/api/v1",
	))
	mux.HandleFunc("GET /api/v1/market/my-listings", server.WithMetrics(
		r.authMiddleware.WithAuth(
			server.WithMethod(
				server.WithJSONResponse(r.handler.ListMyListings),
				http.MethodGet,
			),
		),
		"/api/v1",
	))
	mux.HandleFunc("POST /api/v1/market/listings", server.WithMetrics(
		r.authMiddleware.WithAuth(
			server.WithMethod(
				server.WithJSONResponse(r.handler.CreateListing),
				http.MethodPost,
			),
		),
		"/api/v1",
	))
	mux.HandleFunc("GET /api/v1/market/listings/{id}", server.WithMetrics(
		server.WithMethod(
			server.WithJSONResponse(r.handler.GetListing),
			http.MethodGet,
		),
		"/api/v1",
	))
	mux.HandleFunc("PATCH /api/v1/market/listings/{id}", server.WithMetrics(
		r.authMiddleware.WithAuth(
			server.WithMethod(
				server.WithJSONResponse(r.handler.UpdateListing),
				http.MethodPatch,
			),
		),
		"/api/v1",
	))
	mux.HandleFunc("DELETE /api/v1/market/listings/{id}", server.WithMetrics(
		r.authMiddleware.WithAuth(
			server.WithMethod(
				server.WithJSONResponse(r.handler.DeleteListing),
				http.MethodDelete,
			),
		),
		"/api/v1",
	))

	// Channels (auth required)
	mux.HandleFunc("GET /api/v1/market/my-channels", server.WithMetrics(
		r.authMiddleware.WithAuth(
			server.WithMethod(
				server.WithJSONResponse(r.handler.ListMyChannels),
				http.MethodGet,
			),
		),
		"/api/v1",
	))
	mux.HandleFunc("GET /api/v1/market/channels/{id}/refresh", server.WithMetrics(
		r.authMiddleware.WithAuth(
			server.WithMethod(
				server.WithJSONResponse(r.handler.RefreshChannel),
				http.MethodGet,
			),
		),
		"/api/v1",
	))
	mux.HandleFunc("GET /api/v1/market/channels/{id}/stats", server.WithMetrics(
		r.authMiddleware.WithAuth(
			server.WithMethod(
				server.WithJSONResponse(r.handler.GetChannelStats),
				http.MethodGet,
			),
		),
		"/api/v1",
	))

	// Deals
	mux.HandleFunc("POST /api/v1/market/deals", server.WithMetrics(
		r.authMiddleware.WithAuth(
			server.WithMethod(
				server.WithJSONResponse(r.handler.CreateDeal),
				http.MethodPost,
			),
		),
		"/api/v1",
	))
	mux.HandleFunc("GET /api/v1/market/deals/{id}", server.WithMetrics(
		server.WithMethod(
			server.WithJSONResponse(r.handler.GetDeal),
			http.MethodGet,
		),
		"/api/v1",
	))
	mux.HandleFunc("GET /api/v1/market/listings/{listing_id}/deals", server.WithMetrics(
		server.WithMethod(
			server.WithJSONResponse(r.handler.ListDealsByListingID),
			http.MethodGet,
		),
		"/api/v1",
	))
	mux.HandleFunc("GET /api/v1/market/my-deals", server.WithMetrics(
		r.authMiddleware.WithAuth(
			server.WithMethod(
				server.WithJSONResponse(r.handler.ListMyDeals),
				http.MethodGet,
			),
		),
		"/api/v1",
	))
	mux.HandleFunc("PATCH /api/v1/market/deals/{id}", server.WithMetrics(
		r.authMiddleware.WithAuth(
			server.WithMethod(
				server.WithJSONResponse(r.handler.UpdateDealDraft),
				http.MethodPatch,
			),
		),
		"/api/v1",
	))
	mux.HandleFunc("POST /api/v1/market/deals/{id}/sign", server.WithMetrics(
		r.authMiddleware.WithAuth(
			server.WithMethod(
				server.WithJSONResponse(r.handler.SignDeal),
				http.MethodPost,
			),
		),
		"/api/v1",
	))
	mux.HandleFunc("POST /api/v1/market/deals/{id}/send-chat-message", server.WithMetrics(
		r.authMiddleware.WithAuth(
			server.WithMethod(
				server.WithJSONResponse(r.handler.SendDealChatMessage),
				http.MethodPost,
			),
		),
		"/api/v1",
	))
	mux.HandleFunc("GET /api/v1/market/deals/{id}/messages", server.WithMetrics(
		r.authMiddleware.WithAuth(
			server.WithMethod(
				server.WithJSONResponse(r.handler.ListDealMessages),
				http.MethodGet,
			),
		),
		"/api/v1",
	))

	return server.MuxWithCORS(mux, &corsConfig)
}
