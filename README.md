# KiKaraage/shelley: fork of a coding agent from exe.dev

Here's some patches I've made on top of Shelley:

| Patch | Status | Description |
|-------|--------|-------------|
| Patch | Status | Description |
|-------|--------|-------------|
| [P001](PATCH.md#p001) (S · Aug 7) | active | Moved Diffs, Git Graph, Terminal into always-visible header buttons |
| [P002](PATCH.md#p002) (S · Aug 7) | active | Added tooltips to New Thread, overflow menu, hamburger, and drawer buttons |
| [P003](PATCH.md#p003) (S · Aug 7) | active | In-app terminal launches zsh instead of bash |
| [P004](PATCH.md#p004) (M · Aug 7) | active | Hid working directory/command sections in BashTool; added Copy Command/Results buttons |
| [P005](PATCH.md#p005) (S · Aug 7) | active | Added `~/.agents/skills/` to skill discovery paths |
| [P006](PATCH.md#p006) (L · Aug 7) | active | Git commit attribution is now a user-configurable dropdown (co-author, assisted-by, off) |
| [P007](PATCH.md#p007) (L · Aug 9) | active | Custom model enable/disable, import from /v1/models, context window fix |
| [P008](PATCH.md#p008) (M · Aug 9) | active | Import pricing from /v1/models into DB |
| [P009](PATCH.md#p009) (M · Aug 9) | active | ModelsModal optimistic toggle/delete + checkpoint selection bar |
| [P010](PATCH.md#p010) (M · Aug 9) | active | Use 24h for all time display |
| [P011](PATCH.md#p011) (S · Aug 9) | active | Show repo basename in conversation drawer CWD |
| [P012](PATCH.md#p012) (S · Aug 10) | active | Switched 10 UI selectors from monospace to sans-serif font |
| [P013](PATCH.md#p013) (S · Aug 10) | active | Hid bash tool output block when output is empty |
| [P014](PATCH.md#p014) (L · Aug 10) | active | Export session as self-contained HTML to GitHub Gist |
| [P015](PATCH.md#p015) (S · Aug 10) | superseded | Context warning threshold relative to model context window |
| [P016](PATCH.md#p016) (M · Aug 10) | active | Settled thread preview in sidebar + auto-unarchive on send |
| [P017](PATCH.md#p017) (M · Aug 10) | active | Moved collapse/close buttons to drawer header left; renamed title to "Shelley" |
| [P018](PATCH.md#p018) (M · Aug 11) | active | Git branch + owner/repo slug in drawer meta row |
| [P019](PATCH.md#p019) (S · Aug 11) | active | `build-custom` installs to `~/.local/bin/shelley` |
| [P020](PATCH.md#p020) (M · Aug 11) | active | Repo favicon in drawer meta row |
| [P021](PATCH.md#p021) (M · Aug 11) | active | Blue "was working" unread indicator in conversation drawer |
| [P022](PATCH.md#p022) (S · Aug 11) | active | Seamless status bar + message input container |
| [P023](PATCH.md#p023) (S · Aug 11) | active | Subagent preview/date share same flex row |
| [P024](PATCH.md#p024) (S · Aug 11) | active | Favicon on settled rows + tooltips + preview limit reduced |
| [P025](PATCH.md#p025) (M · Aug 12) | active | Fixed gist export: DB writes, privacy leak, error toasts, stale status |
| [P026](PATCH.md#p026) (S · Aug 12) | active | Slug generation retry for tagged/conversation models |
| [P027](PATCH.md#p027) (M · Aug 12) | active | Block chained `cd <path> && ...` when path equals cwd |

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
