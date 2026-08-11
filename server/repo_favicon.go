package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// faviconCandidates lists well-known favicon paths checked in order.
var faviconCandidates = []string{
	"favicon.ico",
	"favicon.png",
	"favicon.svg",
	"favicon-32x32.png",
	"favicon-16x16.png",
	"apple-touch-icon.png",
	"apple-touch-icon-precomposed.png",
	"android-chrome-192x192.png",
	"android-chrome-512x512.png",
}

// iconSourceFiles lists HTML files that may declare a <link rel="icon">.
var iconSourceFiles = []string{
	"index.html",
	"index.htm",
}

var (
	linkIconHTMLRe = regexp.MustCompile(`(?i)<link[^>]+rel=["\'](?:shortcut )?icon["\'][^>]*>`)
	iconRelRe      = regexp.MustCompile(`(?i)rel=["\'](?:shortcut )?icon["\']`)
	iconHrefRe     = regexp.MustCompile(`(?i)href=["\']([^"\']+)["\']`)
)

// resolveRepoFavicon resolves a favicon file inside repoRoot.
// It returns the absolute path and true if found.
func resolveRepoFavicon(repoRoot string) (string, bool) {
	// 1. Check well-known candidate paths.
	for _, candidate := range faviconCandidates {
		path := filepath.Join(repoRoot, candidate)
		info, err := os.Stat(path)
		if err == nil && info.Mode().IsRegular() {
			return path, true
		}
	}

	// 2. Parse HTML source files for <link rel="icon" href="...">.
	for _, src := range iconSourceFiles {
		path := filepath.Join(repoRoot, src)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)
		matches := linkIconHTMLRe.FindAllString(content, -1)
		for _, m := range matches {
			if !iconRelRe.MatchString(m) {
				continue
			}
			hrefMatch := iconHrefRe.FindStringSubmatch(m)
			if hrefMatch == nil {
				continue
			}
			href := hrefMatch[1]
			if strings.HasPrefix(href, "/") || strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
				continue // absolute URLs aren't local
			}
			resolved := filepath.Join(repoRoot, filepath.Dir(src), href)
			info, err := os.Stat(resolved)
			if err == nil && info.Mode().IsRegular() {
				return resolved, true
			}
		}
	}

	return "", false
}

// faviconContentType returns a Content-Type for a favicon file path.
func faviconContentType(path string) string {
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".ico":
		return "image/x-icon"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}

// handleRepoFavicon serves a favicon resolved from a git repo root.
func (s *Server) handleRepoFavicon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	repoRoot := r.URL.Query().Get("root")
	if repoRoot == "" {
		http.Error(w, "root required", http.StatusBadRequest)
		return
	}

	repoRoot = filepath.Clean(repoRoot)
	if !filepath.IsAbs(repoRoot) {
		http.Error(w, "absolute path required", http.StatusBadRequest)
		return
	}

	resolved, ok := resolveRepoFavicon(repoRoot)
	if !ok {
		http.Error(w, "favicon not found", http.StatusNotFound)
		return
	}

	// Security: ensure the resolved path is inside repoRoot.
	realRepoRoot, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		http.Error(w, "repo path not accessible", http.StatusForbidden)
		return
	}
	realResolved, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		http.Error(w, "favicon not accessible", http.StatusNotFound)
		return
	}
	if !strings.HasPrefix(realResolved, realRepoRoot+string(filepath.Separator)) && realResolved != realRepoRoot {
		http.Error(w, "favicon not in repo", http.StatusForbidden)
		return
	}

	f, err := os.Open(resolved)
	if err != nil {
		http.Error(w, fmt.Sprintf("open failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", faviconContentType(resolved))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	io.Copy(w, f)
}
