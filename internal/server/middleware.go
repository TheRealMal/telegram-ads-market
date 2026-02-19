package server

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"ads-mrkt/internal/errors"
	"ads-mrkt/internal/server/templates/response"
)

type CORSConfig struct {
	AllowOrigin  []string
	AllowMethods []string
	AllowHeaders []string
}

// MuxWithCORS is a middleware that sets CORS headers
func MuxWithCORS(next http.Handler, config *CORSConfig) http.Handler {
	allowedMethods := strings.Join(config.AllowMethods, ", ")
	allowedHeaders := strings.Join(config.AllowHeaders, ", ")

	var allowedOrigin string
	if len(config.AllowOrigin) == 0 {
		allowedOrigin = "*"
	} else {
		allowedOrigin = config.AllowOrigin[0] // First value is the default origin
	}

	allowedOrigins := map[string]struct{}{}
	for _, origin := range config.AllowOrigin {
		allowedOrigins[origin] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := allowedOrigins[r.Header.Get("Origin")]; ok {
			w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		} else {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}
		w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// WithMethod is a middleware that checks if the endpoint was called using a
// specific HTTP method and rejects it otherwise.
func WithMethod(next http.HandlerFunc, method string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, fmt.Sprintf("Only %s method is allowed", method), http.StatusMethodNotAllowed)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// WithJSONResponse wraps an APIHandler and handles JSON response formatting
func WithJSONResponse(handler APIHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Call the handler to get data or error
		data, err := handler(w, r)

		// Set the Content-Type header
		w.Header().Set("Content-Type", "application/json")

		if err != nil {
			var (
				errCode         string
				errorData       interface{}
				serviceError    errors.ServiceError
				serviceErrorPtr *errors.ServiceError
			)
			switch {
			case stderrors.As(err, &serviceError):
				errCode = string(serviceError.Code)
				errorData = serviceError.Data
				slog.Error("ServiceError", "error", serviceError, "stack", serviceError.Err)
			case stderrors.As(err, &serviceErrorPtr):
				errCode = string(serviceErrorPtr.Code)
				errorData = serviceErrorPtr.Data
				slog.Error("ServiceError", "error", serviceErrorPtr, "stack", serviceErrorPtr.Err)
			default:
				errCode = err.Error()
				slog.Error("UnknownError", "error", err)
			}

			errorResponse := &response.Template{
				Ok:        false,
				ErrorCode: errCode,
				Data:      errorData,
			}

			// Encode and send the error response
			if err := json.NewEncoder(w).Encode(*errorResponse); err != nil {
				http.Error(w, `{"ok": false, "error_code": "internal_error"}`, http.StatusInternalServerError)
			}
			return
		}

		// Create the success response
		successResponse := response.Template{
			Ok:   true,
			Data: data,
		}

		// Encode and send the success response
		if err := json.NewEncoder(w).Encode(successResponse); err != nil {
			http.Error(w, `{"ok": false, "error_code": "internal_error"}`, http.StatusInternalServerError)
			return
		}
	}
}
