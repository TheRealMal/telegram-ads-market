package server

import (
	"net/http"
)

func (s *Server) HealthHandler(w http.ResponseWriter, _ *http.Request) (interface{}, error) {
	status := s.healthChecker.GetHealthStatus()

	if !status.Healthy {
		w.WriteHeader(http.StatusServiceUnavailable)
		return status, nil
	}

	return status, nil
}
