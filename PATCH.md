# PATCH.md — Shelley customizations

`Base` = short SHA of the last upstream commit before the change (upstream SHAs are stable across rebases; ours are not).
`Status`: active | superseded | reverted. `Intent`/`Watchouts` only when not self-explanatory.

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
Changes: In-app terminal now launches zsh by default instead of bash (`exec "${SHELL:-zsh}" -i`).
