package httpapi

import "net/http"

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.Health)
	mux.HandleFunc("GET /ready", s.Ready)

	mux.HandleFunc("POST /jobs", s.CreateJob)
	mux.HandleFunc("GET /jobs{id}", s.GetJob)
	mux.HandleFunc("GET /jobs/{id}/status", s.JobStatus)
	return mux
}
