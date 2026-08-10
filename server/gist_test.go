package server

import (
	"fmt"
	"testing"
)

func TestIsGenericSlug(t *testing.T) {
	tests := []struct {
		name  string
		slug  *string
		wants bool
	}{
		{"nil", nil, true},
		{"empty", ptr(""), true},
		{"whitespace", ptr("   "), true},
		{"untitled lower", ptr("untitled"), true},
		{"untitled mixed", ptr("Untitled"), true},
		{"untitled with number", ptr("Untitled 3"), true},
		{"draft", ptr("Draft"), true},
		{"draft lower", ptr("draft"), true},
		{"shelley", ptr("Shelley"), true},
		{"valid name", ptr("my-project"), false},
		{"valid session", ptr("Debugging auth bug"), false},
		{"shelley prefix but different", ptr("Shelley-v2"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGenericSlug(tt.slug)
			if got != tt.wants {
				t.Errorf("isGenericSlug(%v) = %v, want %v", tt.slug, got, tt.wants)
			}
		})
	}
}

func TestGistFilename(t *testing.T) {
	got := gistFilename(ptr("anything"))
	if got != "index.html" {
		t.Errorf("gistFilename(...) = %q, want %q", got, "index.html")
	}
	got = gistFilename(nil)
	if got != "index.html" {
		t.Errorf("gistFilename(nil) = %q, want %q", got, "index.html")
	}
}

func TestClassifyGHErr(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if got := classifyGHErr(nil); got != nil {
			t.Errorf("classifyGHErr(nil) = %v, want nil", got)
		}
	})
	t.Run("not authenticated", func(t *testing.T) {
		got := classifyGHErr(fmt.Errorf("not logged in to github.com"))
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.code != "gh_not_auth" {
			t.Errorf("code = %q, want gh_not_auth", got.code)
		}
	})
	t.Run("auth required", func(t *testing.T) {
		got := classifyGHErr(fmt.Errorf("authentication required"))
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.code != "gh_not_auth" {
			t.Errorf("code = %q, want gh_not_auth", got.code)
		}
	})
	t.Run("other error", func(t *testing.T) {
		got := classifyGHErr(fmt.Errorf("network timeout"))
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.code != "" {
			t.Errorf("code = %q, want empty", got.code)
		}
	})
}

func ptr(s string) *string { return &s }
