package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"shelley.exe.dev/db/generated"
	"shelley.exe.dev/llm"
	"shelley.exe.dev/llm/ant"
	"shelley.exe.dev/llm/gem"
	"shelley.exe.dev/llm/llmhttp"
	"shelley.exe.dev/llm/oai"
	"shelley.exe.dev/models"
)

// ModelAPI is the API representation of a model
type ModelAPI struct {
	ModelID         string `json:"model_id"`
	DisplayName     string `json:"display_name"`
	ProviderType    string `json:"provider_type"`
	Endpoint        string `json:"endpoint"`
	APIKey          string `json:"api_key"`
	ModelName       string `json:"model_name"`
	MaxTokens       int64  `json:"max_tokens"`
	Tags            string `json:"tags"` // Comma-separated tags (e.g., "slug" for slug generation)
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	// ImageSupport is one of "auto", "yes", or "no". "auto" is resolved
	// automatically from the model's endpoint and name.
	ImageSupport      string `json:"image_support"`
	ReasoningSupport  string `json:"reasoning_support"`
	ReasoningMap      string `json:"reasoning_map"`
	SupportsReasoning bool   `json:"supports_reasoning"`
	// SupportsImages is the resolved boolean that "image_support" evaluates
	// to for this model. It lets the UI show what "auto" resolves to.
	SupportsImages bool `json:"supports_images"`
	Enabled        bool `json:"enabled"`
	ContextWindow  int64 `json:"context_window"`
	// Pricing: USD per million tokens. Zero means unknown.
	InputPrice     float64 `json:"input_price"`
	OutputPrice    float64 `json:"output_price"`
	CacheReadPrice float64 `json:"cache_read_price"`
	CacheWritePrice float64 `json:"cache_write_price"`
}

// CreateModelRequest is the request body for creating a model.
type CreateModelRequest struct {
	DisplayName      string `json:"display_name"`
	ProviderType     string `json:"provider_type"`
	Endpoint         string `json:"endpoint"`
	APIKey           string `json:"api_key"`
	ModelName        string `json:"model_name"`
	MaxTokens        int64  `json:"max_tokens"`
	Tags             string `json:"tags"` // Comma-separated tags
	ReasoningEffort  string `json:"reasoning_effort,omitempty"`
	ImageSupport     string `json:"image_support"`     // "auto"|"yes"|"no"; empty = "auto"
	ReasoningSupport string `json:"reasoning_support"` // "auto"|"yes"|"no"; empty = "auto"
	ReasoningMap     string `json:"reasoning_map"`     // JSON map of Shelley level to provider-supported level
	Enabled          bool   `json:"enabled"`
	ContextWindow    int64  `json:"context_window"`
}

// UpdateModelRequest is the request body for updating a model.
type UpdateModelRequest struct {
	DisplayName      string  `json:"display_name"`
	ProviderType     string  `json:"provider_type"`
	Endpoint         string  `json:"endpoint"`
	APIKey           string  `json:"api_key"` // Empty string means keep existing
	ModelName        string  `json:"model_name"`
	MaxTokens        int64   `json:"max_tokens"`
	Tags             string  `json:"tags"` // Comma-separated tags
	ReasoningEffort  *string `json:"reasoning_effort,omitempty"`
	ImageSupport     string  `json:"image_support"`     // "auto"|"yes"|"no"; empty preserves existing
	ReasoningSupport string  `json:"reasoning_support"` // "auto"|"yes"|"no"; empty preserves existing
	ReasoningMap     string  `json:"reasoning_map"`
	Enabled          *bool   `json:"enabled,omitempty"`
	ContextWindow    *int64  `json:"context_window,omitempty"`
	InputPrice       *float64 `json:"input_price,omitempty"`
	OutputPrice      *float64 `json:"output_price,omitempty"`
	CacheReadPrice   *float64 `json:"cache_read_price,omitempty"`
	CacheWritePrice  *float64 `json:"cache_write_price,omitempty"`
}

// validImageSupport returns the canonical value or an error.
func validSupportSetting(field, v string) (string, error) {
	switch v {
	case "", "auto":
		return "auto", nil
	case "yes", "no":
		return v, nil
	default:
		return "", fmt.Errorf("%s must be one of 'auto', 'yes', 'no'; got %q", field, v)
	}
}

func validImageSupport(v string) (string, error) {
	return validSupportSetting("image_support", v)
}

func validReasoningSupport(v string) (string, error) {
	return validSupportSetting("reasoning_support", v)
}

func validReasoningMap(raw string) error {
	if raw == "" {
		return nil
	}
	var values map[string]string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return fmt.Errorf("reasoning_map must be a JSON object: %w", err)
	}
	valid := map[string]bool{"off": true, "minimal": true, "low": true, "medium": true, "high": true, "xhigh": true, "max": true}
	for from, to := range values {
		if !valid[from] || strings.TrimSpace(to) == "" {
			return fmt.Errorf("reasoning_map keys must use off, minimal, low, medium, high, xhigh, or max and values must be non-empty; got %q: %q", from, to)
		}
	}
	return nil
}

// TestModelRequest is the request body for testing a model
type TestModelRequest struct {
	ModelID          string  `json:"model_id,omitempty"` // If provided, use stored API key
	ProviderType     string  `json:"provider_type"`
	Endpoint         string  `json:"endpoint"`
	APIKey           string  `json:"api_key"`
	ModelName        string  `json:"model_name"`
	ReasoningSupport string  `json:"reasoning_support"`
	ReasoningMap     string  `json:"reasoning_map"`
	ReasoningEffort  *string `json:"reasoning_effort,omitempty"`
}

func toModelAPI(m generated.Model) ModelAPI {
	return ModelAPI{
		ModelID:           m.ModelID,
		DisplayName:       m.DisplayName,
		ProviderType:      m.ProviderType,
		Endpoint:          m.Endpoint,
		APIKey:            m.ApiKey,
		ModelName:         m.ModelName,
		MaxTokens:         m.MaxTokens,
		Tags:              m.Tags,
		ReasoningEffort:   m.ReasoningEffort,
		ImageSupport:      m.ImageSupport,
		ReasoningSupport:  m.ReasoningSupport,
		ReasoningMap:      m.ReasoningMap,
		SupportsReasoning: models.ResolveSupportsReasoning(m.Endpoint, m.ModelName, m.ReasoningSupport),
		SupportsImages:    models.ResolveSupportsImages(m.Endpoint, m.ModelName, m.ImageSupport),
		Enabled:           m.Enabled == 1,
		ContextWindow:     m.ContextWindow,
		InputPrice:        m.InputPrice,
		OutputPrice:       m.OutputPrice,
		CacheReadPrice:    m.CacheReadPrice,
		CacheWritePrice:   m.CacheWritePrice,
	}
}

func (s *Server) handleCustomModels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListModels(w, r)
	case http.MethodPost:
		s.handleCreateModel(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	models, err := s.db.GetModels(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get models: %v", err), http.StatusInternalServerError)
		return
	}

	apiModels := make([]ModelAPI, len(models))
	for i, m := range models {
		apiModels[i] = toModelAPI(m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiModels)
}

func (s *Server) handleCreateModel(w http.ResponseWriter, r *http.Request) {
	var req CreateModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.DisplayName == "" || req.ProviderType == "" || req.Endpoint == "" || req.APIKey == "" || req.ModelName == "" {
		http.Error(w, "display_name, provider_type, endpoint, api_key, and model_name are required", http.StatusBadRequest)
		return
	}

	// Validate provider type
	if req.ProviderType != "anthropic" && req.ProviderType != "openai" && req.ProviderType != "openai-responses" && req.ProviderType != "gemini" {
		http.Error(w, "provider_type must be 'anthropic', 'openai', 'openai-responses', or 'gemini'", http.StatusBadRequest)
		return
	}

	// Generate a human-readable model ID derived from the endpoint and model name.
	modelID, err := s.generateUniqueModelID(r.Context(), req.Endpoint, req.ModelName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate model ID: %v", err), http.StatusInternalServerError)
		return
	}

	// Default max tokens
	if req.MaxTokens <= 0 {
		req.MaxTokens = 200000
	}

	imageSupport, err := validImageSupport(req.ImageSupport)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reasoningSupport, err := validReasoningSupport(req.ReasoningSupport)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validReasoningMap(req.ReasoningMap); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	enabled := int64(1)
	if !req.Enabled {
		enabled = 0
	}
	model, err := s.db.CreateModel(r.Context(), generated.CreateModelParams{
		ModelID:          modelID,
		DisplayName:      req.DisplayName,
		ProviderType:     req.ProviderType,
		Endpoint:         req.Endpoint,
		ApiKey:           req.APIKey,
		ModelName:        req.ModelName,
		MaxTokens:        req.MaxTokens,
		Tags:             req.Tags,
		ReasoningEffort:  req.ReasoningEffort,
		ImageSupport:     imageSupport,
		ReasoningSupport: reasoningSupport,
		ReasoningMap:     req.ReasoningMap,
		Enabled:          enabled,
		ContextWindow:    req.ContextWindow,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create model: %v", err), http.StatusInternalServerError)
		return
	}

	// Refresh the model manager's cache
	if err := s.llmManager.RefreshCustomModels(); err != nil {
		s.logger.Warn("Failed to refresh custom models cache", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toModelAPI(*model))
}

func (s *Server) handleCustomModel(w http.ResponseWriter, r *http.Request) {
	// Extract model ID from URL path: /api/custom-models/{id} or /api/custom-models/{id}/duplicate
	path := strings.TrimPrefix(r.URL.Path, "/api/custom-models/")
	if path == "" {
		http.Error(w, "Invalid model ID", http.StatusBadRequest)
		return
	}

	// Check for /duplicate suffix
	if strings.HasSuffix(path, "/duplicate") {
		modelID := strings.TrimSuffix(path, "/duplicate")
		if r.Method == http.MethodPost {
			s.handleDuplicateModel(w, r, modelID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.Contains(path, "/") {
		http.Error(w, "Invalid model ID", http.StatusBadRequest)
		return
	}
	modelID := path

	switch r.Method {
	case http.MethodGet:
		s.handleGetModel(w, r, modelID)
	case http.MethodPut:
		s.handleUpdateModel(w, r, modelID)
	case http.MethodDelete:
		s.handleDeleteModel(w, r, modelID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetModel(w http.ResponseWriter, r *http.Request, modelID string) {
	model, err := s.db.GetModel(r.Context(), modelID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get model: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toModelAPI(*model))
}

func (s *Server) handleUpdateModel(w http.ResponseWriter, r *http.Request, modelID string) {
	// First, get the existing model to get the current API key if not provided
	existing, err := s.db.GetModel(r.Context(), modelID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Model not found: %v", err), http.StatusNotFound)
		return
	}

	var req UpdateModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Fall back to existing values for all fields not provided.
	if req.DisplayName == "" {
		req.DisplayName = existing.DisplayName
	}
	if req.ProviderType == "" {
		req.ProviderType = existing.ProviderType
	}
	if req.Endpoint == "" {
		req.Endpoint = existing.Endpoint
	}
	if req.ModelName == "" {
		req.ModelName = existing.ModelName
	}
	if req.Tags == "" {
		req.Tags = existing.Tags
	}
	if req.ReasoningMap == "" {
		req.ReasoningMap = existing.ReasoningMap
	}

	// Use existing API key if not provided
	apiKey := req.APIKey
	if apiKey == "" {
		apiKey = existing.ApiKey
	}

	// Default max tokens
	if req.MaxTokens <= 0 {
		req.MaxTokens = existing.MaxTokens
	}

	// Empty support settings preserve the existing values; otherwise validate.
	imageSupport := existing.ImageSupport
	if req.ImageSupport != "" {
		v, err := validImageSupport(req.ImageSupport)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		imageSupport = v
	}
	reasoningSupport := existing.ReasoningSupport
	if req.ReasoningSupport != "" {
		v, err := validReasoningSupport(req.ReasoningSupport)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		reasoningSupport = v
	}
	if err := validReasoningMap(req.ReasoningMap); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reasoningEffort := existing.ReasoningEffort
	if req.ReasoningEffort != nil {
		reasoningEffort = *req.ReasoningEffort
	}
	enabled := existing.Enabled
	if req.Enabled != nil {
		if *req.Enabled {
			enabled = 1
		} else {
			enabled = 0
		}
	}
	contextWindow := existing.ContextWindow
	if req.ContextWindow != nil {
		contextWindow = *req.ContextWindow
	}

	inputPrice := existing.InputPrice
	if req.InputPrice != nil {
		inputPrice = *req.InputPrice
	}
	outputPrice := existing.OutputPrice
	if req.OutputPrice != nil {
		outputPrice = *req.OutputPrice
	}
	cacheReadPrice := existing.CacheReadPrice
	if req.CacheReadPrice != nil {
		cacheReadPrice = *req.CacheReadPrice
	}
	cacheWritePrice := existing.CacheWritePrice
	if req.CacheWritePrice != nil {
		cacheWritePrice = *req.CacheWritePrice
	}
	model, err := s.db.UpdateModel(r.Context(), generated.UpdateModelParams{
		DisplayName:      req.DisplayName,
		ProviderType:     req.ProviderType,
		Endpoint:         req.Endpoint,
		ApiKey:           apiKey,
		ModelName:        req.ModelName,
		MaxTokens:        req.MaxTokens,
		Tags:             req.Tags,
		ReasoningEffort:  reasoningEffort,
		ImageSupport:     imageSupport,
		ReasoningSupport: reasoningSupport,
		ReasoningMap:     req.ReasoningMap,
		Enabled:          enabled,
		ContextWindow:    contextWindow,
		InputPrice:       inputPrice,
		OutputPrice:      outputPrice,
		CacheReadPrice:   cacheReadPrice,
		CacheWritePrice:  cacheWritePrice,
		ModelID:          modelID,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update model: %v", err), http.StatusInternalServerError)
		return
	}

	// Refresh the model manager's cache
	if err := s.llmManager.RefreshCustomModels(); err != nil {
		s.logger.Warn("Failed to refresh custom models cache", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toModelAPI(*model))
}

func (s *Server) handleDeleteModel(w http.ResponseWriter, r *http.Request, modelID string) {
	err := s.db.DeleteModel(r.Context(), modelID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete model: %v", err), http.StatusInternalServerError)
		return
	}

	// Refresh the model manager's cache
	if err := s.llmManager.RefreshCustomModels(); err != nil {
		s.logger.Warn("Failed to refresh custom models cache", "error", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// DuplicateModelRequest allows overriding fields when duplicating
type DuplicateModelRequest struct {
	DisplayName string `json:"display_name,omitempty"`
}

func (s *Server) handleDuplicateModel(w http.ResponseWriter, r *http.Request, modelID string) {
	// Get the source model (including API key)
	source, err := s.db.GetModel(r.Context(), modelID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Source model not found: %v", err), http.StatusNotFound)
		return
	}

	// Parse optional overrides
	var req DuplicateModelRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req) // Ignore errors - all fields optional
	}

	// Generate a new human-readable model ID. Since the duplicate shares the
	// source's endpoint and model name, this naturally gets a numeric suffix.
	newModelID, err := s.generateUniqueModelID(r.Context(), source.Endpoint, source.ModelName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate model ID: %v", err), http.StatusInternalServerError)
		return
	}

	// Use provided display name or generate one
	displayName := req.DisplayName
	if displayName == "" {
		displayName = source.DisplayName + " (copy)"
	}

	// Create the duplicate with the same API key
	model, err := s.db.CreateModel(r.Context(), generated.CreateModelParams{
		ModelID:          newModelID,
		DisplayName:      displayName,
		ProviderType:     source.ProviderType,
		Endpoint:         source.Endpoint,
		ApiKey:           source.ApiKey, // Copy the API key!
		ModelName:        source.ModelName,
		MaxTokens:        source.MaxTokens,
		Tags:             "", // Don't copy tags
		ReasoningEffort:  source.ReasoningEffort,
		ImageSupport:     source.ImageSupport,
		ReasoningSupport: source.ReasoningSupport,
		ReasoningMap:     source.ReasoningMap,
		Enabled:          source.Enabled,
		ContextWindow:    source.ContextWindow,
		InputPrice:       source.InputPrice,
		OutputPrice:      source.OutputPrice,
		CacheReadPrice:   source.CacheReadPrice,
		CacheWritePrice:  source.CacheWritePrice,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to duplicate model: %v", err), http.StatusInternalServerError)
		return
	}

	// Refresh the model manager's cache
	if err := s.llmManager.RefreshCustomModels(); err != nil {
		s.logger.Warn("Failed to refresh custom models cache", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toModelAPI(*model))
}

func (s *Server) handleTestModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TestModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// A model ID supplies hidden legacy configuration and, when omitted by the
	// caller, its stored API key. The UI intentionally does not expose the
	// legacy reasoning_effort field, but Test must still mirror runtime.
	if req.ModelID != "" {
		model, err := s.db.GetModel(r.Context(), req.ModelID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Model not found: %v", err), http.StatusNotFound)
			return
		}
		if req.APIKey == "" {
			req.APIKey = model.ApiKey
		}
		if req.ReasoningEffort == nil {
			req.ReasoningEffort = &model.ReasoningEffort
		}
	}

	if req.ProviderType == "" || req.Endpoint == "" || req.APIKey == "" || req.ModelName == "" {
		http.Error(w, "provider_type, endpoint, api_key, and model_name are required", http.StatusBadRequest)
		return
	}

	reasoningEffort := ""
	if req.ReasoningEffort != nil {
		reasoningEffort = *req.ReasoningEffort
	}

	// Create the appropriate service based on provider type
	var service llm.Service
	switch req.ProviderType {
	case "anthropic":
		service = &ant.Service{
			APIKey:        req.APIKey,
			URL:           req.Endpoint,
			Model:         req.ModelName,
			ThinkingLevel: llm.ThinkingLevelMedium,
		}
	case "openai":
		service = &oai.Service{
			APIKey:          req.APIKey,
			ModelURL:        req.Endpoint,
			ReasoningEffort: reasoningEffort,
			Model: oai.Model{
				UserName:           "",
				ModelName:          req.ModelName,
				TextVerbosity:      "",
				URL:                req.Endpoint,
				APIKeyEnv:          "",
				IsReasoningModel:   false,
				UseSimplifiedPatch: false,
				SupportsImages:     true,
			},
		}
	case "gemini":
		service = &gem.Service{
			APIKey:          req.APIKey,
			URL:             req.Endpoint,
			Model:           req.ModelName,
			ReasoningEffort: reasoningEffort,
		}
	case "openai-responses":
		service = &oai.ResponsesService{
			APIKey: req.APIKey,
			Model: oai.Model{
				UserName:           "",
				ModelName:          req.ModelName,
				TextVerbosity:      "",
				URL:                req.Endpoint,
				APIKeyEnv:          "",
				IsReasoningModel:   false,
				UseSimplifiedPatch: false,
				SupportsImages:     true,
			},
			// Match createServiceFromModel so Test reflects real runtime behavior:
			// medium is the default when no explicit override is given.
			ThinkingLevel:   llm.ThinkingLevelMedium,
			ReasoningEffort: reasoningEffort,
		}
	default:
		http.Error(w, "Invalid provider_type", http.StatusBadRequest)
		return
	}
	if _, err := validReasoningSupport(req.ReasoningSupport); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validReasoningMap(req.ReasoningMap); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	service = models.WrapReasoningConfig(service, req.Endpoint, req.ModelName, req.ReasoningSupport, req.ReasoningMap)

	// Send a simple test request
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	request := &llm.Request{
		Messages: []llm.Message{
			{
				Role: llm.MessageRoleUser,
				Content: []llm.Content{
					{Type: llm.ContentTypeText, Text: "Say 'test successful' in exactly two words."},
				},
			},
		},
	}

	response, err := service.Do(ctx, request)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Test failed: %v", err),
		})
		return
	}

	// Check if we got a response with actual text content
	// (skip thinking blocks which may appear first)
	var responseText string
	for _, content := range response.Content {
		if content.Type == llm.ContentTypeText && content.Text != "" {
			responseText = content.Text
			break
		}
	}

	if responseText == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Test failed: empty response from model",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Test successful! Response: %s", responseText),
	})
}

// ImportModelsRequest is the body for POST /api/custom-models/import.
type ImportModelsRequest struct {
	ProviderType string `json:"provider_type"` // only "openai" accepted
	Endpoint     string `json:"endpoint"`
	APIKey       string `json:"api_key"`
}

// ImportModelsResponse reports the result of an import.
type ImportModelsResponse struct {
	Imported int          `json:"imported"`
	Skipped  int          `json:"skipped"`
	Models   []ModelAPI   `json:"models"`
}

// oaiListModelsResponse is the shape of GET /v1/models for OpenAI-compatible APIs.
type oaiListModelsResponse struct {
	Data []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		// Pricing is per-million-token pricing in USD. Prompt = input, Completion = output.
		// When only cache_prompt is present it is used for both cache read and write.
		Pricing *struct {
			Prompt     string `json:"prompt"`
			Completion string `json:"completion"`
			CacheRead  string `json:"cache_prompt"`
			CacheWrite string `json:"cache_write"`
		} `json:"pricing"`
		ContextLength        int64 `json:"context_length"`
		MaxCompletionTokens  int64 `json:"max_completion_tokens"`
	} `json:"data"`
}

// parseModelPricing extracts per-million-token USD pricing from a /v1/models
// pricing object. The API reports prices as per-token strings; we convert to
// per-million. When cache_prompt is set but cache_write is not, it is used for
// both cache read and cache write pricing.
func parseModelPricing(p *struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
	CacheRead  string `json:"cache_prompt"`
	CacheWrite string `json:"cache_write"`
}) (input, output, cacheRead, cacheWrite float64) {
	if p == nil {
		return
	}
	parseFloat := func(s string) float64 {
		if s == "" {
			return 0
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		return v // already per-million
	}
	input = parseFloat(p.Prompt)
	output = parseFloat(p.Completion)
	cacheRead = parseFloat(p.CacheRead)
	cacheWrite = parseFloat(p.CacheWrite)
	if cacheWrite == 0 && cacheRead > 0 {
		cacheWrite = cacheRead
	}
	return
}

const maxImportModelsBytes = 4 << 20 // 4 MiB

func (s *Server) handleImportModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ImportModelsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.ProviderType != "openai" {
		http.Error(w, "provider_type must be 'openai' (only OpenAI Chat Completions endpoints supported for import)", http.StatusBadRequest)
		return
	}
	if req.Endpoint == "" || req.APIKey == "" {
		http.Error(w, "endpoint and api_key are required", http.StatusBadRequest)
		return
	}

	// Fetch /v1/models from the endpoint.
	modelsURL := strings.TrimRight(req.Endpoint, "/") + "/models"
	fetchReq, err := http.NewRequestWithContext(r.Context(), "GET", modelsURL, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusInternalServerError)
		return
	}
	fetchReq.Header.Set("Authorization", "Bearer "+req.APIKey)

	httpc := llmhttp.NewClient(nil)
	resp, err := httpc.Do(fetchReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch models from endpoint: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		http.Error(w, fmt.Sprintf("Endpoint returned %d: %s", resp.StatusCode, string(body)), http.StatusBadGateway)
		return
	}

	var list oaiListModelsResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxImportModelsBytes)).Decode(&list); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse models response: %v", err), http.StatusBadGateway)
		return
	}

	// Fetch existing models to detect duplicates.
	existing, err := s.db.GetModels(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list existing models: %v", err), http.StatusInternalServerError)
		return
	}
	existingKeys := make(map[string]bool, len(existing))
	for _, m := range existing {
		// Match by endpoint + model_name.
		existingKeys[m.Endpoint+"|"+m.ModelName] = true
	}

	var result ImportModelsResponse
	for _, m := range list.Data {
		if m.ID == "" {
			continue
		}
		key := req.Endpoint + "|" + m.ID
		if existingKeys[key] {
			result.Skipped++
			continue
		}
		existingKeys[key] = true

		displayName := m.Name
		if i := strings.Index(displayName, ":"); i >= 0 {
			displayName = strings.TrimSpace(displayName[i+1:])
		}
		if displayName == "" {
			displayName = m.ID
		}

		modelID, err := s.generateUniqueModelID(r.Context(), req.Endpoint, m.ID)
		if err != nil {
			s.logger.Warn("Failed to generate model ID for imported model", "model_name", m.ID, "error", err)
			continue
		}

		inputPrice, outputPrice, cacheReadPrice, cacheWritePrice := parseModelPricing(m.Pricing)
		maxTokens := int64(200000)
		if m.MaxCompletionTokens > 0 {
			maxTokens = m.MaxCompletionTokens
		}
		contextWindow := m.ContextLength
		created, err := s.db.CreateModel(r.Context(), generated.CreateModelParams{
			ModelID:          modelID,
			DisplayName:      displayName,
			ProviderType:     "openai",
			Endpoint:         req.Endpoint,
			ApiKey:           req.APIKey,
			ModelName:        m.ID,
			MaxTokens:        maxTokens,
			Tags:             "",
			ReasoningEffort:  "",
			ImageSupport:     "auto",
			ReasoningSupport: "auto",
			ReasoningMap:     "",
			Enabled:          1,
			ContextWindow:    contextWindow,
			InputPrice:       inputPrice,
			OutputPrice:      outputPrice,
			CacheReadPrice:   cacheReadPrice,
			CacheWritePrice:  cacheWritePrice,
		})
		if err != nil {
			s.logger.Warn("Failed to create imported model", "model_name", m.ID, "error", err)
			continue
		}
		result.Models = append(result.Models, toModelAPI(*created))
		result.Imported++
	}

	// Refresh the model manager so imported models are available immediately.
	if err := s.llmManager.RefreshCustomModels(); err != nil {
		s.logger.Warn("Failed to refresh custom models after import", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
