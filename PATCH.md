# PATCH.md: Shelley customizations

`Base` = short SHA of the last upstream commit before the change (upstream SHAs are stable across rebases; ours are not).
`Status`: active | superseded | reverted. `Intent`/`Watchouts` only when not self-explanatory.

## Quick start

- **Check state**: expect clean working tree on branch `ki`:
  ```sh
  cd ~/.config/shelley/shelley-customization
  git status --short && git branch --show-current
  ```
- **Check upstream**: rebase if `origin/main` is ahead:
  ```sh
  git fetch origin main --tags
  git merge-base HEAD origin/main # current upstream base
  ```
- **Build & install**: UI + templates + custom ldflags stamp, installed to `~/.local/bin/shelley`:
  ```sh
  make build-custom
  bin/shelley version # verify: -ki.<sha>, customized:true
  ```
- **Restart**: delayed so it doesn't kill the current turn (no tmux installed):
  ```sh
  setsid bash -c 'sleep 5; systemctl --user restart shelley' >/dev/null 2>&1 < /dev/null &
  ```

## Rules

- Before starting work, assign a patch number first (P123, can be a new patch or an update to existing patch). This will be used as commit prefix
- Keep the changes minimum, minimize DB migrations/schema change when possible
- Never plain `make build`: only the `build-custom` ldflags stamp; never restart shelley mid-turn: always delayed via `setsid`
- Never anchor to our own commit SHAs (rewritten on rebase), use upstream `Base` SHAs
- Never amend/smash commit if your ongoing patch number is different than the latest previous commit

## P001
Move Diffs, Git Graph, Terminal into always-visible header buttons
- Status: superseded
- Base: 5c2cce3
- Files: ChatInterface.vue, ChatOverflowMenu.vue, styles.css
- Changes:
  - Moved Diffs, Git Graph, Terminal from the overflow menu into always-visible header buttons, ordered [+, Diffs, Git Graph, Terminal, ⋮]; removed the menu items, their emits, and the now-unused hasCwd prop. Also removed the separators around Archive Conversation and put the theme and notification switches on one row in the overflow menu (split 60:40 — theme has 3 options, notifications 2).
- Superseded by: upstream moved the Diffs/Git Graph/Terminal header buttons into the header itself (they live in both header and overflow upstream). The custom commits for this patch were dropped from history during the v0.968 rebase; the tooltips it introduced for the +/⋮/☰ buttons now live under P002.

## P002
Add tooltips to New Thread, overflow menu, hamburger, and drawer buttons
- Status: active
- Base: 1d4cbe7
- Files: ChatInterface.vue, ChatOverflowMenu.vue, ConversationDrawer.vue
- Changes:
  - Added tooltips to the New Thread (+), overflow menu (⋮), and mobile hamburger (☰) header buttons, plus the mobile drawer's + and x buttons.

## P003
Launch zsh instead of bash in the in-app terminal
- Status: active
- Base: 1d4cbe7
- Files: ChatInterface.vue
- Changes:
  - In-app terminal now launches zsh instead of bash (`exec zsh -i`). The server wraps the command as `bash --login -c '<cmd>'`, but `exec zsh -i` replaces bash with zsh.

## P004
Hide working directory/command sections in BashTool; add Copy Command/Results buttons
- Status: active
- Base: 1d4cbe7
- Files: BashTool.vue, styles.css
- Changes:
  - Bash tool summary: Remove working directory label, only show truncated command + tool call status icon + chevron toggle
  - Bash tool results: Hide the "Working Directory" and "Command" sections; added "Copy Command" and "Copy Results" buttons to the right of the Output header.
  - Added `max-height: 350px; overflow: auto` to `.tool-result-content` (generic fallback card), `.bash-tool-code`, `.bash-tool-streaming`, `.bash-tool-preview-code`, `.tool-code` (GenericTool), `.patch-tool-raw-diff` and `.patch-tool-diff` so long tool outputs scroll instead of stretching the page. Modal overrides (`.tool-detail-modal .bash-tool-code` / `.tool-code`) reset to `max-height: none; overflow: visible` so the detail modal isn't clipped.
  - `.bash-tool-copy-btn` background changed to `var(--gray-800)` in dark mode and gained `transition: background-color 0.15s ease`.

## P005
Add `~/.agents/skills/` to skill discovery paths
- Status: active
- Base: 1d4cbe7
- Files: skills/skills.go, skills/skills_test.go
- Changes:
  - Added ~/.agents/skills/ to DefaultDirs discovery paths so Shelley recognizes skills installed by other agent tools (Claude Code, etc.). The directory is checked alongside ~/.config/shelley/, ~/.config/agents/skills/, and ~/.shelley/.

## P006
Make git commit attribution a user-configurable dropdown (co-author, assisted-by, off)
- Status: active
- Base: 1d4cbe7
- Files: claudetool/bash.go, claudetool/shell.go, claudetool/toolset.go, claudetool/env.go, server/convo.go, server/handlers.go, cmd/shelley/main.go, ui/src/vue/components/ChatOverflowMenu.vue, ui/src/i18n/en.ts (and other locale files), ui/src/i18n/types.ts, claudetool/bash_test.go
- Changes:
  - "Co-authored-by: Shelley" (same as before)
  - "Assisted-by: <model> in Shelley" (derives model display name from active model via ModelInfo.DisplayName; no email)
  - "No agent attribution" (no trailer)
  - The setting is stored via the existing settings API (key: `shelley.attribution`, values: `co-author`, `assisted-by`, `off`). The legacy `git config shelley.no-trailer` path still works as a fallback when no explicit mode is set.
- Watchouts:
  - NewToolSet reads the setting fresh on each turn, so changes take effect on the next message. The `GitAttributionMode` type is shared between bash and shell tools. The `BashTool.resolvedAttribution()` method handles the legacy git config fallback; when an explicit mode is set via the DB setting, it takes precedence. `ShelleyEnv.ModelDisplayName` is populated from `ToolSetConfig.ResolveModelDisplayName()` which calls `LLMProvider.GetModelInfo()`; nil-safe and falls back to the raw model ID.

## P007
Add custom model enable/disable, import from /v1/models, and context window fix
- Status: active
- Base: 1d4cbe7
- Files: db/schema/038-model-enabled.sql, db/schema/039-model-context-window.sql, db/query/models.sql, db/generated/models.sql.go, db/generated/models.go, db/db.go, models/models.go, models/models_test.go, llm/oai/oai.go, llm/oai/oai_responses.go, llm/gem/gem.go, llm/ant/ant.go, server/custom_models.go, server/server.go, ui/src/services/api.ts, ui/src/vue/components/ModelsModal.vue, ui/src/vue/components/ModelFormModal.vue, ui/src/vue/components/ImportModelsModal.vue (new), ui/src/vue/components/customModelConstants.ts, ui/src/styles.css, ui/src/i18n/en.ts (and other locale files), ui/src/i18n/types.ts
- Changes:
  - **Enable/disable toggle**: new `enabled` column on `models` (default 1). The runtime `Manager.customModelRows()` now calls `GetEnabledModels` (WHERE enabled=1) so disabled models are hidden from the model picker and conversation loop, but remain visible in Manage Models for re-enabling. A ToggleSwitch column was added to the Manage Models table; toggling calls `PUT /api/custom-models/{id}` with `{enabled}`.
  - **Import from /v1/models**: new `POST /api/custom-models/import` endpoint. Accepts `{provider_type: "openai", endpoint, api_key}`, fetches `{endpoint}/models` with `Authorization: Bearer {api_key}`, parses the OpenAI-compatible `{data: [{id, name}]}` response, and bulk-inserts net-new models (skipping any whose `endpoint|model_name` already exists). Uses the `name` field for display_name when present, else the `id`. Imported models default to enabled, image_support=auto, reasoning_support=auto, context_window=0 (auto-detect). A new "Import Models" button in the Manage Models header opens a stacked `ImportModelsModal.vue` (endpoint + API key form, reports imported/skipped counts). The import dialog expects the base endpoint URL (e.g. `https://api.openai.com`) and the server appends `/v1/models` itself.
  - **Partial update fix**: `PUT /api/custom-models/{id}` treats every unset field as "keep existing" so partial bodies (like the enable/disable toggle's `{enabled}`) don't wipe provider_type/endpoint/model_name/etc. or fail the provider_type CHECK constraint.
  - **Context window fix**: new `context_window` column on `models` (default 0). Each llm.Service struct (`oai.Service`, `oai.ResponsesService`, `gem.Service`, `ant.Service`) gained a `ContextWindow int` field; `TokenContextWindow()` returns it when non-zero, falling back to the existing hardcoded per-model switch. `createServiceFromModel()` sets `ContextWindow` from the DB column (falling back to `max_tokens` for pre-migration rows). The add/edit form (`ModelFormModal.vue`) now exposes a "Context Window" input alongside "Max Context Tokens" so users can set the real context limit that drives the UI meter, instead of the hardcoded 128k default that unknown models fell into.
  - **Usage cost and cache hit rate**: The status bar context-usage label shows the estimated cost (from the pricing lookup) alongside the token count (e.g. `76k $0.43`). The token marker in the chat timeline also shows per-call cost. The Usage Details modal (per-message) shows Cache Hit Rate = (cache_read + cache_write) / (cache_read + cache_write + input + output) * 100%. The pricing lookup runs when usage entries are first needed (hover/focus/click on the context bar).
- Watchouts:
  - `max_tokens` still controls max output/completion tokens (`max_completion_tokens` on the wire): it is NOT the context window. `context_window` is the total context size for the UI meter. Only OpenAI Chat Completions (`provider_type: "openai"`) is supported for import; Anthropic/Gemini/OpenAI Responses have different model-list APIs. The sqlc-generated files picked up a version bump (v1.30.0 -> v1.31.1) in their headers: harmless. The `customModelRows()` change means any test that creates a model without `Enabled: 1` will have it filtered from the runtime; `models_test.go` was updated to set `Enabled: 1`.

## P008
Import pricing from /v1/models into DB
- Status: active
- Base: 1d4cbe7
- Fixes:
  - resolveCost now matches imported-model pricing by endpoint prefix (usage URLs carry trailing segments like /chat/completions that never matched the base endpoint); path-boundary guard prevents hostname spoofing.
- Files: db/schema/040-model-pricing.sql, db/query/models.sql, db/generated/models.sql.go, db/generated/models.go, server/custom_models.go, server/model_costs.go
- Changes:
  - New migration adds `input_price`, `output_price`, `cache_read_price`, `cache_write_price` columns to `models` (all REAL, default 0).
  - Import (`POST /api/custom-models/import`) now parses the `pricing` field from the `/v1/models` response and stores per-million-token prices. When only `cache_prompt` is set (no `cache_write`), it is used for both cache read and cache write.
  - `POST /api/model-costs` resolves pricing by checking DB-stored custom model prices first, then falling back to the models.dev snapshot.
  - `PUT /api/custom-models/{id}` supports optional `input_price`, `output_price`, `cache_read_price`, `cache_write_price` fields (pointer types: omit to preserve existing).
  - Duplicate model preserves pricing from source.
- Watchouts:
  - Existing models in the DB will have 0 pricing: re-import from the provider to populate. The embedded models.dev snapshot also covers crof.ai models as a fallback.

## P009
Add optimistic toggle/delete + checkpoint selection bar to ModelsModal
- Status: active
- Base: 1d4cbe7
- Files: ui/src/vue/components/ModelsModal.vue, ui/src/styles.css, ui/src/i18n/en.ts (and other locale files), ui/src/i18n/types.ts
- Changes:
  - **Optimistic toggle/delete**: `handleToggleEnabled` and `handleDelete` now update local state immediately without a full `loadModels()` round-trip. Server errors roll back only the affected row. No more blocking wait between toggles.
  - **Checkpoint selection bar**: custom model rows get checkboxes; selecting one or more shows a sticky bar at the bottom with bulk Disable/Enable/Delete actions. All bulk actions are also optimistic with rollback.
  - **Sort enabled first**: custom models sort enabled before disabled in the table.
  - **Checkbox alignment**: header "select all" checkbox visually aligned with per-row checkboxes via PrimeVue `pt` pass-through + scoped CSS.
  - Added `disable`/`enable` i18n keys across all 9 locale files.
- Watchouts:
  - `bulkToggleEnabled` only clears `_pending` on the affected models (not all) to avoid interfering with concurrent individual toggles. `handleDelete` re-finds the index on rollback to handle concurrent array shifts. `bulkDelete` and `handleDelete` restore `selectedKeys` on error.

## P010
Use 24h for all time displays
- Status: active
- Base: 1d4cbe7
- Files: ui/src/utils/messageTime.ts, ui/src/utils/messageTime.test.ts, ui/src/vue/components/ConversationDrawer.vue, ui/src/vue/components/MessageInfoModal.vue, ui/src/vue/components/UsageDetailModal.vue, ui/src/vue/components/VersionChecker.vue, ui/src/vue/components/TokenCostGraph.vue, ui/src/vue/components/GitGraphViewer.vue, ui/src/vue/components/GitRepoPicker.vue, ui/src/utils/conversationMarkdown.ts
- Changes:
  - All time displays now use 24-hour format instead of 12-hour. Added `hour12: false` to the shared `messageTime.ts` formatters (message timestamps, absolute timestamps) and to every other explicit time formatter: ConversationDrawer's today-timestamp, MessageInfoModal, UsageDetailModal, VersionChecker, TokenCostGraph hover time, GitGraphViewer commit date, GitRepoPicker tooltip, and conversationMarkdown's Started/Exported timestamps. The `hour` fields were bumped from `numeric` to `2-digit` in `messageTime.ts` so 24h times render zero-padded (e.g. `09:05` not `9:05`).
- Watchouts:
  - `hour12: false` forces 24h regardless of the user's locale; this intentionally overrides locale defaults. The day-only formatters (formatDay) were left unchanged: they don't render a time-of-day.

## P011
Show repo basename in conversation drawer CWD
- Status: active
- Base: 1d4cbe7
- Files: ConversationDrawer.vue
- Changes:
  - The conversation drawer's CWD display now shows only the repo basename (e.g. `shelley-customization`) instead of the tildified full path (`~/.config/shelley/shelley-customization`). The full path remains in the title tooltip. Also removed the now-unused `tildifyPath` import.

## P012
Switch 10 UI selectors from monospace to sans-serif font
- Status: active
- Base: 1d4cbe7
- Files: ui/src/styles.css
- Changes:
  - Switched several UI elements from monospace (`var(--font-mono)`) to sans-serif (`var(--font-sans)`): `.app-bar-title`, `.conversation-cwd` (also added `font-weight: bold`), `.model-bar-name`, `.system-prompt-label`, `.system-prompt-tools-label`, `.system-prompt-tool-name`, `.bash-tool-copy-btn`, `.status-message`, `.status-readout`, and `.animated-working`.

## P013
Hide bash tool output block when output is empty
- Status: active
- Base: a8e2ee0
- Files: BashTool.vue
- Changes:
  - When a bash tool call produces empty output, the output block is hidden and the label reads "Clean exit" instead of "Output:". When there is output, it still shows "Output" (or "Output (Error)") as before.

## P014
Export session as self-contained HTML to GitHub Gist
- Status: active
- Base: 1d4cbe7
- Files: db/schema/041-gist-id.sql, db/query/conversations.sql, db/generated/conversations.sql.go, db/generated/models.go, server/gist.go, server/gist_export_html.go, server/gist_handlers.go, server/embed/gist.html.tmpl, server/handlers.go, server/server.go, ui/src/services/api_gist.ts, ui/src/vue/components/ChatOverflowMenu.vue, ui/src/vue/components/ChatInterface.vue, ui/src/styles.css, ui/src/i18n/en.ts (and other locale files), ui/src/i18n/types.ts
- Changes:
  - Export Shelley session as a self-contained HTML file to a GitHub secret gist via the `gh` CLI.
  - **DB**: new `gist_id TEXT` column on `conversations` (migration 041).
  - **Backend**: `server/gist.go`: `gh gist create/edit` wrapper, slug validation (rejects `Untitled`, `Draft`, `Shelley`, empty), temp file management, error classification (`gh_not_auth`). `server/gist_export_html.go`: `exportGistHTML()` builds self-contained HTML with base64-encoded JSON data blob + embedded JS markdown renderer. `server/gist_handlers.go`: `GET/POST/PATCH /api/conversation/{id}/export-gist`. `server/server.go`: auto-update hook on both single-message and batch `isAgentEndOfTurn` paths.
  - **Frontend**: overflow menu item ("Export to Gist" / "Update Gist"), gist state tracking in ChatInterface, toast feedback, i18n in all 9 locales.
  - **HTML template**: `server/embed/gist.html.tmpl`: self-contained HTML with vanilla JS renderer. Bubbles reserved for user prompts and assistant text only. Thinking blocks, tool calls, and tool results render as standalone collapsible blocks between bubbles (not inside them).
  - **Auto-update**: after every agent end-of-turn (not during streaming/thinking/user cancel), gist is silently updated if one exists. Errors go to console only.
  - **URL format**: `https://gisthost.github.io/?{gist-id}`: renders the HTML directly.
  - **Tests**: `gist_test.go` (slug validation, error classification), `gist_export_html_test.go` (13 edge cases: nil fields, malformed JSON, tool calls with nil/string input, tool result errors, image-only results, etc.), `api_gist.test.ts` (plain-text error body handling).
- Watchouts:
  - Requires `gh` CLI installed and authenticated. The `init()` staleness check in `ui/embedfs.go` calls `os.Exit(1)` when build is stale: rebuild UI before testing. `html/template` was replaced with `text/template` for the embedded HTML to avoid escaping content inside `<script>` tags.

## P015
Make context warning threshold relative to model context window
- Status: superseded (upstream 48bfcc3 removed the ⚠️ icon and replaced the warning with color-coded token counts)
- Base: 1d4cbe7
- Files: ui/src/vue/components/ContextUsageBar.vue
- Changes:
  - Made the context warning icon (⚠️) threshold relative to the model's context window instead of a hardcoded 100k tokens. When `maxContextTokens` is known, the warning now triggers at 70% usage (matching the existing label color thresholds). Falls back to the absolute 100k threshold only when the model has no declared context window.
- Watchouts:
  - The `percentage` computed prop is already defined above the warning check so it can be reused directly. The auto-open popup feature (`hasAutoOpened` / localStorage gate) still uses the same condition.

## P016
Add settled thread preview in sidebar + auto-unarchive on send
- Status: active
- Base: e653372
- Files: ui/src/vue/components/ConversationDrawer.vue, ui/src/vue/components/ChatInterface.vue, ui/src/i18n/en.ts, ui/src/styles.css
- Changes:
  - Settled (archived) thread preview in sidebar + auto-unarchive on send.
  - **Main view**: up to 25 latest archived threads shown as compact one-line `.conversation-item-settled` rows below the active conversation list, separated by a `.conversation-list-separator` line. Each row shows thread name (left) + timestamp (right); on hover the timestamp fades out and Restore + Delete buttons overlay in-place using `.btn-icon-sm` icons matching active-row button sizing.
  - **Active highlight**: settled rows get the `.active` class (blue background) when the currently viewed conversation matches, with full opacity override.
  - **Click behavior**: clicking a settled thread opens it in the main chat view via `selectConversation`. Sending a new message to a settled thread auto-unarchives it (`api.unarchiveConversation`) and fires `onConversationUnarchived` to update the sidebar.
  - **Archived view**: toggled via "View Archived" footer button. Uses full `ConversationRow` layout (same as active threads) with all original buttons and actions. Header reads "Archived".
  - **i18n**: "Archive" action renamed to "Settle" in English (`archiveConversation`, `archiveConversationAction`, `archiveCurrentConversation`, `archive` keys).
  - **Eager loading**: archived conversations are loaded on mount for instant settled preview display.
- Watchouts:
  - Non-English locale files retain their original archive/restore translations (not renamed to "Settle"). The `api_gist.ts` pre-existing type errors are unrelated and predate this patch.

## P017
Move collapse/close buttons to drawer header left; rename title to "Shelley"
- Status: active
- Base: 1d4cbe7
- Files: ui/src/vue/components/ConversationDrawer.vue, ui/src/vue/components/ConversationDrawerRow.vue, ui/src/styles.css
- Changes:
  - Moved the collapse (<<) and close (x) buttons from the right side of the drawer header to the left edge, before the title. Renamed the drawer title from "Conversations" to "Shelley" (archived view still shows the "archived" translation). Both `.working-indicator` and `.drawer-working-indicator` now have a solid green background (`--green-500, #22c55e`) with no border. Tightened conversation item padding and gaps. Row structure: Row 1 = [favicon · cwd · gitRepoName]; Row 2 = [title] [hover-only action buttons] [working indicator] [subagent badge]; Row 3 = [date] [preview]. Action buttons live on Row 2 (right of title, left of working indicator) and collapse to zero width when not hovered. Git commit line removed from template. Action buttons (rename/edit tags/archive) only visible on hover via CSS opacity transition (always visible on mobile via `@media (max-width: 768px)` override). The main working indicator is hidden while any of the row's subagents are running (`runningSubagentCount > 0`), since the subagent-count badge already shows its own running ring in that case: reducing duplicate green-dot noise on the row. The indicator still appears on the subagent conversation item and inside the badge.

## P018
Show git branch + owner/repo slug in drawer meta row
- Status: active
- Base: 9c96638
- Files: gitstate/gitstate.go, gitstate/gitstate_test.go, server/server.go, server/handlers.go, cmd/go2ts.go, ui/src/generated-types.ts, ui/src/vue/components/ConversationDrawerRow.vue, ui/src/vue/components/ChatInterface.vue, ui/src/vue/components/GitBranchIcon.vue (new), ui/src/styles.css
- Changes:
  - **Second meta item = git branch name**: exposed the current branch (`gitstate.GitState.Branch`) as `git_branch` on `ConversationWithState` and render it after the repo slug, with a `repo · branch` tooltip. Falls back to the repo name when the branch is unavailable (e.g. detached HEAD).
  - **First meta item = owner/repo from the remote**: derived an `owner/repo` slug from the repo's remotes as `git_remote` on `ConversationWithState`. The slug resolves the remote this repo actually pushes to: priority: the current branch's upstream remote (e.g. your fork), then `upstream`, then `origin`. Read from the repo `config` file without shelling out to git (with a git-binary fallback). The first meta item shows this slug in place of the cwd folder name; the full cwd remains the tooltip.
  - **Git branch icon**: extracted the chat header's git-branch glyph into a shared `GitBranchIcon.vue` component, used in both the chat header and the drawer meta row (icon import over inline SVG). The drawer icon inherits the row's text color (including the white active state) at the same 0.8 opacity as the branch text.
  - **Branch name at regular font weight** (`font-weight: 400`); the meta row packs tighter (`gap: 0.25rem`).
  - **Drawer actions row collapses to zero width** when the conversation item isn't hovered (`max-width: 0; overflow: hidden`), transitioning open on hover instead of just fading opacity.
- Watchouts:
  - The slug prefers the branch's upstream remote so forks show the fork's owner/repo (e.g. `KiKaraage/shelley`) rather than the original. `gitstate.GitState.Equal` includes `RemoteSlug` so cache invalidation accounts for remote changes. The `git_branch`/`git_remote` fields were hand-added to `ui/src/generated-types.ts` because the local `go2ts` generator produces non-reproducible output (tabs + a `gist_id` field); regenerate carefully if you ever run it.

## P019
Install `build-custom` output to `~/.local/bin/shelley`
- Status: active
- Base: 1d4cbe7
- Files: Makefile
- Changes:
  - `build-custom` now installs the freshly built binary to `~/.local/bin/shelley` in addition to `bin/shelley`. It creates `~/.local/bin` if missing, then `install -m 0755` copies the binary there. Prints both the build and install confirmation lines.
- Watchouts:
  - `~/.local/bin` must be on `PATH` for the installed binary to be used. This is a convenience copy; the running service still restarts from the side-by-side+rename path in the Quick start, not from this install.

## P020
Show repo favicon in drawer meta row
- Status: active
- Base: 9c96638
- Files: server/repo_favicon.go (new), server/repo_favicon_test.go (new), server/server.go, ConversationDrawerRow.vue, styles.css
- Changes:
  - Added repo favicon to the drawer meta row: a small favicon image rendered before the repo name for conversations that live in a git repo.
  - **Server endpoint** `GET /api/repo-favicon?root=<repoRoot>`: resolves a favicon from the repo root by checking well-known paths (`favicon.ico`, `favicon.png`, `favicon.svg`, `favicon-32x32.png`, `favicon-16x16.png`, `apple-touch-icon.png`, `apple-touch-icon-precomposed.png`, `android-chrome-192x192.png`, `android-chrome-512x512.png`) in order, then parses `<link rel="icon">` declarations in `index.html` / `index.htm` via regex. Validates the resolved path stays inside the repo root via `EvalSymlinks`. Serves with correct content-type and `Cache-Control: max-age=3600`.
  - **Frontend** `ConversationDrawerRow.vue`: renders `<img class="drawer-favicon">` before the repo name span when `git_repo_root` is present. Uses `git_repo_root` (the worktree toplevel, not `git_worktree_root`) because that's the checked-out working directory where favicon files live. A `faviconFailed` ref hides the image on 404 (no favicon in repo). Resets on conversation change via `watch`.
  - **Tests** `repo_favicon_test.go`: four tests covering exact candidate match, HTML `<link rel="icon">` parsing, not-found, and priority order.
- Watchouts:
  - The endpoint resolves candidates fresh on each request (cheap stat calls). Browser caching (`max-age=3600`) prevents redundant fetches across drawer re-renders. The regex only handles `<link>` elements where `rel` and `href` are on the same tag; self-closing or multiline variants with `rel` on one tag and `href` on another won't parse. The endpoint does not handle `manifest.json` or `<meta name="msapplication-TileColor">`: just the HTML `<link>` path and well-known file names.

## P021
Add blue "was working" unread indicator to conversation drawer
- Status: active
- Base: 9c96638
- Files: ConversationDrawer.vue, ConversationDrawerRow.vue, conversationDrawerShared.ts, styles.css, ui/src/i18n/en.ts (and other locale files), ui/src/i18n/types.ts
- Changes:
  - Light blue "was working" unread indicator in the conversation drawer. When an agent finishes working (green dot disappears), a light blue dot (`#60a5fa`) appears in its place on the same row, persisting until the user clicks the row to open the conversation. Applies to both top-level conversation rows and subagent rows. Purely client-side: a transient `Set<string>` of recently-working conversation IDs tracked in `seenWorkingIds`, populated by a `watch` on `convState.working` (true→false transition) in `ConversationDrawerRow.vue`, and cleared by `selectConversation` in `ConversationDrawer.vue`. No DB changes, no localStorage: resets on page reload. Added `unreadResponses` i18n key across all 9 locale files. The unread dot is suppressed when the conversation (or subagent) is the currently selected one, so open threads never show a stale blue dot.

## P022
Make status bar + message input container seamless
- Status: active
- Base: 9c96638
- Files: ui/src/styles.css
- Changes:
  - Constrained `.status-bar-content` to `width: 100%; max-width: 800px; margin: 0 auto;` matching `.message-input-form`, so status text aligns with input box edges while preserving full-width flex layout for `space-between` to work.
  - Removed `border-top: 1px solid var(--border)` from both `.message-input-container` and `.status-bar` (including the mobile media query override) so the two bars blend into one continuous surface.
  - Zeroed out `padding-bottom` on `.status-bar` to tighten spacing above the input container.
  - Reduced `.status-bar` `min-height` from `2.5rem` to `2rem`.
  - Reduced `.message-input-container` `padding` from `1rem` to `0.5rem 1rem` (top/bottom from 1rem to 0.5rem).

## P023
Put subagent preview/date on the same flex row
- Status: active
- Base: 9c96638
- Files: ConversationDrawerRow.vue
- Changes:
  - Subagent preview and date now share the same flex row (`.conversation-meta`), matching the normal conversation item layout. Previously the subagent item had three separate rows: title, preview, and date. Now it has two: title, then date + preview. Removed `.drawer-subagent-date` override so the date uses the same `0.75rem` font size as normal items.

## P024
Add favicon to settled rows + tooltips; reduce preview limit
- Status: active
- Base: 9c96638
- Files: ConversationDrawer.vue, ConversationDrawerRow.vue, styles.css
- Changes:
  - **Favicon on settled rows**: compact settled preview rows now show the repo favicon (14×14) to the left of the thread title, resolved via `/api/repo-favicon?root=<cwd>`. A `reactive` map tracks IDs that 404 to hide the image.
  - **Favicon fallback for archived conversations**: `ConversationDrawerRow.vue`'s `repoRootForFavicon` now falls back to `cwd` when `git_repo_root` is absent (archived conversations lack git state from the API).
  - **Settled row tooltips**: added `v-tooltip.top` to restore, delete, confirm-delete, and cancel buttons in the settled compact rows. Added `:title` to the settled title div for truncated thread names.
  - **Settled preview limit**: reduced from 25 to 20.

## P025
Fix gist export: DB writes, privacy leak, error toasts, stale status
- Status: active
- Base: 9c96638
- Files: server/gist_handlers.go, ui/src/services/api_gist.ts, ui/src/vue/components/ChatInterface.vue
- Changes:
  - **SetGistID used read-only DB transaction**: `db.Queries()` opens an `Rx` (read-only) connection (`query_only=1`). `SetGistID` is a write (UPDATE), so it always failed with `SQLITE_READONLY`. Switched to `db.QueriesTx()` (writer connection). This was the root cause of the 500 on export and the PATCH 404 (gist created on GitHub but `gist_id` never persisted).
  - **Silent SetGistID failure**: the POST handler swallowed the DB error and returned success to the frontend, leaving stale state where `gistId` was set but the DB had NULL. Now returns 500 on DB write failure.
  - **Privacy leak in gist export**: `ListMessages` included `excluded_from_context=true` messages (internal reasoning, tool results) leaking into public gists. Switched all three call sites to `ListMessagesForContext`.
  - **Dead error toasts**: `exportGistAction` catch block checked `msg.includes("gh_not_auth")` but `errorMessage()` returned `data.message` (human text), not `data.error` (code). Both friendly-toast branches never fired. Introduced `GistError` class carrying the error code; catch now matches `err.code` and uses i18n keys (`gistGhNotAuth`, `gistNeedsName`).
  - **Stale gist status on conversation switch**: the gist status watcher had no request cancellation: switching conversations quickly let a stale `getGistStatus` response overwrite the current conversation's state. Added a generation counter to discard stale responses.
- Watchouts:
  - `db.Queries()` vs `db.QueriesTx()` distinction is critical: always use `QueriesTx` for write operations. The `gh gist edit` push has CDN propagation latency; the gist URL may show stale HTML briefly after updating.

## P026
Add slug generation retry for tagged/conversation models
- Status: active
- Base: 4a98848
- Files: slug/slug.go, slug/slug_test.go, server/handlers.go
- Changes:
  - Added retry for slug-tagged and conversation-model slug generation.
  - New `callSlugLLMWithRetry` retries once after a 2s delay, respecting context cancellation. Applied to models tagged `"slug"` (primary slug providers) and the conversation model (last-resort fallback). Not applied to `predictable`, `slug-backup`, or preferred substring models: for those, falling through to the next model in the chain is a better recovery path than retrying.
  - Goroutine timeout bumped 15s → 30s in both `handlers.go` call sites to accommodate the additional retry delay.
  - Root cause: provider-level retries (ant/oai/gem) have 15s+ backoffs that blow past the slug goroutine's 10s per-call + 15s outer timeout, making provider retries dead code for slug calls. Slug-level retry with a short delay works within the budget.
- Watchouts:
  - `slugRetries` and `slugRetryDelay` are package-level constants in `slug/slug.go`. The `flakyLLMService` test mock validates recovery on second attempt; `TestGenerateSlugText_TaggedModelWins` now takes ~2s due to the retry delay on the failing tagged model.

## P027
Block chained `cd <path> && ...` when path equals cwd
- Status: active
- Base: 4a98848
- Files: claudetool/bash.go, claudetool/bashkit/bashkit.go, claudetool/bashkit/bashkit_test.go
- Changes:
  - Block chained `cd <path> && ...` when `<path>` resolves to the current working directory: a pointless no-op.
  - New `ChainsCdToSameDir(bashScript, cwd)` resolves the cd target against cwd (handles absolute, relative, `~`, `.`/`..`) and returns `(true, target)` when they match.
  - Refactored `ChainsCdWithCommand` to share AST-walking helpers (`cdTarget`, `chainsCd`, `chainsCdStmt`) with the new function.
  - `bash.go` calls `ChainsCdToSameDir` after `bashkit.Check()` and returns a permission error when the cd target equals cwd.
  - Error: `permission denied: cd target "<target>" is the same as the current working directory (<cwd>). Run the command directly without chaining cd`
  - Tests: 13 cases covering abs/relative/~/dot-dot/subshell/fallback patterns.
- Watchouts:
  - Only blocks same-dir chains; different-dir chains still get the existing hint from `ChainsCdWithCommand`.

## P028
Include thinking, user, and tool-use content in conversation previews
- Status: active
- Base: 4a98848
- Files: db/query/conversations.sql, db/generated/conversations.sql.go, server/conversation_preview_test.go
- Changes:
  - Conversation drawer previews now include thinking content and user messages. The five preview queries previously only extracted Type=2 (text) blocks from agent messages. They now scan both agent and user messages for Type 2 (text), 3 (thinking), and 5 (tool_use) blocks, extracting the appropriate field via COALESCE. Previews show: agent text > thinking > tool name > user text, depending on which is newest. Added `previewThinkingBlock` helper and `writeUserMsg` helper; added test cases E (thinking-only), F (thinking then text), G (user message).
- Watchouts:
  - ToolName and Thinking fields serialize as empty strings in non-relevant blocks, so each COALESCE leg wraps its field in NULLIF(x, '').

## P029
Add "Auto Expand" brevity mode with turn-level collapse bands
- Status: active
- Base: 4a98848
- Files: ui/src/services/settings.ts, ui/src/utils/conversationView.ts, ui/src/utils/conversationView.test.ts, ui/src/vue/components/renderNode.ts, ui/src/i18n/types.ts, ui/src/i18n/en.ts (and other locale files), ui/src/vue/components/ChatOverflowMenu.vue, ui/src/vue/components/ChatInterface.vue, ui/src/vue/components/MessageRenderNode.vue, ui/src/vue/components/TurnBand.vue (new), ui/src/vue/components/tools/ThinkingContent.vue, ui/src/vue/composables/toolDetail.ts, ui/src/styles.css
- Changes:
  - Added "Auto Expand" as a third brevity mode alongside "See All" and "See End of Turn".
  - **Settings**: `ConversationViewMode` type extended with `"auto-expand"`; `getConversationViewMode()` persists and reads the new value.
  - **Visibility**: `isVisibleConversationMessage()` returns `true` for `"auto-expand"` mode (shows everything).
  - **Render model**: `coalescedItems` computed returns all items when mode is not `"end-of-turn"` (both `"all"` and `"auto-expand"` pass through).
  - **Turn-band wrapping**: `wrapTurnsInBands()` post-processes `sectionNodes` in `buildRenderModel()` for auto-expand mode. It scans for human user messages (turn starts) and end-of-turn agent messages (turn ends), collects inner content between them, and wraps previous turns' inner content in `{ kind: "turn-band", key, duration, children }` nodes. The current turn (latest generation) stays fully expanded.
  - **Duration formatting**: `formatDuration()` computes time between start/end messages, outputting "Xs", "Xm YYs", or "Xh Ym".
  - **Toggle state**: `reactive(new Set<string>())` called `manuallyExpandedTurns` tracks user-toggled turns. `toggleTurn()` and `isTurnExpanded()` functions manage the state.
  - **TurnBand.vue**: New component with transparent background, centered "Run for Xm YYs" text, dashed bottom border, no chevron, expand/collapse with slot for children.
  - **MessageRenderNode.vue**: Added `turn-band` kind handling, imports `TurnBand.vue`, accepts `onToggleTurn` and `isTurnExpanded` props.
  - **ChatInterface.vue**: Passes `toggleTurn` and `isTurnExpanded` to `MessageRenderNode`.
  - **ThinkingContent.vue**: Defaults `isExpanded` to `true` when `conversationViewMode` is `"auto-expand"`.
  - **toolDetail.ts**: `useToolExpanded()` defaults to `true` when `conversationViewMode` is `"auto-expand"`.
  - **ChatOverflowMenu.vue**: Cycle changed from 2-state to 3-state: all → end-of-turn → auto-expand → all. Updated SVG icons, aria labels, and label computation.
  - **i18n**: Added `seeAutoExpand` key across all 9 locale files.
  - **CSS**: Added `.turn-band`, `.turn-toggle`, `.turn-toggle-label`, `.turn-band-body` styles.
  - **Tests**: Added auto-expand assertions to `conversationView.test.ts`.
- Watchouts:
  - The `wrapTurnsInBands` function processes nodes in reverse order to avoid index shifting. `carried-band` nodes within a turn collapse with the turn. Tool pills between messages are also inner content that collapses. The current turn is always expanded regardless of manual toggle state.

## P030
Split chained `&&` bash commands across lines in tool summary
- Status: active
- Base: 4a98848
- Files: BashTool.vue, styles.css
- Changes:
  - Bash tool summary now splits chained `&&` commands across multiple lines. Each segment after the first gets its own line prefixed with `&&`. Per-line CSS truncation (`text-overflow: ellipsis`) keeps long segments within available width. Single-command inputs render identically to before (one line). No hardcoded max length: CSS handles truncation per line.

## P031
Show repo favicon in chat header
- Status: active
- Base: 4a98848
- Files: ChatInterface.vue, styles.css
- Changes:
  - Add the repo favicon to `.header-left` in the chat header, positioned before `.header-title`. Renders a `1.3rem × 1.3rem` `<img>` via `/api/repo-favicon?root=<cwd>`, sized to match the header button icons. The favicon URL falls back through `currentConversation.cwd → selectedCwd`, so it shows whenever any repo context exists: even with no active conversation. Hidden only when no cwd is available at all. Same `/api/repo-favicon` endpoint already used by the drawer rows (PATCH-020). No new server code.
- Watchouts:
  - The base `Conversation` type lacks `git_repo_root` (only present on `ConversationWithState`), so the computed uses `cwd` directly: the server resolves the repo root internally.

## P032
Add tag picker dropdown to chat header
- Status: active
- Base: 4a98848
- Files: ChatInterface.vue, HeaderTagPicker.vue (new), TagPicker.vue (new), styles.css
- Changes:
  - Added a tag picker dropdown to the chat header, positioned in `.header-left` after `.header-title`.
  - **TagPicker.vue**: reusable picker (trigger slot + popover) with suggested tags (filterable), type-to-create input, and toggle-to-apply/remove. Suggested tags: `explore`, `plan`, `verify-plan`, `working`, `verify-work`, `audit`, `revise`, `done`. Input at top auto-focused on open, filters by substring. Already-applied tags show a `✓` checkmark; click toggles apply/remove via `api.updateConversationTags`. Typed text not matching any suggested tag shows a `+ Create "#text"` row; Enter applies it. No delete buttons: removal lives in the drawer. Escape closes the dropdown. Emits `update` with the refreshed conversation.
  - **HeaderTagPicker.vue**: `pi pi-tag` icon button using `btn-icon` (same sizing as Diffs/GitGraph/Terminal buttons), `text severity="secondary"`. Wraps `TagPicker` via its `#trigger` slot and forwards the `update` event to `onUpdated`.
  - **ChatInterface.vue**: `HeaderTagPicker` inserted after `<h1 class="header-title">` inside `.header-left`. Receives `currentConversation` and `onConversationUpdate` props so tag changes refresh the sidebar.
  - **styles.css**: `.tag-picker-wrapper`, `.tag-picker-menu` (min-width 10rem), `.tag-picker-input`, `.tag-picker-item`, `.tag-picker-check`, `.tag-picker-hash`, `.tag-picker-create`, `.tag-picker-empty`: follows the same design tokens as `.group-by-menu`. Removed `overflow: hidden` from `.header-left` so the dropdown floats outside the header area.
- Watchouts:
  - The tag vocabulary is a client-side const (`SUGGESTED_TAGS`), not derived from the server: adding a tag to one conversation won't auto-populate the dropdown for others. The `onConversationUpdate` prop must be wired for tag changes to reflect in the sidebar; without it the button still works but the sidebar won't refresh until a page reload.

## P033
Add per-block action bars to thinking and text blocks
- Status: superseded (upstream 8d34550 "Give thinking blocks their own copy action inside the message" implements the same per-block copy with `splitContentEntities` in coalesceContent.ts; the upstream approach also groups adjacent answer content and keeps message-level info/fork)
- Base: 4a98848
- Files: Message.vue, styles.css
- Changes:
  - Per-block action bars on thinking and text content blocks. The copy/fork/details overlay now appears per-block instead of once for the whole message. Only thinking blocks and text blocks get their own action bar: tool blocks get nothing. Each block's copy extracts only that block's text. Hover tracks the specific block index; a tap on the message body still toggles visibility globally via `showActionBar`.

## P034
Merge drawer tag row into timestamp+preview row
- Status: active
- Base: 4a98848
- Files: ConversationDrawerRow.vue, styles.css
- Changes:
  - Merged the tag row (formerly Row 3) into the timestamp+preview row (Row 4), with tags pushed to the right edge via `margin-left: auto`.
  - The tags `<div>` now lives inside `.conversation-meta` alongside the date and preview spans, using a new `.conversation-tags-inline` class that drops `margin-top` and adds auto-left-margin + `flex-shrink: 0`.
  - Row count for non-draft conversations drops from 4 to 3: [favicon · cwd · branch] / [title · actions · indicators] / [date · preview · tags].
  - Fixed `.message-action-button:last-child` tooltip clipping the right edge.

## P035
Make message blocks full-width; shrink-wrap user bubbles with background
- Status: active
- Base: 4a98848
- Files: styles.css
- Changes:
  - `.message-content` is now `display: flex` (column) with `width: 100%` so all blocks stretch to the full container width.
  - `.message-user` gets `width: auto` and `background: var(--bg-tertiary)` so it shrink-wraps to content and has a visible bubble.
- Watchouts:
  - The per-block action-bar changes documented in the original P035 entry are superseded by upstream 8d34550 (see P033); only the width/background CSS remains.

## P036
Rewrite previous-conversations skill to use `shelley client`
- Status: active
- Base: 4a98848
- Files: skills/builtin/previous-conversations/SKILL.md
- Changes:
  - Rewrote the skill to query conversations through `shelley client` (list, read, search) instead of guessing the SQLite path. Kept the `sqlite3` queries as a raw-SQL fallback for when the server is down or direct DB access is needed.
