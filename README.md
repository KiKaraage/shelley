# KiKaraage/shelley: fork of a coding agent from exe.dev

Here's some patches I've made on top of Shelley:

| Patch | Scope | Status | Description |
|-------|-------|--------|-------------|
| [P001](PATCH.md#p001) (S · Aug 7) | ui | superseded | Move Diffs, Git Graph, Terminal into always-visible header buttons (upstream moved them into the header itself) |
| [P002](PATCH.md#p002) (S · Aug 7) | ui | active | Add tooltips to New Thread, overflow menu, hamburger, and drawer buttons |
| [P003](PATCH.md#p003) (S · Aug 7) | ui | active | Launch zsh instead of bash in the in-app terminal |
| [P004](PATCH.md#p004) (M · Aug 7) | ui | active | Hide working directory/command sections in BashTool; add Copy Command/Results buttons |
| [P005](PATCH.md#p005) (S · Aug 7) | go | active | Add `~/.agents/skills/` to skill discovery paths |
| [P006](PATCH.md#p006) (L · Aug 7) | full | active | Make git commit attribution a user-configurable dropdown (co-author, assisted-by, off) |
| [P007](PATCH.md#p007) (L · Aug 9) | full | active | Add custom model enable/disable, import from /v1/models, and context window fix |

| [P008](PATCH.md#p008) (M · Aug 9) | go | active | Import pricing from /v1/models into DB |
| [P009](PATCH.md#p009) (M · Aug 9) | ui | active | Add optimistic toggle/delete + checkpoint selection bar to ModelsModal |
| [P010](PATCH.md#p010) (M · Aug 9) | ui | active | Use 24h for all time displays |
| [P011](PATCH.md#p011) (S · Aug 9) | ui | active | Show repo basename in conversation drawer CWD |
| [P012](PATCH.md#p012) (S · Aug 10) | ui | active | Switch 10 UI selectors from monospace to sans-serif font |
| [P013](PATCH.md#p013) (S · Aug 10) | ui | active | Hide bash tool output block when output is empty |
| [P014](PATCH.md#p014) (L · Aug 10) | full | active | Export session as self-contained HTML to GitHub Gist |
| [P015](PATCH.md#p015) (S · Aug 10) | ui | superseded | Make context warning threshold relative to model context window |
| [P016](PATCH.md#p016) (M · Aug 10) | ui | active | Add settled thread preview in sidebar + auto-unarchive on send |
| [P017](PATCH.md#p017) (M · Aug 10) | ui | active | Move collapse/close buttons to drawer header left; rename title to "Shelley" |
| [P018](PATCH.md#p018) (M · Aug 11) | full | active | Show git branch + owner/repo slug in drawer meta row |
| [P019](PATCH.md#p019) (S · Aug 11) | build | active | Install `build-custom` output to `~/.local/bin/shelley` |
| [P020](PATCH.md#p020) (M · Aug 11) | full | active | Show repo favicon in drawer meta row |
| [P021](PATCH.md#p021) (M · Aug 11) | ui | active | Add blue "was working" unread indicator to conversation drawer |
| [P022](PATCH.md#p022) (S · Aug 11) | ui | active | Make status bar + message input container seamless |
| [P023](PATCH.md#p023) (S · Aug 11) | ui | active | Put subagent preview/date on the same flex row |
| [P024](PATCH.md#p024) (S · Aug 11) | ui | active | Add favicon to settled rows + tooltips; reduce preview limit |
| [P025](PATCH.md#p025) (M · Aug 12) | full | active | Fix gist export: DB writes, privacy leak, error toasts, stale status |
| [P026](PATCH.md#p026) (S · Aug 12) | go | active | Add slug generation retry for tagged/conversation models |
| [P027](PATCH.md#p027) (M · Aug 12) | go | active | Block chained `cd <path> && ...` when path equals cwd |
| [P028](PATCH.md#p028) (M · Aug 12) | go | active | Include thinking, user, and tool-use content in conversation previews |
| [P029](PATCH.md#p029) (L · Aug 12) | ui | active | Add "Auto Expand" brevity mode with turn-level collapse bands |
| [P030](PATCH.md#p030) (S · Aug 12) | ui | active | Split chained `&&` bash commands across lines in tool summary |
| [P031](PATCH.md#p031) (S · Aug 12) | ui | active | Show repo favicon in chat header |
| [P032](PATCH.md#p032) (M · Aug 12) | ui | active | Add tag picker dropdown to chat header |
| [P033](PATCH.md#p033) (M · Aug 13) | ui | active | Add per-block action bars to thinking and text blocks |
| [P034](PATCH.md#p034) (S · Aug 13) | ui | active | Merge drawer tag row into timestamp+preview row |
| [P035](PATCH.md#p035) (M · Aug 13) | ui | active | Make message blocks full-width; shrink-wrap user bubbles with background |
| [P036](PATCH.md#p036) (S · Aug 14) | skills | active | Rewrite previous-conversations skill to use `shelley client` |
| [P037](PATCH.md#p037) (M · Aug 14) | ui | active | Use shared TagPicker for drawer tag editing |
| [P038](PATCH.md#p038) (S · Aug 14) | ui | active | Show leading `#` comments in bash tool summary |
| [P039](PATCH.md#p039) (L · Aug 15) | full | active | Shelley Tasks homepage: task list, create/edit modal, agent tool, shortcuts |
| [P040](PATCH.md#p040) (L · Aug 15) | full | active | Tasks homepage refinements: fix Create Thread + TagPicker overflow, pill dir chooser, multiline task title, latest-thread cwd autofill, in-input bold+blue tag highlighting via contenteditable, start-truncated target dir, border/caret fixes |

Scope: `ui` (frontend only) · `go` (backend only) · `full` (UI + Go/DB) · `build` · `skills`

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
| [P041](PATCH.md#p041) (S · Aug 15) | ui | active | Right sidebar TaskList overlay from the chat header |
