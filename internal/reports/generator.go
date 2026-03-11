package reports

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"
)

type Report struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Quarter   string    `json:"quarter"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Generate(ctx context.Context, userID int64, quarter string) (Report, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(k.title, 'Unmapped'), a.enhanced_text, a.raw_text, COALESCE(a.impact_note, '')
		FROM achievements a
		LEFT JOIN kpis k ON a.kpi_id = k.id
		WHERE a.user_id = ? AND a.quarter = ?
		ORDER BY k.title, a.created_at
	`, userID, quarter)
	if err != nil {
		return Report{}, fmt.Errorf("query achievements: %w", err)
	}
	defer rows.Close()

	grouped := map[string][]string{}
	for rows.Next() {
		var kpiTitle, enhanced, raw, impact string
		if err := rows.Scan(&kpiTitle, &enhanced, &raw, &impact); err != nil {
			return Report{}, err
		}
		text := enhanced
		if strings.TrimSpace(text) == "" {
			text = raw
		}
		if impact != "" {
			text += " Impact: " + impact
		}
		grouped[kpiTitle] = append(grouped[kpiTitle], "- "+text)
	}

	if len(grouped) == 0 {
		grouped["No achievements logged"] = []string{"- Add daily achievements before generating a report."}
	}

	var sections []string
	titles := make([]string, 0, len(grouped))
	for title := range grouped {
		titles = append(titles, title)
	}
	slices.Sort(titles)
	for _, title := range titles {
		achievements := grouped[title]
		sections = append(sections, fmt.Sprintf("%s\n%s", title, strings.Join(achievements, "\n")))
	}
	body := strings.Join(sections, "\n\n")
	reportTitle := fmt.Sprintf("Quarterly KPI Summary %s", quarter)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO reports (user_id, quarter, title, body)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, quarter) DO UPDATE SET title = excluded.title, body = excluded.body, created_at = CURRENT_TIMESTAMP
	`, userID, quarter, reportTitle, body)
	if err != nil {
		return Report{}, fmt.Errorf("save report: %w", err)
	}

	return s.Get(ctx, userID, quarter)
}

func (s *Service) Get(ctx context.Context, userID int64, quarter string) (Report, error) {
	var report Report
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, quarter, title, body, created_at
		FROM reports
		WHERE user_id = ? AND quarter = ?
	`, userID, quarter).Scan(&report.ID, &report.UserID, &report.Quarter, &report.Title, &report.Body, &report.CreatedAt)
	return report, err
}
