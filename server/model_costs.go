package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"shelley.exe.dev/models/modelsdev"
)

// resolveCost returns pricing (USD per million tokens) for (endpoint, model).
// It checks the DB's custom models first (imported from /v1/models), then
// falls back to the models.dev snapshot.
func (s *Server) resolveCost(ctx context.Context, endpoint, model string) (modelsdev.Cost, bool) {
	if model == "" {
		return modelsdev.Cost{}, false
	}
	// Check DB-stored pricing (from /v1/models import). Usage URLs carry
	// extra path segments (e.g. /chat/completions) that are not part of the
	// DB-stored base endpoint. Match by prefix, but only when the next byte
	// after the base is '/' or end-of-string (prevents hostname spoofing).
	if dbModels, err := s.db.GetEnabledModels(ctx); err == nil {
		for _, m := range dbModels {
			if m.ModelName == model && m.InputPrice > 0 {
				if m.Endpoint == endpoint ||
					(len(endpoint) > len(m.Endpoint) && endpoint[len(m.Endpoint)] == '/' && strings.HasPrefix(endpoint, m.Endpoint)) {
					return modelsdev.Cost{
						Input:      m.InputPrice,
						Output:     m.OutputPrice,
						CacheRead:  m.CacheReadPrice,
						CacheWrite: m.CacheWritePrice,
					}, true
				}
			}
		}
	}
	// Fall back to models.dev snapshot.
	return modelsdev.LookupCost(endpoint, model)
}

// handleModelCosts resolves pricing (USD per million tokens) for a batch of
// (model, url) pairs seen in a conversation's usage data. Models without
// pricing map to null.
func (s *Server) handleModelCosts(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Models []struct {
			Model string `json:"model"`
			URL   string `json:"url"`
		} `json:"models"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	costs := make(map[string]*modelsdev.Cost, len(req.Models))
	for _, m := range req.Models {
		if m.Model == "" {
			continue
		}
		if c, found := s.resolveCost(r.Context(), m.URL, m.Model); found {
			costs[m.Model] = &c
		} else {
			costs[m.Model] = nil
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"costs": costs})
}

// handleSubagentUsage aggregates LLM usage across a conversation's subagents
// (recursively) and prices it. The token-cost graph shows this as a separate
// "plus $X for subagents" line; subagent calls are not part of the graph.
// Descendants' indirect usage (other_usage_data entries) is included.
func (s *Server) handleSubagentUsage(w http.ResponseWriter, r *http.Request, conversationID string) {
	rows, err := s.db.GetSubagentUsage(r.Context(), conversationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	otherRows, err := s.db.GetSubagentOtherUsage(r.Context(), conversationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var resp struct {
		LLMCalls       int64    `json:"llm_calls"`
		EstimatedUsd   float64  `json:"estimated_usd"`
		ReportedUsd    float64  `json:"reported_usd"`
		UnpricedModels []string `json:"unpriced_models"`
		UnpricedCalls  int64    `json:"unpriced_calls"`
	}
	resp.UnpricedModels = []string{}
	fold := func(model, url string, llmCalls, in, cacheWrite, cacheRead, out int64, costUsd float64) {
		resp.LLMCalls += llmCalls
		resp.ReportedUsd += costUsd
		if c, found := s.resolveCost(r.Context(), url, model); found {
			resp.EstimatedUsd += float64(in)*c.Input/1e6 +
				float64(cacheWrite)*c.CacheWrite/1e6 +
				float64(cacheRead)*c.CacheRead/1e6 +
				float64(out)*c.Output/1e6
		} else {
			resp.UnpricedModels = append(resp.UnpricedModels, model)
			resp.UnpricedCalls += llmCalls
		}
	}
	for _, row := range rows {
		model, url := "", ""
		if row.ModelName != nil {
			model = *row.ModelName
		}
		if row.LlmApiUrl != nil {
			url = *row.LlmApiUrl
		}
		fold(model, url, row.LlmCalls, row.InputTokens, row.CacheCreationInputTokens, row.CacheReadInputTokens, row.OutputTokens, row.CostUsd)
	}
	for _, row := range otherRows {
		fold(row.ModelName, row.LlmApiUrl, row.LlmCalls, row.InputTokens, row.CacheCreationInputTokens, row.CacheReadInputTokens, row.OutputTokens, row.CostUsd)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
