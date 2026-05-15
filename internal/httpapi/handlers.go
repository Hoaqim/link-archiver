package httpapi

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"

	"github.com/Hoaqim/link-archiver/internal/queue"
	"github.com/google/uuid"
)

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	//TODO: actual checks
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
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

	job := queue.Job{
		ID:  uuid.NewString(),
		URL: req.URL,
	}
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
	writeJSON(w, http.StatusAccepted, map[string]string{"id": job.ID})

}

func (s *Server) GetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}

	data, ct, err := s.Storage.Get(r.Context(), id+".html")
	if err != nil {
		if isNotFound(err) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		s.Logger.Error("storage get", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) JobStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	ok, err := s.Storage.Exists(r.Context(), id+".html")
	if err != nil {
		s.Logger.Error("storage exists", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	status := "pending"
	if ok {
		status = "done"
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": id, "status": status})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func isNotFound(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}
