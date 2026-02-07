package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// ServiceRoutes maps service names to their local ports
var ServiceRoutes = map[string]string{
	"market":   "9080",
	"telegram": "10080",
}

// Server timeout constants
const (
	ReadTimeoutSeconds  = 10
	WriteTimeoutSeconds = 10
	IdleTimeoutSeconds  = 120
)

// For dev purposes only.
// ----------------------------------------------------------------------------------
// This proxy server acts as a unified gateway for all microservices in the platform.
// It routes requests to the appropriate service based on the URL path.
//
// The proxy supports the following routing pattern:
// - /api/v1/{service}/* routes to the corresponding service on its designated port
// - 8080 is the default port for the proxy server
//
// For example:
// - /api/v1/market/listings routes to the market service on port 9080
// - /api/v1/bot/updates routes to the bot service on port 10080
//
// This simplifies client integration by providing a single endpoint for all API calls
// while maintaining separation of services in the backend architecture.
func main() {
	// Create a single multiplexer
	mux := http.NewServeMux()

	// Register routes for each service with service prefix
	for service, port := range ServiceRoutes {
		targetURL, err := url.Parse(fmt.Sprintf("http://localhost:%s", port))
		if err != nil {
			log.Fatalf("Error parsing target URL for %s: %v", service, err)
		}

		// Create a reverse proxy for this service
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// Handle /api/v1/{service} path format
		apiPath := fmt.Sprintf("/api/v1/%s", service)
		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			// Get the full original path
			originalPath := r.URL.Path

			// Strip the API service prefix (/api/v1/{service}/)
			pathSuffix := strings.TrimPrefix(r.URL.Path, apiPath)
			r.URL.Path = apiPath + pathSuffix

			log.Printf("Proxying API request: %s -> %s%s", originalPath, targetURL, r.URL.Path)
			proxy.ServeHTTP(w, r)
		}

		mux.HandleFunc(apiPath, handlerFunc)
		mux.HandleFunc(apiPath+"/", handlerFunc)

		log.Printf("Registered API route: %s -> localhost:%s", apiPath, port)
	}

	// Start the server on a single port
	proxyPort := "8080"
	log.Printf("Starting reverse proxy on port %s", proxyPort)

	// Create server with timeouts
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", proxyPort),
		Handler:      mux,
		ReadTimeout:  ReadTimeoutSeconds * time.Second,
		WriteTimeout: WriteTimeoutSeconds * time.Second,
		IdleTimeout:  IdleTimeoutSeconds * time.Second,
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Error starting proxy server: %v", err)
	}
}
