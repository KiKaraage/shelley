# KiKaraage/shelley: fork of a coding agent from exe.dev

Here's some patches I've made on top of Shelley:

| Patch | Status | Description |
|-------|--------|-------------|
| Patch | Status | Description |
|-------|--------|-------------|
| [PATCH-001](PATCH.md#patch-001) (S · Aug 7) | active | Moved Diffs, Git Graph, Terminal into always-visible header buttons |
| [PATCH-002](PATCH.md#patch-002) (S · Aug 7) | active | Added tooltips to New Thread, overflow menu, hamburger, and drawer buttons |
| [PATCH-003](PATCH.md#patch-003) (S · Aug 7) | active | In-app terminal launches zsh instead of bash |
| [PATCH-004](PATCH.md#patch-004) (M · Aug 7) | active | Hid working directory/command sections in BashTool; added Copy Command/Results buttons |
| [PATCH-005](PATCH.md#patch-005) (S · Aug 7) | active | Added `~/.agents/skills/` to skill discovery paths |
| [PATCH-006](PATCH.md#patch-006) (L · Aug 7) | active | Git commit attribution is now a user-configurable dropdown (co-author, assisted-by, off) |
| [PATCH-007](PATCH.md#patch-007) (L · Aug 9) | active | Custom model enable/disable, import from /v1/models, context window fix |
| [PATCH-008](PATCH.md#patch-008) (M · Aug 9) | active | Import pricing from /v1/models into DB |
| [PATCH-009](PATCH.md#patch-009) (M · Aug 9) | active | ModelsModal optimistic toggle/delete + checkpoint selection bar |
| [PATCH-010](PATCH.md#patch-010) (M · Aug 9) | active | Use 24h for all time display |
| [PATCH-011](PATCH.md#patch-011) (S · Aug 9) | active | Show repo basename in conversation drawer CWD |
| [PATCH-012](PATCH.md#patch-012) (S · Aug 10) | active | Switched 10 UI selectors from monospace to sans-serif font |
| [PATCH-013](PATCH.md#patch-013) (S · Aug 10) | active | Hid bash tool output block when output is empty |
| [PATCH-014](PATCH.md#patch-014) (L · Aug 10) | active | Export session as self-contained HTML to GitHub Gist |
| [PATCH-015](PATCH.md#patch-015) (S · Aug 10) | superseded | Context warning threshold relative to model context window |
| [PATCH-016](PATCH.md#patch-016) (M · Aug 10) | active | Settled thread preview in sidebar + auto-unarchive on send |
| [PATCH-017](PATCH.md#patch-017) (M · Aug 10) | active | Moved collapse/close buttons to drawer header left; renamed title to "Shelley" |
| [PATCH-018](PATCH.md#patch-018) (M · Aug 11) | active | Git branch + owner/repo slug in drawer meta row |
| [PATCH-019](PATCH.md#patch-019) (S · Aug 11) | active | `build-custom` installs to `~/.local/bin/shelley` |
| [PATCH-020](PATCH.md#patch-020) (M · Aug 11) | active | Repo favicon in drawer meta row |
| [PATCH-021](PATCH.md#patch-021) (M · Aug 11) | active | Blue "was working" unread indicator in conversation drawer |
| [PATCH-022](PATCH.md#patch-022) (S · Aug 11) | active | Seamless status bar + message input container |
| [PATCH-023](PATCH.md#patch-023) (S · Aug 11) | active | Subagent preview/date share same flex row |
| [PATCH-024](PATCH.md#patch-024) (S · Aug 11) | active | Favicon on settled rows + tooltips + preview limit reduced |
| [PATCH-025](PATCH.md#patch-025) (M · Aug 12) | active | Fixed gist export: DB writes, privacy leak, error toasts, stale status |
| [PATCH-026](PATCH.md#patch-026) (S · Aug 12) | active | Slug generation retry for tagged/conversation models |
| [PATCH-027](PATCH.md#patch-027) (M · Aug 12) | active | Block chained `cd <path> && ...` when path equals cwd |

---

Shelley is a mobile-friendly, web-based, multi-conversation, multi-modal,
multi-model, single-user coding agent built for but not exclusive to
[exe.dev](https://exe.dev/). It does not come with authorization or sandboxing:
bring your own.

- *Mobile-friendly* because ideas can come any time.
- *Web-based*, because terminal-based scroll back is punishment for shoplifting in some countries.
- *Multi-modal* because screenshots, charts, and graphs are necessary, not to mention delightful.
- *Multi-model* to benefit from all the innovation going on.
- *Single-user* because it makes sense to bring the agent to the compute.

# Architecture 

Go for backend, SQLite for storage, and Typescript with Vue 3 + PrimeVue for the UI. 

The data model is that Conversations have Messages, which might be from the
user, the model, the tools, or the harness. All of that is stored in the
database, and we use a SSE endpoint to keep the UI updated. 

## Build from Source

- Run `make serve` to start Shelley locally.
- If you want to see how mobile looks, and you're on your home network where you've got mDNS working fine, run `socat TCP-LISTEN:9001,fork TCP:localhost:9000`
