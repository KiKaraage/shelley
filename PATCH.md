# PATCH.md — Shelley customizations

`Base` = short SHA of the last upstream commit before the change (upstream SHAs are stable across rebases; ours are not).
`Status`: active | superseded | reverted. `Intent`/`Watchouts` only when not self-explanatory.

## Quick start

```sh
# 1. Check state
cd ~/.config/shelley/shelley-customization
git status --short && git branch --show-current   # expect: clean, on "custom"

# 2. Check upstream
git fetch origin main --tags
git merge-base HEAD origin/main                   # current upstream base
# if origin/main is ahead of this: rebase (below) before making new changes

# 3. Build UI, then Go binary (custom stamp)
cd ui && pnpm run build && cd ..
BASE=$(git merge-base HEAD origin/main)
TAG=$(git describe --tags --abbrev=0 --match 'v[0-9]*' "$BASE"); SHA=$(git rev-parse --short HEAD)
go build -ldflags "-X shelley.exe.dev/version.Version=${TAG#v}-custom.$SHA -X shelley.exe.dev/version.Tag=$TAG -X shelley.exe.dev/version.Customized=true" -o bin/shelley ./cmd/shelley
bin/shelley version                               # verify: -custom.<sha>, customized:true

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
  - **Import from /v1/models**: new `POST /api/custom-models/import` endpoint. Accepts `{provider_type: "openai", endpoint, api_key}`, fetches `{endpoint}/v1/models` with `Authorization: Bearer {api_key}`, parses the OpenAI-compatible `{data: [{id, name}]}` response, and bulk-inserts net-new models (skipping any whose `endpoint|model_name` already exists). Uses the `name` field for display_name when present, else the `id`. Imported models default to enabled, image_support=auto, reasoning_support=auto, context_window=0 (auto-detect). A new "Import Models" button in the Manage Models header opens a stacked `ImportModelsModal.vue` (endpoint + API key form, reports imported/skipped counts).
  - **Context window fix**: new `context_window` column on `models` (default 0). Each llm.Service struct (`oai.Service`, `oai.ResponsesService`, `gem.Service`, `ant.Service`) gained a `ContextWindow int` field; `TokenContextWindow()` returns it when non-zero, falling back to the existing hardcoded per-model switch. `createServiceFromModel()` sets `ContextWindow` from the DB column (falling back to `max_tokens` for pre-migration rows). The add/edit form (`ModelFormModal.vue`) now exposes a "Context Window" input alongside "Max Context Tokens" so users can set the real context limit that drives the UI meter, instead of the hardcoded 128k default that unknown models fell into.
Watchouts: `max_tokens` still controls max output/completion tokens (`max_completion_tokens` on the wire) — it is NOT the context window. `context_window` is the total context size for the UI meter. Only OpenAI Chat Completions (`provider_type: "openai"`) is supported for import; Anthropic/Gemini/OpenAI Responses have different model-list APIs. The sqlc-generated files picked up a version bump (v1.30.0 -> v1.31.1) in their headers — harmless. The `customModelRows()` change means any test that creates a model without `Enabled: 1` will have it filtered from the runtime; `models_test.go` was updated to set `Enabled: 1`.
