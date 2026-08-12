package server

import (
	"context"
	"strings"
	"testing"

	"shelley.exe.dev/db"
	"shelley.exe.dev/llm"
)

// writeAgentMsg writes an agent message with the given content blocks.
func writeAgentMsg(t *testing.T, database *db.DB, convID string, content []llm.Content) {
	t.Helper()
	_, err := database.CreateMessage(context.Background(), db.CreateMessageParams{
		ConversationID: convID,
		Type:           db.MessageTypeAgent,
		LLMData: llm.Message{
			Role:    llm.MessageRoleAssistant,
			Content: content,
		},
	})
	if err != nil {
		t.Fatalf("CreateMessage(%s): %v", convID, err)
	}
}

// writeUserMsg writes a user message with the given text.
func writeUserMsg(t *testing.T, database *db.DB, convID string, text string) {
	t.Helper()
	_, err := database.CreateMessage(context.Background(), db.CreateMessageParams{
		ConversationID: convID,
		Type:           db.MessageTypeUser,
		LLMData: llm.Message{
			Role:    llm.MessageRoleUser,
			Content: []llm.Content{{Type: llm.ContentTypeText, Text: text}},
		},
	})
	if err != nil {
		t.Fatalf("CreateMessage(%s): %v", convID, err)
	}
}

func previewTextBlock(s string) llm.Content {
	return llm.Content{Type: llm.ContentTypeText, Text: s}
}

func previewToolBlock(name string) llm.Content {
	return llm.Content{Type: llm.ContentTypeToolUse, ToolName: name}
}

func previewThinkingBlock(s string) llm.Content {
	return llm.Content{Type: llm.ContentTypeThinking, Thinking: s}
}

// TestConversationListPreview exercises the SQL-side preview extraction that
// the conversation-list query computes inline (correlated subqueries against
// messages, see conversations.sql): it must pick the most recent agent
// message that has a usable text block, and within that message use the LAST
// non-empty text block (the summary/conclusion in tool-calling turns). It also
// checks the tool-only fallback and that a preview carries a timestamp.
func TestConversationListPreview(t *testing.T) {
	t.Parallel()
	srv, database, _ := newTestServer(t)
	ctx := context.Background()

	// Conversation A: last agent message mixes text + tool calls. The final
	// text block should win over the earlier one.
	convA, err := database.CreateConversation(ctx, nil, true, nil, nil, db.ConversationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	writeAgentMsg(t, database, convA.ConversationID, []llm.Content{previewTextBlock("first answer")})
	writeAgentMsg(t, database, convA.ConversationID, []llm.Content{
		previewTextBlock("intermediate thought"),
		previewToolBlock("bash"),
		previewTextBlock("final summary"),
	})

	// Conversation B: the newest agent message is tool-only. Since tool calls
	// are now included, the preview shows the tool name.
	convB, err := database.CreateConversation(ctx, nil, true, nil, nil, db.ConversationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	writeAgentMsg(t, database, convB.ConversationID, []llm.Content{previewTextBlock("earlier text")})
	writeAgentMsg(t, database, convB.ConversationID, []llm.Content{previewToolBlock("bash")})

	// Conversation C: the agent message has only a tool call. The preview
	// shows the tool name.
	convC, err := database.CreateConversation(ctx, nil, true, nil, nil, db.ConversationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	writeAgentMsg(t, database, convC.ConversationID, []llm.Content{previewToolBlock("bash")})

	// Conversation D: a very long agent reply. The preview is capped in SQL
	// (substr ..., 1, 300) so we don't haul multi-KB replies across the driver
	// for a one-line UI field.
	convD, err := database.CreateConversation(ctx, nil, true, nil, nil, db.ConversationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	longText := strings.Repeat("x", 5000)
	writeAgentMsg(t, database, convD.ConversationID, []llm.Content{previewTextBlock(longText)})

	// Conversation E: the agent message has only thinking content (no text
	// blocks). The preview must fall back to the thinking text.
	convE, err := database.CreateConversation(ctx, nil, true, nil, nil, db.ConversationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	writeAgentMsg(t, database, convE.ConversationID, []llm.Content{previewThinkingBlock("thinking only")})

	// Conversation F: the agent message has thinking then text. Within a
	// message the text block (later in the content array) wins, so the preview
	// shows the text, not the thinking.
	convF, err := database.CreateConversation(ctx, nil, true, nil, nil, db.ConversationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	writeAgentMsg(t, database, convF.ConversationID, []llm.Content{
		previewThinkingBlock("thinking first"),
		previewTextBlock("text wins"),
	})

	// Conversation G: a user message is the newest message. The preview shows
	// the user's text.
	convG, err := database.CreateConversation(ctx, nil, true, nil, nil, db.ConversationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	writeAgentMsg(t, database, convG.ConversationID, []llm.Content{previewTextBlock("old reply")})
	writeUserMsg(t, database, convG.ConversationID, "what about pelicans?")

	// Drive the actual conversation-list path: the preview/preview_updated_at
	// columns are computed inside the list query itself, then copied onto each
	// ConversationWithState by decorateConversations. Reading the list back is
	// exactly what /api/conversations serves.
	list, err := srv.conversationListWithStateInternal(ctx, 100, 0, "", false, true)
	if err != nil {
		t.Fatalf("conversationListWithStateInternal: %v", err)
	}
	byID := map[string]ConversationWithState{}
	for _, cws := range list {
		byID[cws.ConversationID] = cws
	}

	if got := byID[convA.ConversationID].Preview; got != "final summary" {
		t.Errorf("convA preview = %q, want %q", got, "final summary")
	}
	if got := byID[convB.ConversationID].Preview; got != "bash" {
		t.Errorf("convB preview = %q, want %q", got, "bash")
	}
	if got := byID[convC.ConversationID].Preview; got != "bash" {
		t.Errorf("convC preview = %q, want %q", got, "bash")
	}
	if byID[convA.ConversationID].PreviewUpdatedAt == "" {
		t.Errorf("convA preview should carry an updatedAt timestamp")
	}
	// The preview message's max sequence_id is surfaced too.
	if byID[convA.ConversationID].MaxSequenceID == 0 {
		t.Errorf("convA should carry a non-zero max_sequence_id")
	}
	// convD's long preview is truncated to the SQL cap (300 chars) and still
	// carries a timestamp.
	if got := byID[convD.ConversationID].Preview; len(got) != 300 {
		t.Errorf("convD preview len = %d, want 300 (truncated)", len(got))
	}
	if byID[convD.ConversationID].PreviewUpdatedAt == "" {
		t.Errorf("convD preview should carry an updatedAt timestamp")
	}
	// convE has only thinking content, so the preview shows the thinking text.
	if got := byID[convE.ConversationID].Preview; got != "thinking only" {
		t.Errorf("convE preview = %q, want %q", got, "thinking only")
	}
	// convF has thinking then text; the text block wins within the message.
	if got := byID[convF.ConversationID].Preview; got != "text wins" {
		t.Errorf("convF preview = %q, want %q", got, "text wins")
	}
	// convG has a user message after the agent message; user text wins.
	if got := byID[convG.ConversationID].Preview; got != "what about pelicans?" {
		t.Errorf("convG preview = %q, want %q", got, "what about pelicans?")
	}
}
