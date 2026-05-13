package httpapi

import "net/http"

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.Health)
	mux.HandleFunc("GET /ready", s.Ready)

	mux.HandleFunc("POST /jobs", s.CreateJob)
	return mux
}
