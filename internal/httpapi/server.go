package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/Hoaqim/link-archiver/internal/queue"
	"github.com/Hoaqim/link-archiver/internal/storage"
)

type Server struct {
	Logger  *slog.Logger
	Queue   queue.Queue
	Storage storage.Storage
}

func (s *Server) Handler() http.Handler {
	return s.routes()
}
