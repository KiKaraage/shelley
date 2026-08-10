package server

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"shelley.exe.dev/db/generated"
)

func TestExportGistHTML_EmptyConversation(t *testing.T) {
	conv := &generated.Conversation{
		ConversationID: "c1",
		Slug:           ptr("test-slug"),
		Model:          ptr("claude-opus-4"),
		Cwd:            ptr("/tmp/work"),
		CreatedAt:      time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
	}
	html, err := exportGistHTML(conv, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, "<title>test-slug") {
		t.Error("missing title")
	}
	if !strings.Contains(html, "session-data") {
		t.Error("missing session-data element")
	}
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("missing doctype")
	}
}

func TestExportGistHTML_NilFields(t *testing.T) {
	// No slug, no model, no cwd — should not panic.
	conv := &generated.Conversation{
		ConversationID: "c2",
		CreatedAt:      time.Now(),
	}
	html, err := exportGistHTML(conv, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, "Session") {
		t.Error("missing fallback title")
	}
}

// --- Edge case messages that occur in real agent sessions ---

func TestExportGistHTML_MalformedLlmData(t *testing.T) {
	conv := &generated.Conversation{
		ConversationID: "c3",
		Slug:           ptr("edge-cases"),
		CreatedAt:      time.Now(),
	}
	bad := "not json at all"
	msgs := []generated.Message{
		{MessageID: "m1", ConversationID: "c3", SequenceID: 1, Type: "agent", LlmData: &bad, CreatedAt: time.Now()},
	}
	html, err := exportGistHTML(conv, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should produce valid HTML — the client JS handles malformed data gracefully.
	if !strings.Contains(html, "session-data") {
		t.Error("missing session-data")
	}
}

func TestExportGistHTML_NilLlmData(t *testing.T) {
	conv := &generated.Conversation{
		ConversationID: "c4", Slug: ptr("nil-llm"), CreatedAt: time.Now(),
	}
	msgs := []generated.Message{
		{MessageID: "m1", ConversationID: "c4", SequenceID: 1, Type: "user", LlmData: nil, CreatedAt: time.Now()},
	}
	_, err := exportGistHTML(conv, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportGistHTML_ToolCallWithNilInput(t *testing.T) {
	conv := &generated.Conversation{
		ConversationID: "c5", Slug: ptr("tool-nil"), CreatedAt: time.Now(),
	}
	// ToolUse block with nil ToolInput.
	llmData := `{"Content":[{"Type":5,"ToolName":"bash","ToolInput":null}]}`
	msgs := []generated.Message{
		{MessageID: "m1", ConversationID: "c5", SequenceID: 1, Type: "agent", LlmData: &llmData, CreatedAt: time.Now()},
	}
	_, err := exportGistHTML(conv, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportGistHTML_ToolCallWithStringInput(t *testing.T) {
	conv := &generated.Conversation{
		ConversationID: "c6", Slug: ptr("tool-string"), CreatedAt: time.Now(),
	}
	// ToolInput as a raw string (not JSON object) — some tools do this.
	llmData := `{"Content":[{"Type":5,"ToolName":"patch","ToolInput":"hello world"}]}`
	msgs := []generated.Message{
		{MessageID: "m1", ConversationID: "c6", SequenceID: 1, Type: "agent", LlmData: &llmData, CreatedAt: time.Now()},
	}
	html, err := exportGistHTML(conv, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, "session-data") {
		t.Error("missing session-data")
	}
}

func TestExportGistHTML_ToolResultError(t *testing.T) {
	conv := &generated.Conversation{
		ConversationID: "c7", Slug: ptr("tool-err"), CreatedAt: time.Now(),
	}
	llmData := `{"Content":[{"Type":6,"ToolResult":[{"Type":2,"Text":"permission denied"}],"ToolError":true}]}`
	msgs := []generated.Message{
		{MessageID: "m1", ConversationID: "c7", SequenceID: 1, Type: "agent", LlmData: &llmData, CreatedAt: time.Now()},
	}
	_, err := exportGistHTML(conv, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportGistHTML_ToolResultEmptyContent(t *testing.T) {
	conv := &generated.Conversation{
		ConversationID: "c8", Slug: ptr("tool-empty"), CreatedAt: time.Now(),
	}
	llmData := `{"Content":[{"Type":6,"ToolResult":[]}]}`
	msgs := []generated.Message{
		{MessageID: "m1", ConversationID: "c8", SequenceID: 1, Type: "agent", LlmData: &llmData, CreatedAt: time.Now()},
	}
	_, err := exportGistHTML(conv, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportGistHTML_ToolResultNilContent(t *testing.T) {
	conv := &generated.Conversation{
		ConversationID: "c9", Slug: ptr("tool-nil-res"), CreatedAt: time.Now(),
	}
	llmData := `{"Content":[{"Type":6,"ToolResult":null}]}`
	msgs := []generated.Message{
		{MessageID: "m1", ConversationID: "c9", SequenceID: 1, Type: "agent", LlmData: &llmData, CreatedAt: time.Now()},
	}
	_, err := exportGistHTML(conv, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportGistHTML_ToolResultImageOnly(t *testing.T) {
	// Tool result with only an image (MediaType set, no Text) — screenshots we skip.
	conv := &generated.Conversation{
		ConversationID: "c10", Slug: ptr("tool-img"), CreatedAt: time.Now(),
	}
	llmData := `{"Content":[{"Type":6,"ToolResult":[{"Type":6,"MediaType":"image/png","DisplayImageURL":"/api/read/fake"}]}]}`
	msgs := []generated.Message{
		{MessageID: "m1", ConversationID: "c10", SequenceID: 1, Type: "agent", LlmData: &llmData, CreatedAt: time.Now()},
	}
	_, err := exportGistHTML(conv, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportGistHTML_RedactedThinking(t *testing.T) {
	conv := &generated.Conversation{
		ConversationID: "c11", Slug: ptr("redacted"), CreatedAt: time.Now(),
	}
	llmData := `{"Content":[{"Type":4,"Text":"[redacted]"},{"Type":2,"Text":"Done."}]}`
	msgs := []generated.Message{
		{MessageID: "m1", ConversationID: "c11", SequenceID: 1, Type: "agent", LlmData: &llmData, CreatedAt: time.Now()},
	}
	_, err := exportGistHTML(conv, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportGistHTML_EmptyContentArray(t *testing.T) {
	conv := &generated.Conversation{
		ConversationID: "c12", Slug: ptr("empty-content"), CreatedAt: time.Now(),
	}
	llmData := `{"Content":[]}`
	msgs := []generated.Message{
		{MessageID: "m1", ConversationID: "c12", SequenceID: 1, Type: "agent", LlmData: &llmData, CreatedAt: time.Now()},
	}
	_, err := exportGistHTML(conv, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportGistHTML_MessageTypes(t *testing.T) {
	conv := &generated.Conversation{
		ConversationID: "c13", Slug: ptr("all-types"), CreatedAt: time.Now(),
	}
	textLLM := `{"Content":[{"Type":2,"Text":"Hello"}]}`
	msgs := []generated.Message{
		{MessageID: "m1", Type: "user", LlmData: &textLLM, CreatedAt: time.Now()},
		{MessageID: "m2", Type: "agent", LlmData: &textLLM, CreatedAt: time.Now()},
		{MessageID: "m3", Type: "error", UserData: ptr(`{"text":"boom"}`), CreatedAt: time.Now()},
		{MessageID: "m4", Type: "warning", UserData: ptr(`{"text":"careful"}`), CreatedAt: time.Now()},
		{MessageID: "m5", Type: "system", CreatedAt: time.Now()},
	}
	_, err := exportGistHTML(conv, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportGistHTML_ProducesDecodableBase64(t *testing.T) {
	conv := &generated.Conversation{
		ConversationID: "c14", Slug: ptr("decode-test"), CreatedAt: time.Now(),
	}
	textLLM := `{"Content":[{"Type":2,"Text":"Hello"}]}`
	msgs := []generated.Message{
		{MessageID: "m1", Type: "user", LlmData: &textLLM, CreatedAt: time.Now()},
		{MessageID: "m2", Type: "agent", LlmData: &textLLM, CreatedAt: time.Now()},
	}
	html, err := exportGistHTML(conv, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Extract base64 from <script id="session-data">
	start := strings.Index(html, `id="session-data" type="application/json">`)
	if start == -1 {
		t.Fatal("missing session-data tag")
	}
	start += len(`id="session-data" type="application/json">`)
	end := strings.Index(html[start:], "</script>")
	if end == -1 {
		t.Fatal("missing closing script tag")
	}
	b64 := strings.TrimSpace(html[start : start+end])

	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}

	var data gistData
	if err := json.Unmarshal(decoded, &data); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if len(data.Messages) != 2 {
		t.Errorf("got %d messages, want 2", len(data.Messages))
	}
	if data.Conversation.Slug != "decode-test" {
		t.Errorf("slug = %q, want decode-test", data.Conversation.Slug)
	}
}
