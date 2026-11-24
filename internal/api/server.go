package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/anandmishra1/cloudforge/internal/config"
	"github.com/anandmishra1/cloudforge/internal/detect"
	"github.com/anandmishra1/cloudforge/internal/generate"
	"github.com/anandmishra1/cloudforge/internal/k8s"
)

type Server struct {
	logger      *slog.Logger
	kube        *k8s.Client
	mu          sync.RWMutex
	deployments map[string]*Deployment
}

type Deployment struct {
	ID         string    `json:"id"`
	App        string    `json:"app"`
	Repository string    `json:"repository"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"createdAt"`
	Logs       []string  `json:"-"`
}

type deployRequest struct {
	Repository string `json:"repository"`
	Name       string `json:"name"`
	OutputDir  string `json:"outputDir"`
}

func NewServer(logger *slog.Logger, kube *k8s.Client) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{logger: logger, kube: kube, deployments: make(map[string]*Deployment)}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/v1/deployments", s.listDeployments)
	mux.HandleFunc("POST /api/v1/deployments", s.createDeployment)
	mux.HandleFunc("GET /api/v1/deployments/{id}/logs", s.logs)
	mux.HandleFunc("GET /api/v1/apps/{app}/status", s.status)
	mux.HandleFunc("POST /api/v1/apps/{app}/rollback", s.rollback)
	return loggingMiddleware(s.logger, mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) listDeployments(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*Deployment, 0, len(s.deployments))
	for _, deployment := range s.deployments {
		copy := *deployment
		copy.Logs = nil
		items = append(items, &copy)
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createDeployment(w http.ResponseWriter, r *http.Request) {
	var request deployRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if request.Repository == "" {
		writeError(w, http.StatusBadRequest, "repository is required")
		return
	}
	root, err := config.RepoPath(request.Repository)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	detection, err := detect.Detect(root)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	app := config.DefaultApp(root, request.Name)
	id := fmt.Sprintf("%s-%d", app.Name, time.Now().UnixNano())
	deployment := &Deployment{ID: id, App: app.Name, Repository: request.Repository, Status: "generating", Message: "detected " + string(detection.Stack), CreatedAt: time.Now().UTC()}
	s.mu.Lock()
	s.deployments[id] = deployment
	s.mu.Unlock()
	go s.generate(id, app, detection, request.OutputDir)
	writeJSON(w, http.StatusAccepted, deployment)
}

func (s *Server) generate(id string, app config.App, detection detect.Result, output string) {
	if output == "" {
		output = ".cloudforge/generated/" + app.Name
	}
	files, err := generate.Generate(generate.Options{App: app, Detection: detection, OutputDir: output})
	s.mu.Lock()
	defer s.mu.Unlock()
	deployment := s.deployments[id]
	if err != nil {
		deployment.Status = "failed"
		deployment.Message = err.Error()
		deployment.Logs = append(deployment.Logs, err.Error())
		return
	}
	deployment.Status = "generated"
	deployment.Message = fmt.Sprintf("generated %d files", len(files))
	deployment.Logs = append(deployment.Logs, deployment.Message)
}

func (s *Server) logs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	deployment, ok := s.deployments[id]
	logs := []string(nil)
	if ok {
		logs = append(logs, deployment.Logs...)
	}
	s.mu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, canFlush := w.(http.Flusher)
	offset := 0
	for {
		for _, line := range logs[offset:] {
			fmt.Fprintf(w, "data: %s\n\n", strings.ReplaceAll(line, "\n", " "))
		}
		offset = len(logs)
		if canFlush {
			flusher.Flush()
		}
		if deploymentStatus(s, id) != "generating" {
			break
		}
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-r.Context().Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		s.mu.RLock()
		logs = append([]string(nil), s.deployments[id].Logs...)
		s.mu.RUnlock()
	}
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	if s.kube == nil {
		writeError(w, http.StatusServiceUnavailable, "Kubernetes client is not configured")
		return
	}
	status, err := s.kube.Status(r.Context(), r.PathValue("app"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) rollback(w http.ResponseWriter, r *http.Request) {
	if s.kube == nil {
		writeError(w, http.StatusServiceUnavailable, "Kubernetes client is not configured")
		return
	}
	deployment, err := s.kube.Rollback(r.Context(), r.PathValue("app"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"app": deployment.Name, "message": "rollback applied"})
}

func deploymentStatus(s *Server, id string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if deployment, ok := s.deployments[id]; ok {
		return deployment.Status
	}
	return "failed"
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started).String())
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
