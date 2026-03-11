package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/example/kpi-chaser/internal/achievements"
	"github.com/example/kpi-chaser/internal/auth"
	"github.com/example/kpi-chaser/internal/config"
	"github.com/example/kpi-chaser/internal/dashboard"
	"github.com/example/kpi-chaser/internal/jobs"
	"github.com/example/kpi-chaser/internal/kpi"
	"github.com/example/kpi-chaser/internal/reports"
)

type Server struct {
	authService        *auth.Service
	dashboardService   *dashboard.Service
	kpiService         *kpi.Service
	achievementService *achievements.Service
	reportService      *reports.Service
}

func NewServer(cfg config.Config, db *sql.DB) (*Server, error) {
	queue := jobs.NewQueue(db)
	return &Server{
		authService:        auth.NewService(db, cfg.JWTSecret),
		dashboardService:   dashboard.NewService(db),
		kpiService:         kpi.NewService(db),
		achievementService: achievements.NewService(db, queue),
		reportService:      reports.NewService(db),
	}, nil
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Post("/auth/register", s.handleRegister)
	r.Post("/auth/login", s.handleLogin)

	r.Group(func(protected chi.Router) {
		protected.Use(s.authService.Middleware)
		protected.Get("/me", s.handleMe)
		protected.Get("/dashboard", s.handleDashboard)
		protected.Get("/kpis", s.handleListKPIs)
		protected.Post("/kpis", s.handleCreateKPI)
		protected.Put("/kpis/{id}", s.handleUpdateKPI)
		protected.Delete("/kpis/{id}", s.handleDeleteKPI)
		protected.Get("/achievements", s.handleListAchievements)
		protected.Post("/achievements", s.handleCreateAchievement)
		protected.Put("/achievements/{id}", s.handleUpdateAchievement)
		protected.Delete("/achievements/{id}", s.handleDeleteAchievement)
		protected.Post("/achievements/{id}/enhance", s.handleEnhanceAchievement)
		protected.Post("/reports/generate", s.handleGenerateReport)
		protected.Get("/reports/{quarter}", s.handleGetReport)
	})

	return cors(r)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, token, err := s.authService.Register(r.Context(), input.Email, input.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": user, "token": token})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, token, err := s.authService.Login(r.Context(), input.Email, input.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "token": token})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := s.authService.GetUser(r.Context(), auth.UserIDFromContext(r.Context()))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	summary, err := s.dashboardService.GetSummary(r.Context(), auth.UserIDFromContext(r.Context()), r.URL.Query().Get("quarter"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleListKPIs(w http.ResponseWriter, r *http.Request) {
	items, err := s.kpiService.List(r.Context(), auth.UserIDFromContext(r.Context()), r.URL.Query().Get("quarter"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateKPI(w http.ResponseWriter, r *http.Request) {
	var input kpi.KPI
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input.UserID = auth.UserIDFromContext(r.Context())
	item, err := s.kpiService.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleUpdateKPI(w http.ResponseWriter, r *http.Request) {
	var input kpi.KPI
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input.ID = id
	input.UserID = auth.UserIDFromContext(r.Context())
	item, err := s.kpiService.Update(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleDeleteKPI(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.kpiService.Delete(r.Context(), auth.UserIDFromContext(r.Context()), id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListAchievements(w http.ResponseWriter, r *http.Request) {
	items, err := s.achievementService.List(r.Context(), auth.UserIDFromContext(r.Context()), r.URL.Query().Get("quarter"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateAchievement(w http.ResponseWriter, r *http.Request) {
	var input achievements.Achievement
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input.UserID = auth.UserIDFromContext(r.Context())
	item, err := s.achievementService.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleUpdateAchievement(w http.ResponseWriter, r *http.Request) {
	var input achievements.Achievement
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input.ID = id
	input.UserID = auth.UserIDFromContext(r.Context())
	item, err := s.achievementService.Update(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleDeleteAchievement(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.achievementService.Delete(r.Context(), auth.UserIDFromContext(r.Context()), id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleEnhanceAchievement(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	err = s.achievementService.EnqueueEnhancement(r.Context(), auth.UserIDFromContext(r.Context()), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

func (s *Server) handleGenerateReport(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Quarter string `json:"quarter"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	report, err := s.reportService.Generate(r.Context(), auth.UserIDFromContext(r.Context()), input.Quarter)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	report, err := s.reportService.Get(r.Context(), auth.UserIDFromContext(r.Context()), chi.URLParam(r, "quarter"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
