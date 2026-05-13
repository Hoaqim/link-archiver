package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/Hoaqim/link-archiver/internal/queue"
)

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	//TODO: actual checks
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func (s *Server) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req queue.Req

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "URL required", http.StatusBadRequest)
		return
	}

	var job queue.Job
	payload, err := json.Marshal(job)
	if err != nil {
		s.Logger.Error("Marshal job error", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := s.Queue.Enqueue(r.Context(), payload); err != nil {
		s.Logger.Error("Enqueue error", "err", err)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	s.Logger.Info("Job enqueued", "id", job.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"id": job.ID})
}
