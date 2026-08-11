package server

import (
	"context"
	"encoding/json"
	"net/http"

	"shelley.exe.dev/db/generated"
)

// handleGetGistStatus returns whether a gist exists for this conversation.
func (s *Server) handleGetGistStatus(w http.ResponseWriter, r *http.Request, conversationID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	conv, err := s.db.GetConversationByID(ctx, conversationID)
	if err != nil {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}
	gistURL := ""
	if conv.GistID != nil && *conv.GistID != "" {
		gistURL = gistHostURL(*conv.GistID)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"gist_id":  conv.GistID,
		"gist_url": gistURL,
	})
}

// handleExportGist creates a new secret gist for this conversation.
func (s *Server) handleExportGist(w http.ResponseWriter, r *http.Request, conversationID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	conv, err := s.db.GetConversationByID(ctx, conversationID)
	if err != nil {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	// Gate: slug must not be generic.
	if isGenericSlug(conv.Slug) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "gist_needs_name",
			"message": "Session must have a name (not 'Untitled' or 'Draft')",
		})
		return
	}

	// If a gist already exists, update it instead of creating a new one.
	if conv.GistID != nil && *conv.GistID != "" {
		s.updateGistForConversation(ctx, w, conv)
		return
	}

	messages, err := s.db.ListMessagesForContext(ctx, conversationID)
	if err != nil {
		s.logger.Error("Failed to list messages for gist", "id", conversationID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	html, err := exportGistHTML(conv, messages)
	if err != nil {
		s.logger.Error("Failed to build gist HTML", "id", conversationID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	gistID, gistURL, err := createGist(ctx, conv.Slug, html)
	if err != nil {
		err2 := classifyGHErr(err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   err2.code,
			"message": err2.msg,
		})
		return
	}

	if err := s.db.Queries(ctx, func(q *generated.Queries) error {
		return q.SetGistID(ctx, generated.SetGistIDParams{
			GistID:         &gistID,
			ConversationID: conversationID,
		})
	}); err != nil {
		s.logger.Error("Failed to save gist ID", "id", conversationID, "error", err)
		http.Error(w, "Failed to save gist ID", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"gist_id":  gistID,
		"gist_url": gistURL,
	})
}

// handleUpdateGist patches an existing gist with current conversation data.
func (s *Server) handleUpdateGist(w http.ResponseWriter, r *http.Request, conversationID string) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	conv, err := s.db.GetConversationByID(ctx, conversationID)
	if err != nil {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}
	if conv.GistID == nil || *conv.GistID == "" {
		http.Error(w, "No gist exists for this conversation", http.StatusNotFound)
		return
	}
	s.updateGistForConversation(ctx, w, conv)
}

// updateGistForConversation regenerates the HTML and updates the gist.
func (s *Server) updateGistForConversation(ctx context.Context, w http.ResponseWriter, conv *generated.Conversation) {
	messages, err := s.db.ListMessagesForContext(ctx, conv.ConversationID)
	if err != nil {
		s.logger.Error("Failed to list messages for gist update", "id", conv.ConversationID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	html, err := exportGistHTML(conv, messages)
	if err != nil {
		s.logger.Error("Failed to build gist HTML", "id", conv.ConversationID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := updateGist(ctx, *conv.GistID, conv.Slug, html); err != nil {
		err2 := classifyGHErr(err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   err2.code,
			"message": err2.msg,
		})
		return
	}

	gistURL := gistHostURL(*conv.GistID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"gist_id":  *conv.GistID,
		"gist_url": gistURL,
	})
}

// UpdateGistForCompletion is called after an agent turn completes. It checks
// if a gist exists and updates it. Errors are logged, never surfaced.
func (s *Server) UpdateGistForCompletion(ctx context.Context, conversationID string) {
	conv, err := s.db.GetConversationByID(ctx, conversationID)
	if err != nil || conv.GistID == nil || *conv.GistID == "" {
		return
	}
	messages, err := s.db.ListMessagesForContext(ctx, conversationID)
	if err != nil {
		s.logger.Warn("gist auto-update: failed to list messages", "id", conversationID, "error", err)
		return
	}
	html, err := exportGistHTML(conv, messages)
	if err != nil {
		s.logger.Warn("gist auto-update: failed to build HTML", "id", conversationID, "error", err)
		return
	}
	if err := updateGist(ctx, *conv.GistID, conv.Slug, html); err != nil {
		s.logger.Warn("gist auto-update: gh failed", "id", conversationID, "error", err)
	}
}
