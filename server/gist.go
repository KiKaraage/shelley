package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const gistHostBase = "https://gisthost.github.io/"

// gistHostURL returns the user-facing URL that renders the gist HTML directly.
func gistHostURL(gistID string) string {
	return gistHostBase + "?" + gistID
}

// isGenericSlug returns true if the slug is missing, blank, or one of the
// placeholder names that shouldn't be used as a gist filename.
func isGenericSlug(slug *string) bool {
	if slug == nil || *slug == "" {
		return true
	}
	s := strings.ToLower(strings.TrimSpace(*slug))
	if s == "" || s == "untitled" || s == "draft" || s == "shelley" {
		return true
	}
	return strings.HasPrefix(s, "untitled ") // "Untitled 3" etc.
}

// gistFilename returns the filename used inside the gist. Named index.html
// so that gisthost.github.io renders the HTML directly.
func gistFilename(slug *string) string {
	return "index.html"
}

// writeGistTmp writes html to a temp dir and returns the path.
func writeGistTmp(html string) (string, error) {
	dir, err := os.MkdirTemp("", "shelley-gist-*")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "index.html")
	if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return path, nil
}

// createGist creates a new secret GitHub gist via the gh CLI and returns
// the gist ID and the gisthost URL.
func createGist(ctx context.Context, slug *string, html string) (gistID, gistURL string, err error) {
	path, err := writeGistTmp(html)
	if err != nil {
		return "", "", fmt.Errorf("create temp: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(path))

	desc := "Shelley session"
	if slug != nil && *slug != "" {
		desc = "Shelley: " + *slug
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "gist", "create",
		"--desc", desc,
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", "", fmt.Errorf("gh gist create failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		if ctx.Err() == context.DeadlineExceeded {
			return "", "", fmt.Errorf("gh gist create timed out")
		}
		return "", "", fmt.Errorf("gh gist create: %w", err)
	}

	// gh gist create prints the gist URL on success.
	raw := strings.TrimSpace(string(out))
	// Extract gist ID from URL: https://gist.github.com/{user}/{id}
	parts := strings.Split(raw, "/")
	if len(parts) > 0 {
		gistID = parts[len(parts)-1]
		if strings.Contains(gistID, "?") {
			gistID = gistID[:strings.Index(gistID, "?")]
		}
	}
	if gistID == "" {
		return "", "", fmt.Errorf("could not parse gist ID from: %s", raw)
	}
	return gistID, gistHostURL(gistID), nil
}

// updateGist updates an existing gist via the gh CLI.
// gh gist edit <id> <file> replaces the gist content with the file.
func updateGist(ctx context.Context, gistID string, slug *string, html string) error {
	path, err := writeGistTmp(html)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(path))

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "gist", "edit", gistID,
		"--filename", "index.html",
		path,
	)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("gh gist edit failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("gh gist edit timed out")
		}
		return fmt.Errorf("gh gist edit: %w", err)
	}
	return nil
}

// gistError wraps an error with a code the UI can detect for targeted messaging.
type gistError struct {
	msg  string
	code string // "gh_not_auth"
}

func (e *gistError) Error() string { return e.msg }

// classifyGHErr inspects a gh CLI error and classifies it.
func classifyGHErr(err error) *gistError {
	if err == nil {
		return nil
	}
	s := err.Error()
	if strings.Contains(s, "not logged in") || strings.Contains(s, "authentication required") {
		return &gistError{msg: "gh not authenticated — run: gh auth login", code: "gh_not_auth"}
	}
	return &gistError{msg: s, code: ""}
}
