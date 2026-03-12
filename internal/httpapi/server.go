package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

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
		protected.Post("/dashboard/reset", s.handleResetAllProgress)
		protected.Get("/kpis", s.handleListKPIs)
		protected.Get("/kpis/hierarchy", s.handleListKPIsWithHierarchy)
		protected.Post("/kpis", s.handleCreateKPI)
		protected.Get("/kpis/{id}/children", s.handleGetKPIChildren)
		protected.Post("/kpis/{id}/subkpis", s.handleCreateSubKPI)
		protected.Put("/kpis/{id}", s.handleUpdateKPI)
		protected.Put("/kpis/{id}/progress", s.handleUpdateKPIProgress)
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
		log.Printf("auth.register.failed email=%q ip=%q error=%q", input.Email, clientIP(r), err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	log.Printf("auth.register.succeeded user_id=%d email=%q ip=%q", user.ID, user.Email, clientIP(r))
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
	limiterKey := strings.ToLower(strings.TrimSpace(input.Email)) + "|" + clientIP(r)
	user, token, err := s.authService.Login(r.Context(), input.Email, input.Password, limiterKey)
	if err != nil {
		log.Printf("auth.login.failed email=%q ip=%q error=%q", input.Email, clientIP(r), err)
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	log.Printf("auth.login.succeeded user_id=%d email=%q ip=%q", user.ID, user.Email, clientIP(r))
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
	userID := auth.UserIDFromContext(r.Context())
	quarter := r.URL.Query().Get("quarter")
	log.Printf("dashboard.view user_id=%d quarter=%q", userID, quarter)
	summary, err := s.dashboardService.GetSummary(r.Context(), userID, quarter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleResetAllProgress(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Confirm-Action") != "reset-progress" {
		writeError(w, http.StatusForbidden, errors.New("missing reset confirmation header"))
		return
	}

	var input struct {
		Password     string `json:"password"`
		Confirmation string `json:"confirmation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if input.Confirmation != "RESET" {
		writeError(w, http.StatusBadRequest, errors.New(`confirmation must be "RESET"`))
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	if err := s.authService.VerifyPassword(r.Context(), userID, input.Password); err != nil {
		log.Printf("dashboard.reset.denied user_id=%d error=%q", userID, err)
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	result, err := s.dashboardService.ResetAllProgress(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	log.Printf(
		"dashboard.reset.completed user_id=%d kpis_updated=%d achievements_deleted=%d reports_deleted=%d jobs_deleted=%d",
		userID,
		result.KPIsUpdated,
		result.AchievementsDeleted,
		result.ReportsDeleted,
		result.JobsDeleted,
	)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleListKPIs(w http.ResponseWriter, r *http.Request) {
	items, err := s.kpiService.List(r.Context(), auth.UserIDFromContext(r.Context()), r.URL.Query().Get("quarter"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleListKPIsWithHierarchy(w http.ResponseWriter, r *http.Request) {
	items, err := s.kpiService.ListWithHierarchy(r.Context(), auth.UserIDFromContext(r.Context()), r.URL.Query().Get("quarter"))
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

func (s *Server) handleGetKPIChildren(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	items, err := s.kpiService.GetChildren(r.Context(), auth.UserIDFromContext(r.Context()), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateSubKPI(w http.ResponseWriter, r *http.Request) {
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

	item, err := s.kpiService.CreateSubKPI(r.Context(), auth.UserIDFromContext(r.Context()), id, input)
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

func (s *Server) handleUpdateKPIProgress(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var input struct {
		ProgressQ1 *int `json:"progressQ1"`
		ProgressQ2 *int `json:"progressQ2"`
		ProgressQ3 *int `json:"progressQ3"`
		ProgressQ4 *int `json:"progressQ4"`

		Quarter  string `json:"quarter"`
		Progress *int   `json:"progress"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	// Backward-compatible payload: { "quarter": "Q1", "progress": 50 }.
	if input.Progress != nil && input.Quarter != "" {
		switch input.Quarter {
		case "Q1", "q1":
			input.ProgressQ1 = input.Progress
		case "Q2", "q2":
			input.ProgressQ2 = input.Progress
		case "Q3", "q3":
			input.ProgressQ3 = input.Progress
		case "Q4", "q4":
			input.ProgressQ4 = input.Progress
		default:
			writeError(w, http.StatusBadRequest, errors.New("quarter must be Q1, Q2, Q3, or Q4"))
			return
		}
	}

	item, err := s.kpiService.UpdateProgress(r.Context(), auth.UserIDFromContext(r.Context()), id, kpi.ProgressUpdate{
		ProgressQ1: input.ProgressQ1,
		ProgressQ2: input.ProgressQ2,
		ProgressQ3: input.ProgressQ3,
		ProgressQ4: input.ProgressQ4,
	})
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
	userID := auth.UserIDFromContext(r.Context())
	quarter := chi.URLParam(r, "quarter")
	log.Printf("report.view user_id=%d quarter=%q", userID, quarter)
	report, err := s.reportService.Get(r.Context(), userID, quarter)
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

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host := r.RemoteAddr
	if addr, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
		host = addr.Addr().String()
	}
	return host
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
