package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRepoFavicon_ExactMatch(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "favicon.png"), []byte("fake"), 0o644)
	path, ok := resolveRepoFavicon(dir)
	if !ok || filepath.Base(path) != "favicon.png" {
		t.Fatalf("expected favicon.png, got %q, ok=%v", path, ok)
	}
}

func TestResolveRepoFavicon_HTMLLinkIcon(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	os.WriteFile(filepath.Join(dir, "assets", "icon.svg"), []byte("<svg/>"), 0o644)
	os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<html><link rel="icon" href="assets/icon.svg"/></html>`), 0o644)
	path, ok := resolveRepoFavicon(dir)
	if !ok || filepath.Base(path) != "icon.svg" {
		t.Fatalf("expected assets/icon.svg, got %q, ok=%v", path, ok)
	}
}

func TestResolveRepoFavicon_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, ok := resolveRepoFavicon(dir)
	if ok {
		t.Fatal("expected no favicon found")
	}
}

func TestResolveRepoFavicon_Priority(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "favicon.ico"), []byte("ico"), 0o644)
	os.WriteFile(filepath.Join(dir, "favicon.png"), []byte("png"), 0o644)
	path, ok := resolveRepoFavicon(dir)
	if !ok || filepath.Base(path) != "favicon.ico" {
		t.Fatalf("expected favicon.ico (first candidate), got %q, ok=%v", path, ok)
	}
}
