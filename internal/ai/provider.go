package ai

import "context"

type EnhancementResult struct {
	EnhancedText string `json:"enhancedText"`
	Category     string `json:"category"`
	ImpactNote   string `json:"impactNote"`
}

type KPITarget struct {
	ID          int64
	Title       string
	Description string
}

type Provider interface {
	EnhanceAchievement(ctx context.Context, rawText string) (EnhancementResult, error)
	MapKPI(ctx context.Context, text string, kpis []KPITarget) (string, error)
}
