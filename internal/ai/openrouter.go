package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/example/kpi-chaser/internal/config"
)

type openRouterProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewProvider(cfg config.Config) Provider {
	if cfg.OpenRouterAPIKey == "" {
		return fallbackProvider{}
	}
	return &openRouterProvider{
		apiKey:  cfg.OpenRouterAPIKey,
		baseURL: cfg.OpenRouterBaseURL,
		model:   cfg.OpenRouterModel,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *openRouterProvider) EnhanceAchievement(ctx context.Context, rawText string) (EnhancementResult, error) {
	payload := map[string]any{
		"model": p.model,
		"response_format": map[string]any{
			"type": "json_object",
		},
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": rawText},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return EnhancementResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return EnhancementResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return EnhancementResult{}, fmt.Errorf("openrouter enhance failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return EnhancementResult{}, fmt.Errorf("decode openrouter response: %w", err)
	}
	if len(result.Choices) == 0 {
		return EnhancementResult{}, fmt.Errorf("no AI response choices")
	}

	var parsed EnhancementResult
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &parsed); err != nil {
		return EnhancementResult{}, fmt.Errorf("decode AI payload: %w", err)
	}
	return parsed, nil
}

func (p *openRouterProvider) MapKPI(ctx context.Context, text string, kpis []KPITarget) (string, error) {
	options := make([]map[string]string, 0, len(kpis))
	for _, item := range kpis {
		options = append(options, map[string]string{
			"title":       item.Title,
			"description": item.Description,
		})
	}

	payload := map[string]any{
		"model": p.model,
		"response_format": map[string]any{
			"type": "json_object",
		},
		"messages": []map[string]string{
			{"role": "system", "content": "Choose the best matching KPI title for the achievement. Return JSON with key title. If nothing fits, return an empty title."},
			{"role": "user", "content": fmt.Sprintf("Achievement: %s\nKPI options: %s", text, mustJSON(options))},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("openrouter KPI mapping failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode openrouter response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no AI response choices")
	}

	var parsed struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &parsed); err != nil {
		return "", fmt.Errorf("decode AI payload: %w", err)
	}
	return strings.TrimSpace(parsed.Title), nil
}

type fallbackProvider struct{}

func (fallbackProvider) EnhanceAchievement(_ context.Context, rawText string) (EnhancementResult, error) {
	cleaned := strings.TrimSpace(rawText)
	return EnhancementResult{
		EnhancedText: "Delivered: " + cleaned,
		Category:     "Process Improvement",
		ImpactNote:   "Captured as a structured KPI-ready achievement.",
	}, nil
}

func (fallbackProvider) MapKPI(_ context.Context, text string, kpis []KPITarget) (string, error) {
	lowerText := strings.ToLower(text)
	for _, item := range kpis {
		words := strings.Fields(strings.ToLower(item.Title))
		if slices.ContainsFunc(words, func(word string) bool {
			return len(word) > 4 && strings.Contains(lowerText, word)
		}) {
			return item.Title, nil
		}
	}
	if len(kpis) == 0 {
		return "", nil
	}
	return kpis[0].Title, nil
}

func mustJSON(value any) string {
	body, _ := json.Marshal(value)
	return string(body)
}
