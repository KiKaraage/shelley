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

- Never plain `make build` — only the `build-custom` ldflags stamp.
- Never copy over the running binary — side-by-side + rename only.
- Never restart shelley mid-turn — always delayed via `setsid`.
- Never anchor to our own commit SHAs (rewritten on rebase) — use upstream `Base` SHAs.
- Always update PATCH.md in the same commit as the code change.
- Don't edit existing DB migrations/schema — new tables only via new migrations.

## PATCH-001
Status: active
Base: 5c2cce3
Files: ChatInterface.vue, ChatOverflowMenu.vue, styles.css
Changes: Moved Diffs, Git Graph, Terminal from the overflow menu into always-visible header buttons, ordered [+, Diffs, Git Graph, Terminal, ⋮]; removed the menu items, their emits, and the now-unused hasCwd prop. Also removed the separators around Archive Conversation and put the theme and notification switches on one row in the overflow menu.
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
