package dashboard

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/example/kpi-chaser/internal/storage"
)

func setupDashboardService(t *testing.T) (*Service, int64) {
	t.Helper()

	db, err := storage.Open(filepath.Join(t.TempDir(), "dashboard-test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := storage.Migrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	result, err := db.Exec(`INSERT INTO users (email, password_hash) VALUES (?, ?)`, "owner@example.com", "hash")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}

	if _, err = db.Exec(`
		INSERT INTO kpis (user_id, quarter, title, progress_q1, progress_q2, progress_q3, progress_q4, annual_progress)
		VALUES (?, '2026-Q1', 'Quarter KPI', 30, 40, 50, 60, 45)
	`, userID); err != nil {
		t.Fatalf("seed kpis: %v", err)
	}
	if _, err = db.Exec(`INSERT INTO achievements (user_id, quarter, raw_text, status) VALUES (?, '2026-Q1', 'Shipped milestone', 'draft')`, userID); err != nil {
		t.Fatalf("seed achievements: %v", err)
	}
	if _, err = db.Exec(`INSERT INTO reports (user_id, quarter, title, body) VALUES (?, '2026-Q1', 'Quarter report', 'body')`, userID); err != nil {
		t.Fatalf("seed reports: %v", err)
	}
	if _, err = db.Exec(`INSERT INTO jobs (user_id, type, status, payload) VALUES (?, 'enhance-achievement', 'pending', '{}')`, userID); err != nil {
		t.Fatalf("seed dashboard data: %v", err)
	}

	return NewService(db), userID
}

func TestResetAllProgress(t *testing.T) {
	ctx := context.Background()
	svc, userID := setupDashboardService(t)

	result, err := svc.ResetAllProgress(ctx, userID)
	if err != nil {
		t.Fatalf("reset all progress: %v", err)
	}

	if result.KPIsUpdated != 1 || result.AchievementsDeleted != 1 || result.ReportsDeleted != 1 || result.JobsDeleted != 1 {
		t.Fatalf("unexpected reset result: %+v", result)
	}

	summary, err := svc.GetSummary(ctx, userID, "2026-Q1")
	if err != nil {
		t.Fatalf("get summary after reset: %v", err)
	}
	if summary.TotalAchievements != 0 || summary.PendingJobs != 0 {
		t.Fatalf("expected cleared achievements and jobs, got %+v", summary)
	}
	if len(summary.KPIProgress) != 1 || summary.KPIProgress[0].ProgressPercent != 0 {
		t.Fatalf("expected reset KPI progress, got %+v", summary.KPIProgress)
	}
}
