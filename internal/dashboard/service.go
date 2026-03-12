package dashboard

import (
	"context"
	"database/sql"
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
