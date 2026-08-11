# PATCH.md — Shelley customizations

`Base` = short SHA of the last upstream commit before the change (upstream SHAs are stable across rebases; ours are not).
`Status`: active | superseded | reverted. `Intent`/`Watchouts` only when not self-explanatory.

## Quick start

```sh
# 1. Check state
cd ~/.config/shelley/shelley-customization
git status --short && git branch --show-current   # expect: clean, on "ki"

# 2. Check upstream
git fetch origin main --tags
git merge-base HEAD origin/main                   # current upstream base
# if origin/main is ahead of this: rebase (below) before making new changes

# 3. Build UI, then Go binary (custom stamp)
cd ui && pnpm run build && cd ..
BASE=$(git merge-base HEAD origin/main)
TAG=$(git describe --tags --abbrev=0 --match 'v[0-9]*' "$BASE"); SHA=$(git rev-parse --short HEAD)
go build -ldflags "-X shelley.exe.dev/version.Version=${TAG#v}-ki.$SHA -X shelley.exe.dev/version.Tag=$TAG -X shelley.exe.dev/version.Customized=true" -o bin/shelley ./cmd/shelley
bin/shelley version                               # verify: -ki.<sha>, customized:true

# 4. Install (side-by-side + rename; never copy over the running binary)
DEST=/var/home/ki/.local/bin/shelley; SRC=bin/shelley
cp "$SRC" "$DEST.new" && chown --reference="$DEST" "$DEST.new" && chmod --reference="$DEST" "$DEST.new" && mv "$DEST.new" "$DEST"

# 5. Restart (delayed so it doesn't kill the current turn; no tmux installed)
setsid bash -c 'sleep 5; systemctl --user restart shelley' >/dev/null 2>&1 < /dev/null &
```

## Rules

Keep the changes minimum. Never plain `make build` — only the `build-custom` ldflags stamp; never copy over the running binary — side-by-side + rename only; never restart shelley mid-turn — always delayed via `setsid`; never anchor to our own commit SHAs (rewritten on rebase) — use upstream `Base` SHAs; always update PATCH.md in the same commit as the code change; don't edit existing DB migrations/schema — new tables only via new migrations.

## PATCH-001
Status: active
Base: 5c2cce3
Files: ChatInterface.vue, ChatOverflowMenu.vue, styles.css
Changes: Moved Diffs, Git Graph, Terminal from the overflow menu into always-visible header buttons, ordered [+, Diffs, Git Graph, Terminal, ⋮]; removed the menu items, their emits, and the now-unused hasCwd prop. Also removed the separators around Archive Conversation and put the theme and notification switches on one row in the overflow menu (split 60:40 — theme has 3 options, notifications 2).
Watchouts: Keep the header buttons as the only entry points for Diffs/Git Graph/Terminal; don't restore the open-diffs/open-git-graph/open-terminal emits or menu items while the header buttons exist.

## PATCH-002
Status: active
Base: 1d4cbe7
Files: ChatInterface.vue, ChatOverflowMenu.vue, ConversationDrawer.vue
Changes: Added tooltips to the New Thread (+), overflow menu (⋮), and mobile hamburger (☰) header buttons, plus the mobile drawer's + and x buttons.

## PATCH-003
Status: active
Base: 1d4cbe7
Files: ChatInterface.vue
Changes: In-app terminal now launches zsh instead of bash (`exec zsh -i`). The server wraps the command as `bash --login -c '<cmd>'`, but `exec zsh -i` replaces bash with zsh.

## PATCH-004
Status: active
Base: 1d4cbe7
Files: BashTool.vue, styles.css
Changes: In the bash tool call details, hid the "Working Directory" and "Command" sections; added "Copy Command" and "Copy Results" buttons to the right of the Output header.

## PATCH-005
Status: active
Base: 1d4cbe7
Files: skills/skills.go, skills/skills_test.go
Changes: Added ~/.agents/skills/ to DefaultDirs discovery paths so Shelley recognizes skills installed by other agent tools (Claude Code, etc.). The directory is checked alongside ~/.config/shelley/, ~/.config/agents/skills/, and ~/.shelley/.

## PATCH-006
Status: active
Base: 1d4cbe7
Files: claudetool/bash.go, claudetool/shell.go, claudetool/toolset.go, claudetool/env.go, server/convo.go, server/handlers.go, cmd/shelley/main.go, ui/src/vue/components/ChatOverflowMenu.vue, ui/src/i18n/en.ts (and other locale files), ui/src/i18n/types.ts, claudetool/bash_test.go
Changes: Git commit attribution is now a user-configurable dropdown ("Git Attribution") in the overflow menu with three options:
  - "Co-authored-by: Shelley" (same as before)
  - "Assisted-by: <model> in Shelley" (derives model display name from active model via ModelInfo.DisplayName; no email)
  - "No agent attribution" (no trailer)
The setting is stored via the existing settings API (key: `shelley.attribution`, values: `co-author`, `assisted-by`, `off`). The legacy `git config shelley.no-trailer` path still works as a fallback when no explicit mode is set.
Watchouts: NewToolSet reads the setting fresh on each turn, so changes take effect on the next message. The `GitAttributionMode` type is shared between bash and shell tools. The `BashTool.resolvedAttribution()` method handles the legacy git config fallback; when an explicit mode is set via the DB setting, it takes precedence. `ShelleyEnv.ModelDisplayName` is populated from `ToolSetConfig.ResolveModelDisplayName()` which calls `LLMProvider.GetModelInfo()`; nil-safe and falls back to the raw model ID.

## PATCH-007
Status: active
Base: 1d4cbe7
Files: db/schema/038-model-enabled.sql, db/schema/039-model-context-window.sql, db/query/models.sql, db/generated/models.sql.go, db/generated/models.go, db/db.go, models/models.go, models/models_test.go, llm/oai/oai.go, llm/oai/oai_responses.go, llm/gem/gem.go, llm/ant/ant.go, server/custom_models.go, server/server.go, ui/src/services/api.ts, ui/src/vue/components/ModelsModal.vue, ui/src/vue/components/ModelFormModal.vue, ui/src/vue/components/ImportModelsModal.vue (new), ui/src/vue/components/customModelConstants.ts, ui/src/styles.css, ui/src/i18n/en.ts (and other locale files), ui/src/i18n/types.ts
Changes: Overhauled custom-model storage and management:
  - **Enable/disable toggle**: new `enabled` column on `models` (default 1). The runtime `Manager.customModelRows()` now calls `GetEnabledModels` (WHERE enabled=1) so disabled models are hidden from the model picker and conversation loop, but remain visible in Manage Models for re-enabling. A ToggleSwitch column was added to the Manage Models table; toggling calls `PUT /api/custom-models/{id}` with `{enabled}`.
  - **Import from /v1/models**: new `POST /api/custom-models/import` endpoint. Accepts `{provider_type: "openai", endpoint, api_key}`, fetches `{endpoint}/models` with `Authorization: Bearer {api_key}`, parses the OpenAI-compatible `{data: [{id, name}]}` response, and bulk-inserts net-new models (skipping any whose `endpoint|model_name` already exists). Uses the `name` field for display_name when present, else the `id`. Imported models default to enabled, image_support=auto, reasoning_support=auto, context_window=0 (auto-detect). A new "Import Models" button in the Manage Models header opens a stacked `ImportModelsModal.vue` (endpoint + API key form, reports imported/skipped counts). The import dialog expects the base endpoint URL (e.g. `https://api.openai.com`) and the server appends `/v1/models` itself.
  - **Partial update fix**: `PUT /api/custom-models/{id}` treats every unset field as "keep existing" so partial bodies (like the enable/disable toggle's `{enabled}`) don't wipe provider_type/endpoint/model_name/etc. or fail the provider_type CHECK constraint.
  - **Context window fix**: new `context_window` column on `models` (default 0). Each llm.Service struct (`oai.Service`, `oai.ResponsesService`, `gem.Service`, `ant.Service`) gained a `ContextWindow int` field; `TokenContextWindow()` returns it when non-zero, falling back to the existing hardcoded per-model switch. `createServiceFromModel()` sets `ContextWindow` from the DB column (falling back to `max_tokens` for pre-migration rows). The add/edit form (`ModelFormModal.vue`) now exposes a "Context Window" input alongside "Max Context Tokens" so users can set the real context limit that drives the UI meter, instead of the hardcoded 128k default that unknown models fell into.
  - **Usage cost and cache hit rate**: The status bar context-usage label shows the estimated cost (from the pricing lookup) alongside the token count (e.g. `76k $0.43`). The token marker in the chat timeline also shows per-call cost. The Usage Details modal (per-message) shows Cache Hit Rate = (cache_read + cache_write) / (cache_read + cache_write + input + output) * 100%. The pricing lookup runs when usage entries are first needed (hover/focus/click on the context bar).
Watchouts: `max_tokens` still controls max output/completion tokens (`max_completion_tokens` on the wire) — it is NOT the context window. `context_window` is the total context size for the UI meter. Only OpenAI Chat Completions (`provider_type: "openai"`) is supported for import; Anthropic/Gemini/OpenAI Responses have different model-list APIs. The sqlc-generated files picked up a version bump (v1.30.0 -> v1.31.1) in their headers — harmless. The `customModelRows()` change means any test that creates a model without `Enabled: 1` will have it filtered from the runtime; `models_test.go` was updated to set `Enabled: 1`.

## PATCH-008
Status: active
Base: 1d4cbe7
Fixes: resolveCost now matches imported-model pricing by endpoint prefix (usage URLs carry trailing segments like /chat/completions that never matched the base endpoint); path-boundary guard prevents hostname spoofing.
Files: db/schema/040-model-pricing.sql, db/query/models.sql, db/generated/models.sql.go, db/generated/models.go, server/custom_models.go, server/model_costs.go
Changes: Import pricing from /v1/models into the DB:
  - New migration adds `input_price`, `output_price`, `cache_read_price`, `cache_write_price` columns to `models` (all REAL, default 0).
  - Import (`POST /api/custom-models/import`) now parses the `pricing` field from the `/v1/models` response and stores per-million-token prices. When only `cache_prompt` is set (no `cache_write`), it is used for both cache read and cache write.
  - `POST /api/model-costs` resolves pricing by checking DB-stored custom model prices first, then falling back to the models.dev snapshot.
  - `PUT /api/custom-models/{id}` supports optional `input_price`, `output_price`, `cache_read_price`, `cache_write_price` fields (pointer types — omit to preserve existing).
  - Duplicate model preserves pricing from source.
Watchouts: Existing models in the DB will have 0 pricing — re-import from the provider to populate. The embedded models.dev snapshot also covers crof.ai models as a fallback.

## PATCH-009
Status: active
Base: 1d4cbe7
Files: ui/src/vue/components/ModelsModal.vue, ui/src/styles.css, ui/src/i18n/en.ts (and other locale files), ui/src/i18n/types.ts
Changes:
  - **Optimistic toggle/delete**: `handleToggleEnabled` and `handleDelete` now update local state immediately without a full `loadModels()` round-trip. Server errors roll back only the affected row. No more blocking wait between toggles.
  - **Checkpoint selection bar**: custom model rows get checkboxes; selecting one or more shows a sticky bar at the bottom with bulk Disable/Enable/Delete actions. All bulk actions are also optimistic with rollback.
  - **Sort enabled first**: custom models sort enabled before disabled in the table.
  - **Checkbox alignment**: header "select all" checkbox visually aligned with per-row checkboxes via PrimeVue `pt` pass-through + scoped CSS.
  - Added `disable`/`enable` i18n keys across all 9 locale files.
Watchouts: `bulkToggleEnabled` only clears `_pending` on the affected models (not all) to avoid interfering with concurrent individual toggles. `handleDelete` re-finds the index on rollback to handle concurrent array shifts. `bulkDelete` and `handleDelete` restore `selectedKeys` on error.

## PATCH-010
Status: active
Base: 1d4cbe7
Files: ui/src/utils/messageTime.ts, ui/src/utils/messageTime.test.ts, ui/src/vue/components/ConversationDrawer.vue, ui/src/vue/components/MessageInfoModal.vue, ui/src/vue/components/UsageDetailModal.vue, ui/src/vue/components/VersionChecker.vue, ui/src/vue/components/TokenCostGraph.vue, ui/src/vue/components/GitGraphViewer.vue, ui/src/vue/components/GitRepoPicker.vue, ui/src/utils/conversationMarkdown.ts
Changes: All time displays now use 24-hour format instead of 12-hour. Added `hour12: false` to the shared `messageTime.ts` formatters (message timestamps, absolute timestamps) and to every other explicit time formatter: ConversationDrawer's today-timestamp, MessageInfoModal, UsageDetailModal, VersionChecker, TokenCostGraph hover time, GitGraphViewer commit date, GitRepoPicker tooltip, and conversationMarkdown's Started/Exported timestamps. The `hour` fields were bumped from `numeric` to `2-digit` in `messageTime.ts` so 24h times render zero-padded (e.g. `09:05` not `9:05`).
Watchouts: `hour12: false` forces 24h regardless of the user's locale; this intentionally overrides locale defaults. The day-only formatters (formatDay) were left unchanged — they don't render a time-of-day.

## PATCH-011
Status: active
Base: 1d4cbe7
Files: ConversationDrawer.vue
Changes: The conversation drawer's CWD display now shows only the repo basename (e.g. `shelley-customization`) instead of the tildified full path (`~/.config/shelley/shelley-customization`). The full path remains in the title tooltip. Also removed the now-unused `tildifyPath` import.

## PATCH-012
Status: active
Base: 1d4cbe7
Files: ui/src/styles.css
Changes: Switched several UI elements from monospace (`var(--font-mono)`) to sans-serif (`var(--font-sans)`): `.app-bar-title`, `.conversation-cwd` (also added `font-weight: bold`), `.model-bar-name`, `.system-prompt-label`, `.system-prompt-tools-label`, `.system-prompt-tool-name`, `.bash-tool-copy-btn`, `.status-message`, `.status-readout`, and `.animated-working`.

## PATCH-013
Status: active
Base: a8e2ee0
Files: BashTool.vue
Changes: When a bash tool call produces empty output, the output block is hidden and the label reads "Clean exit" instead of "Output:". When there is output, it still shows "Output" (or "Output (Error)") as before.


## PATCH-014
Status: active
Base: 1d4cbe7
Files: db/schema/041-gist-id.sql, db/query/conversations.sql, db/generated/conversations.sql.go, db/generated/models.go, server/gist.go, server/gist_export_html.go, server/gist_handlers.go, server/embed/gist.html.tmpl, server/handlers.go, server/server.go, ui/src/services/api_gist.ts, ui/src/vue/components/ChatOverflowMenu.vue, ui/src/vue/components/ChatInterface.vue, ui/src/styles.css, ui/src/i18n/en.ts (and other locale files), ui/src/i18n/types.ts
Changes: Export Shelley session as a self-contained HTML file to a GitHub secret gist via the `gh` CLI.
  - **DB**: new `gist_id TEXT` column on `conversations` (migration 041).
  - **Backend**: `server/gist.go` — `gh gist create/edit` wrapper, slug validation (rejects `Untitled`, `Draft`, `Shelley`, empty), temp file management, error classification (`gh_not_auth`). `server/gist_export_html.go` — `exportGistHTML()` builds self-contained HTML with base64-encoded JSON data blob + embedded JS markdown renderer. `server/gist_handlers.go` — `GET/POST/PATCH /api/conversation/{id}/export-gist`. `server/server.go` — auto-update hook on both single-message and batch `isAgentEndOfTurn` paths.
  - **Frontend**: overflow menu item ("Export to Gist" / "Update Gist"), gist state tracking in ChatInterface, toast feedback, i18n in all 9 locales.
  - **HTML template**: `server/embed/gist.html.tmpl` — self-contained HTML with vanilla JS renderer. Bubbles reserved for user prompts and assistant text only. Thinking blocks, tool calls, and tool results render as standalone collapsible blocks between bubbles (not inside them).
  - **Auto-update**: after every agent end-of-turn (not during streaming/thinking/user cancel), gist is silently updated if one exists. Errors go to console only.
  - **URL format**: `https://gisthost.github.io/?{gist-id}` — renders the HTML directly.
  - **Tests**: `gist_test.go` (slug validation, error classification), `gist_export_html_test.go` (13 edge cases: nil fields, malformed JSON, tool calls with nil/string input, tool result errors, image-only results, etc.), `api_gist.test.ts` (plain-text error body handling).
Watchouts: Requires `gh` CLI installed and authenticated. The `init()` staleness check in `ui/embedfs.go` calls `os.Exit(1)` when build is stale — rebuild UI before testing. `html/template` was replaced with `text/template` for the embedded HTML to avoid escaping content inside `<script>` tags.

## PATCH-015
Status: superseded (upstream 48bfcc3 removed the ⚠️ icon and replaced the warning with color-coded token counts)
Base: 1d4cbe7
Files: ui/src/vue/components/ContextUsageBar.vue
Changes: Made the context warning icon (⚠️) threshold relative to the model's context window instead of a hardcoded 100k tokens. When `maxContextTokens` is known, the warning now triggers at 70% usage (matching the existing label color thresholds). Falls back to the absolute 100k threshold only when the model has no declared context window.
Watchouts: The `percentage` computed prop is already defined above the warning check so it can be reused directly. The auto-open popup feature (`hasAutoOpened` / localStorage gate) still uses the same condition.

## PATCH-016
Status: active
Base: e653372
Files: ui/src/vue/components/ConversationDrawer.vue, ui/src/vue/components/ChatInterface.vue, ui/src/i18n/en.ts, ui/src/styles.css
Changes: Settled (archived) thread preview in sidebar + auto-unarchive on send.
  - **Main view**: up to 25 latest archived threads shown as compact one-line `.conversation-item-settled` rows below the active conversation list, separated by a `.conversation-list-separator` line. Each row shows thread name (left) + timestamp (right); on hover the timestamp fades out and Restore + Delete buttons overlay in-place using `.btn-icon-sm` icons matching active-row button sizing.
  - **Active highlight**: settled rows get the `.active` class (blue background) when the currently viewed conversation matches, with full opacity override.
  - **Click behavior**: clicking a settled thread opens it in the main chat view via `selectConversation`. Sending a new message to a settled thread auto-unarchives it (`api.unarchiveConversation`) and fires `onConversationUnarchived` to update the sidebar.
  - **Archived view**: toggled via "View Archived" footer button. Uses full `ConversationRow` layout (same as active threads) with all original buttons and actions. Header reads "Archived".
  - **i18n**: "Archive" action renamed to "Settle" in English (`archiveConversation`, `archiveConversationAction`, `archiveCurrentConversation`, `archive` keys).
  - **Eager loading**: archived conversations are loaded on mount for instant settled preview display.
Watchouts: Non-English locale files retain their original archive/restore translations (not renamed to "Settle"). The `api_gist.ts` pre-existing type errors are unrelated and predate this patch.

## PATCH-017
Status: active
Base: 1d4cbe7
Files: ui/src/vue/components/ConversationDrawer.vue, ui/src/vue/components/ConversationDrawerRow.vue, ui/src/styles.css
Changes: Moved the collapse (<<) and close (x) buttons from the right side of the drawer header to the left edge, before the title. Renamed the drawer title from "Conversations" to "Shelley" (archived view still shows the "archived" translation). Both `.working-indicator` and `.drawer-working-indicator` now have a solid green background (`--green-500, #22c55e`) with no border. Tightened conversation item padding and gaps. Row structure: Row 1 = [cwd · gitRepoName] (left) + hover-only action buttons (right); Row 2 = [title] [working indicator] [subagent badge]; Row 3 = [date] [preview]. Git commit line removed from template. Action buttons (rename/edit tags/archive) only visible on hover via CSS opacity transition.

## PATCH-018
Status: active
Base: 9c96638
Files: gitstate/gitstate.go, gitstate/gitstate_test.go, server/server.go, server/handlers.go, cmd/go2ts.go, ui/src/generated-types.ts, ui/src/vue/components/ConversationDrawerRow.vue, ui/src/vue/components/ChatInterface.vue, ui/src/vue/components/GitBranchIcon.vue (new), ui/src/styles.css
Changes: Reworked the conversation drawer meta row (Row 1) to show git identity for the repo:
  - **Second meta item = git branch name**: exposed the current branch (`gitstate.GitState.Branch`) as `git_branch` on `ConversationWithState` and render it after the repo slug, with a `repo · branch` tooltip. Falls back to the repo name when the branch is unavailable (e.g. detached HEAD).
  - **First meta item = owner/repo from the remote**: derived an `owner/repo` slug from the repo's remotes as `git_remote` on `ConversationWithState`. The slug resolves the remote this repo actually pushes to — priority: the current branch's upstream remote (e.g. your fork), then `upstream`, then `origin`. Read from the repo `config` file without shelling out to git (with a git-binary fallback). The first meta item shows this slug in place of the cwd folder name; the full cwd remains the tooltip.
  - **Git branch icon**: extracted the chat header's git-branch glyph into a shared `GitBranchIcon.vue` component, used in both the chat header and the drawer meta row (icon import over inline SVG). The drawer icon inherits the row's text color (including the white active state) at the same 0.8 opacity as the branch text.
  - **Branch name at regular font weight** (`font-weight: 400`); the meta row packs tighter (`gap: 0.25rem`).
  - **Drawer actions row collapses to zero width** when the conversation item isn't hovered (`max-width: 0; overflow: hidden`), transitioning open on hover instead of just fading opacity.
Watchouts: The slug prefers the branch's upstream remote so forks show the fork's owner/repo (e.g. `KiKaraage/shelley`) rather than the original. `gitstate.GitState.Equal` includes `RemoteSlug` so cache invalidation accounts for remote changes. The `git_branch`/`git_remote` fields were hand-added to `ui/src/generated-types.ts` because the local `go2ts` generator produces non-reproducible output (tabs + a `gist_id` field); regenerate carefully if you ever run it.

## PATCH-019
Status: active
Base: 1d4cbe7
Files: Makefile
Changes: `build-custom` now installs the freshly built binary to `~/.local/bin/shelley` in addition to `bin/shelley`. It creates `~/.local/bin` if missing, then `install -m 0755` copies the binary there. Prints both the build and install confirmation lines.
Watchouts: `~/.local/bin` must be on `PATH` for the installed binary to be used. This is a convenience copy; the running service still restarts from the side-by-side+rename path in the Quick start, not from this install.
