package httpapi

import "net/http"

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /Health", s.Health)
	mux.HandleFunc("GET /Ready", s.Ready)

	return mux
}
