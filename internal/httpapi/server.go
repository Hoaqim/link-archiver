package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/Hoaqim/link-archiver/internal/queue"
)

type Server struct {
	Logger *slog.Logger
	Queue  queue.Queue
}

func (s *Server) Handler() http.Handler {
	return s.routes()
}
