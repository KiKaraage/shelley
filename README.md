# KiKaraage/shelley: fork of a coding agent from exe.dev

Here's some patches I've made on top of Shelley:

| Patch | Status | Description |
|-------|--------|-------------|
| [PATCH-001](PATCH.md#patch-001) | active | Moved Diffs, Git Graph, Terminal into always-visible header buttons |
| [PATCH-002](PATCH.md#patch-002) | active | Added tooltips to New Thread, overflow menu, hamburger, and drawer buttons |
| [PATCH-003](PATCH.md#patch-003) | active | In-app terminal launches zsh instead of bash |
| [PATCH-004](PATCH.md#patch-004) | active | Hid working directory/command sections in BashTool; added Copy Command/Results buttons |
| [PATCH-005](PATCH.md#patch-005) | active | Added `~/.agents/skills/` to skill discovery paths |
| [PATCH-006](PATCH.md#patch-006) | active | Git commit attribution is now a user-configurable dropdown (co-author, assisted-by, off) |

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