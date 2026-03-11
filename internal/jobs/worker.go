package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/example/kpi-chaser/internal/ai"
)

type Worker struct {
	db       *sql.DB
	queue    *Queue
	provider ai.Provider
}

func NewWorker(db *sql.DB, queue *Queue, provider ai.Provider) *Worker {
	return &Worker{db: db, queue: queue, provider: provider}
}

func (w *Worker) Run(ctx context.Context, concurrency int) error {
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.loop(ctx)
		}()
	}
	<-ctx.Done()
	wg.Wait()
	return ctx.Err()
}

func (w *Worker) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, err := w.queue.ClaimPending(ctx)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		if job == nil {
			time.Sleep(1500 * time.Millisecond)
			continue
		}

		if err := w.handle(ctx, *job); err != nil {
			_ = w.queue.Fail(ctx, job.ID, err)
			continue
		}
		_ = w.queue.Complete(ctx, job.ID)
	}
}

func (w *Worker) handle(ctx context.Context, job Job) error {
	switch job.Type {
	case TypeEnhanceAchievement:
		payload, err := DecodePayload[struct {
			AchievementID int64 `json:"achievementId"`
		}](job.Payload)
		if err != nil {
			return err
		}

		var rawText string
		err = w.db.QueryRowContext(ctx, `
			SELECT raw_text FROM achievements WHERE id = ? AND user_id = ?
		`, payload.AchievementID, job.UserID).Scan(&rawText)
		if err != nil {
			return fmt.Errorf("load achievement: %w", err)
		}

		result, err := w.provider.EnhanceAchievement(ctx, rawText)
		if err != nil {
			return err
		}

		_, err = w.db.ExecContext(ctx, `
			UPDATE achievements
			SET enhanced_text = ?, category = ?, impact_note = ?, status = 'enhanced', updated_at = CURRENT_TIMESTAMP
			WHERE id = ? AND user_id = ?
		`, result.EnhancedText, result.Category, result.ImpactNote, payload.AchievementID, job.UserID)
		if err != nil {
			return err
		}
		return w.enqueueMapKPI(ctx, job.UserID, payload.AchievementID)
	case TypeMapKPI:
		payload, err := DecodePayload[struct {
			AchievementID int64 `json:"achievementId"`
		}](job.Payload)
		if err != nil {
			return err
		}
		return w.mapAchievement(ctx, job.UserID, payload.AchievementID)
	default:
		return fmt.Errorf("unsupported job type %q", job.Type)
	}
}

func (w *Worker) enqueueMapKPI(ctx context.Context, userID, achievementID int64) error {
	payload := fmt.Sprintf(`{"achievementId":%d}`, achievementID)
	_, err := w.db.ExecContext(ctx, `
		INSERT INTO jobs (user_id, type, payload)
		VALUES (?, ?, ?)
	`, userID, TypeMapKPI, payload)
	return err
}

func (w *Worker) mapAchievement(ctx context.Context, userID, achievementID int64) error {
	var quarter string
	var enhancedText string
	err := w.db.QueryRowContext(ctx, `
		SELECT quarter, COALESCE(NULLIF(enhanced_text, ''), raw_text)
		FROM achievements
		WHERE id = ? AND user_id = ?
	`, achievementID, userID).Scan(&quarter, &enhancedText)
	if err != nil {
		return fmt.Errorf("load achievement for mapping: %w", err)
	}

	rows, err := w.db.QueryContext(ctx, `
		SELECT id, title, description
		FROM kpis
		WHERE user_id = ? AND quarter = ?
		ORDER BY created_at
	`, userID, quarter)
	if err != nil {
		return fmt.Errorf("load kpis for mapping: %w", err)
	}
	defer rows.Close()

	var targets []ai.KPITarget
	for rows.Next() {
		var target ai.KPITarget
		if err := rows.Scan(&target.ID, &target.Title, &target.Description); err != nil {
			return err
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}

	title, err := w.provider.MapKPI(ctx, enhancedText, targets)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if target.Title == title {
			_, err = w.db.ExecContext(ctx, `
				UPDATE achievements
				SET kpi_id = ?, status = 'mapped', updated_at = CURRENT_TIMESTAMP
				WHERE id = ? AND user_id = ?
			`, target.ID, achievementID, userID)
			return err
		}
	}
	return nil
}
