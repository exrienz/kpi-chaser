package dashboard

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type KPIProgress struct {
	KPIID            int64  `json:"kpiId"`
	Title            string `json:"title"`
	Weight           int    `json:"weight"`
	AchievementCount int    `json:"achievementCount"`
	EnhancedCount    int    `json:"enhancedCount"`
	ProgressPercent  int    `json:"progressPercent"`
}

type Summary struct {
	Quarter              string        `json:"quarter"`
	TotalKPIs            int           `json:"totalKpis"`
	TotalAchievements    int           `json:"totalAchievements"`
	EnhancedAchievements int           `json:"enhancedAchievements"`
	MappedAchievements   int           `json:"mappedAchievements"`
	DraftAchievements    int           `json:"draftAchievements"`
	PendingJobs          int           `json:"pendingJobs"`
	KPIProgress          []KPIProgress `json:"kpiProgress"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) GetSummary(ctx context.Context, userID int64, quarter string) (Summary, error) {
	summary := Summary{Quarter: quarter}

	err := s.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM kpis WHERE user_id = ? AND quarter = ?),
			(SELECT COUNT(*) FROM achievements WHERE user_id = ? AND quarter = ?),
			(SELECT COUNT(*) FROM achievements WHERE user_id = ? AND quarter = ? AND enhanced_text <> ''),
			(SELECT COUNT(*) FROM achievements WHERE user_id = ? AND quarter = ? AND kpi_id IS NOT NULL),
			(SELECT COUNT(*) FROM achievements WHERE user_id = ? AND quarter = ? AND status = 'draft'),
			(SELECT COUNT(*) FROM jobs WHERE user_id = ? AND status IN ('pending', 'processing'))
	`, userID, quarter, userID, quarter, userID, quarter, userID, quarter, userID, quarter, userID).
		Scan(&summary.TotalKPIs, &summary.TotalAchievements, &summary.EnhancedAchievements, &summary.MappedAchievements, &summary.DraftAchievements, &summary.PendingJobs)
	if err != nil {
		return Summary{}, fmt.Errorf("load dashboard summary: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			k.id,
			k.title,
			k.weight,
			k.annual_progress,
			COUNT(a.id) AS achievement_count,
			SUM(CASE WHEN a.enhanced_text <> '' THEN 1 ELSE 0 END) AS enhanced_count
		FROM kpis k
		LEFT JOIN achievements a ON a.kpi_id = k.id AND a.user_id = k.user_id AND a.quarter = k.quarter
		WHERE k.user_id = ? AND k.quarter = ?
		GROUP BY k.id, k.title, k.weight, k.annual_progress
		ORDER BY k.weight DESC, k.created_at ASC
	`, userID, quarter)
	if err != nil {
		return Summary{}, fmt.Errorf("load kpi progress: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item KPIProgress
		var annualProgress int
		if err := rows.Scan(&item.KPIID, &item.Title, &item.Weight, &annualProgress, &item.AchievementCount, &item.EnhancedCount); err != nil {
			return Summary{}, err
		}
		if annualProgress < 0 {
			annualProgress = 0
		}
		if annualProgress > 100 {
			annualProgress = 100
		}
		item.ProgressPercent = annualProgress
		summary.KPIProgress = append(summary.KPIProgress, item)
	}

	return summary, rows.Err()
}

type ResetResult struct {
	KPIsUpdated         int64 `json:"kpisUpdated"`
	AchievementsDeleted int64 `json:"achievementsDeleted"`
	ReportsDeleted      int64 `json:"reportsDeleted"`
	JobsDeleted         int64 `json:"jobsDeleted"`
}

func (s *Service) ResetAllProgress(ctx context.Context, userID int64) (ResetResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ResetResult{}, fmt.Errorf("begin reset transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	result := ResetResult{}
	if result.KPIsUpdated, err = execRowsAffected(ctx, tx, `
		UPDATE kpis
		SET progress_q1 = 0,
		    progress_q2 = 0,
		    progress_q3 = 0,
		    progress_q4 = 0,
		    annual_progress = 0,
		    updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, userID); err != nil {
		return ResetResult{}, fmt.Errorf("reset kpi progress: %w", err)
	}

	if result.AchievementsDeleted, err = execRowsAffected(ctx, tx, `DELETE FROM achievements WHERE user_id = ?`, userID); err != nil {
		return ResetResult{}, fmt.Errorf("delete achievements: %w", err)
	}
	if result.ReportsDeleted, err = execRowsAffected(ctx, tx, `DELETE FROM reports WHERE user_id = ?`, userID); err != nil {
		return ResetResult{}, fmt.Errorf("delete reports: %w", err)
	}
	if result.JobsDeleted, err = execRowsAffected(ctx, tx, `DELETE FROM jobs WHERE user_id = ?`, userID); err != nil {
		return ResetResult{}, fmt.Errorf("delete jobs: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return ResetResult{}, fmt.Errorf("commit reset transaction: %w", err)
	}
	return result, nil
}

func execRowsAffected(ctx context.Context, tx *sql.Tx, query string, args ...any) (int64, error) {
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	return affected, nil
}
