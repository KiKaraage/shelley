package server

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"shelley.exe.dev/db/generated"
)

//go:embed embed/gist.html.tmpl
var gistHTMLTemplate string

// gistData is the JSON payload embedded in the HTML for client-side rendering.
type gistData struct {
	Conversation gistDataConversation `json:"conversation"`
	Messages     []gistDataMessage    `json:"messages"`
}

type gistDataConversation struct {
	Slug  string `json:"slug"`
	Model string `json:"model"`
	Cwd   string `json:"cwd"`
	Date  string `json:"date"`
}

type gistDataMessage struct {
	Type    string  `json:"type"`
	LlmData *string `json:"llm_data,omitempty"`
	UserData *string `json:"user_data,omitempty"`
	Created string  `json:"created"`
}

// exportGistHTML builds a self-contained HTML page for a conversation.
func exportGistHTML(conv *generated.Conversation, messages []generated.Message) (string, error) {
	slug := ""
	if conv.Slug != nil {
		slug = *conv.Slug
	}
	model := ""
	if conv.Model != nil {
		model = *conv.Model
	}
	cwd := ""
	if conv.Cwd != nil {
		cwd = *conv.Cwd
	}

	data := gistData{
		Conversation: gistDataConversation{
			Slug:  slug,
			Model: model,
			Cwd:   cwd,
			Date:  conv.CreatedAt.Format(time.RFC3339),
		},
	}
	for _, m := range messages {
		gm := gistDataMessage{
			Type:    m.Type,
			LlmData: m.LlmData,
			UserData: m.UserData,
			Created: m.CreatedAt.Format(time.RFC3339),
		}
		data.Messages = append(data.Messages, gm)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal gist data: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(jsonData)

	tmpl, err := template.New("gist").Parse(gistHTMLTemplate)
	if err != nil {
		return "", fmt.Errorf("parse gist template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]string{
		"EncodedData": encoded,
		"Slug":        slug,
	}); err != nil {
		return "", fmt.Errorf("execute gist template: %w", err)
	}
	return buf.String(), nil
}
