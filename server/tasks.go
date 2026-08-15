package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"shelley.exe.dev/db"
	"shelley.exe.dev/gitstate"
)

// TaskResponse is the JSON shape returned to the UI for a task.
type TaskResponse struct {
	TaskID     string   `json:"task_id"`
	Title      string   `json:"title"`
	Cwd        *string  `json:"cwd"`
	Tags       []string `json:"tags"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
	Handled    bool     `json:"handled"`
	HandledAt  *string  `json:"handled_at"`
	ThreadSlug *string  `json:"thread_slug"`
	// Missing is true when the task's cwd no longer exists on disk.
	Missing bool `json:"missing"`
	// GitRemote is the owner/repo slug of the task's cwd, when it's a git repo.
	GitRemote string `json:"git_remote,omitempty"`
}

// taskToResponse converts a db.Task to the JSON response shape.
func taskToResponse(t *db.Task, threadSlug *string) TaskResponse {
	resp := TaskResponse{
		TaskID:     t.TaskID,
		Title:      t.Title,
		Cwd:        t.Cwd,
		Tags:       t.Tags,
		CreatedAt:  t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  t.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Handled:    t.Handled,
		ThreadSlug: threadSlug,
	}
	if t.HandledAt != nil {
		hs := t.HandledAt.UTC().Format("2006-01-02T15:04:05Z")
		resp.HandledAt = &hs
	}
	if t.Cwd != nil && *t.Cwd != "" {
		if info, err := os.Stat(*t.Cwd); err != nil || !info.IsDir() {
			resp.Missing = true
		} else if gs := gitstate.GetGitState(*t.Cwd); gs != nil && gs.IsRepo {
			resp.GitRemote = gs.RemoteSlug
		}
	}
	return resp
}

// threadSlugForTask resolves the slug of the first linked conversation, if any.
func (s *Server) threadSlugForTask(ctx context.Context, taskID string) *string {
	links, err := s.db.ListTaskConversations(ctx, taskID)
	if err != nil || len(links) == 0 {
		return nil
	}
	conv, err := s.db.GetConversationByID(ctx, links[0].ConversationID)
	if err != nil || conv == nil || conv.Slug == nil {
		return nil
	}
	return conv.Slug
}

// handleTasks dispatches GET /api/tasks and POST /api/tasks.
func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListTasks(w, r)
	case http.MethodPost:
		s.handleCreateTask(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTask dispatches per-ID task routes: GET/PATCH/DELETE /api/tasks/{id}.
func (s *Server) handleTask(w http.ResponseWriter, r *http.Request) {
	// Strip the /api/tasks/ prefix to get the task ID.
	id := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleGetTask(w, r, id)
	case http.MethodPatch:
		s.handleUpdateTask(w, r, id)
	case http.MethodDelete:
		s.handleDeleteTask(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListTasks handles GET /api/tasks
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tasks, err := s.db.ListTasks(r.Context())
	if err != nil {
		s.logger.Error("Failed to list tasks", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	out := make([]TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		var slug *string
		if t.Handled {
			slug = s.threadSlugForTask(r.Context(), t.TaskID)
		}
		out = append(out, taskToResponse(t, slug))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// TaskRequest is the create/update request body.
type TaskRequest struct {
	Title string   `json:"title"`
	Cwd   *string  `json:"cwd"`
	Tags  []string `json:"tags"`
}

// cleanCwd validates and cleans a task's target directory. Returns an error
// message if the path is invalid; otherwise returns the cleaned absolute path.
func cleanCwd(cwd *string) (*string, string) {
	if cwd == nil || *cwd == "" {
		return nil, ""
	}
	p := strings.TrimSpace(*cwd)
	if !filepath.IsAbs(p) {
		return nil, "cwd must be an absolute path"
	}
	cleaned := filepath.Clean(p)
	if info, err := os.Stat(cleaned); err != nil || !info.IsDir() {
		// Missing-on-disk is render-only per spec; still store it but flag it.
		return &cleaned, ""
	}
	return &cleaned, ""
}

// handleCreateTask handles POST /api/tasks
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	cwd, msg := cleanCwd(req.Cwd)
	if msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	tags := normalizeTags(req.Tags)
	t, err := s.db.CreateTask(r.Context(), "", title, cwd, tags)
	if err != nil {
		s.logger.Error("Failed to create task", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(taskToResponse(t, nil))
}

// handleGetTask handles GET /api/tasks/{id}
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request, taskID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	t, err := s.db.GetTask(r.Context(), taskID)
	if err != nil {
		s.logger.Error("Failed to get task", "taskID", taskID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	var slug *string
	if t.Handled {
		slug = s.threadSlugForTask(r.Context(), taskID)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(taskToResponse(t, slug))
}

// handleUpdateTask handles PATCH /api/tasks/{id}
func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request, taskID string) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	existing, err := s.db.GetTask(r.Context(), taskID)
	if err != nil {
		s.logger.Error("Failed to get task for update", "taskID", taskID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	title := existing.Title
	if req.Title != "" {
		title = strings.TrimSpace(req.Title)
	}
	cwd := existing.Cwd
	if req.Cwd != nil {
		cleaned, msg := cleanCwd(req.Cwd)
		if msg != "" {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		cwd = cleaned
	}
	t, err := s.db.UpdateTask(r.Context(), taskID, title, cwd)
	if err != nil {
		s.logger.Error("Failed to update task", "taskID", taskID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if req.Tags != nil {
		tags := normalizeTags(req.Tags)
		t, err = s.db.UpdateTaskTags(r.Context(), taskID, tags)
		if err != nil {
			s.logger.Error("Failed to update task tags", "taskID", taskID, "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
	var slug *string
	if t.Handled {
		slug = s.threadSlugForTask(r.Context(), taskID)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(taskToResponse(t, slug))
}

// handleDeleteTask handles DELETE /api/tasks/{id}
func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request, taskID string) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.db.DeleteTask(r.Context(), taskID); err != nil {
		s.logger.Error("Failed to delete task", "taskID", taskID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "task_id": taskID})
}

// TaskDirectoriesResponse is the response for the task directory dropdown.
type TaskDirectoriesResponse struct {
	GitRoots []string `json:"git_roots"`
	Cwds     []string `json:"cwds"`
	// RepoNames maps a git root path to its owner/repo slug (e.g.
	// "kikaraage/shelley"), when the repo has a hosted remote.
	RepoNames map[string]string `json:"repo_names,omitempty"`
}

// directoryOptions returns the distinct past cwds (from conversations and
// tasks), most recently used first, deduplicated by git repo. A git worktree
// (or any cwd inside a repo) collapses to its main repo root so each repo
// appears once; non-repo cwds stay as-is. This deliberately avoids crawling
// the whole filesystem: we only surface directories the user has actually
// worked in.
func (s *Server) directoryOptions(ctx context.Context) []string {
	cwds, err := s.db.ListDistinctCwds(ctx)
	if err != nil {
		s.logger.Error("Failed to list distinct cwds", "error", err)
		return nil
	}
	seen := make(map[string]bool, len(cwds))
	out := make([]string, 0, len(cwds))
	for _, c := range cwds {
		path := c
		if gs := gitstate.GetGitState(c); gs != nil && gs.IsRepo {
			if root := gitstate.MainRepoRoot(c); root != "" {
				path = root
			}
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return out
}

// gitRepoRoots returns the distinct past cwds that are git repositories, for
// the task modal's directory dropdown, deduplicated by main repo root.
func (s *Server) gitRepoRoots(ctx context.Context) []string {
	out := make([]string, 0)
	for _, c := range s.directoryOptions(ctx) {
		if gs := gitstate.GetGitState(c); gs != nil && gs.IsRepo {
			out = append(out, c)
		}
	}
	return out
}

// handleTaskDirectories handles GET /api/tasks/directories
// Returns git roots (from the git repos crawl) plus distinct past cwds
// (from conversations and tasks), for the task modal's directory dropdown.
func (s *Server) handleTaskDirectories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	// Git roots: deduplicated by main repo root.
	gitRoots := s.gitRepoRoots(ctx)

	// Distinct past cwds, deduplicated by main repo root.
	cwds := s.directoryOptions(ctx)

	// Map each git root to its owner/repo slug for pill labels.
	repoNames := make(map[string]string, len(gitRoots))
	for _, g := range gitRoots {
		if gs := gitstate.GetGitState(g); gs != nil && gs.RemoteSlug != "" {
			repoNames[g] = gs.RemoteSlug
		}
	}

	resp := TaskDirectoriesResponse{GitRoots: gitRoots, Cwds: cwds, RepoNames: repoNames}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
