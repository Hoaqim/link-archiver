package httpapi

import (
	"log/slog"
	"net/http"
)

type Server struct {
	Logger *slog.Logger
}

func (s *Server) Handler() http.Handler {
	return s.routes()
}
