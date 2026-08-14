---
name: previous-conversations
description: Use when the user references a previous conversation, asks you to continue earlier work, or you need to look up what was discussed before.
---

Shelley stores conversation history in a SQLite database, but the database
location is not fixed. It defaults to `shelley.db` relative to the server's
working directory, or whatever the server was started with via the global
`-db` flag. Prefer the CLI client, which talks to the running server and never
guesses the path. If a server is not running (or you need raw SQL), fall back
to the `sqlite3` examples below.

## Locate the database

```bash
# The socket path is the reliable way to reach the running server.
# If it isn't at the default location, pass -url explicitly.
SOCKET="$HOME/.config/shelley/shelley.sock"
```

## List recent conversations

```bash
shelley client -url "unix://$SOCKET" list -limit 20
```

## Get messages from a conversation

Replace CONVERSATION_ID with the actual ID:

```bash
shelley client -url "unix://$SOCKET" read CONVERSATION_ID
```

To stream until the agent turn finishes, use `read -wait CONVERSATION_ID`.

## Search conversations

```bash
shelley client -url "unix://$SOCKET" search SEARCH_TERM
```

Search covers both slugs and message content. To list archived
conversations, use `shelley client list -archived`.

## Raw SQL fallback

Use this only when no server is running or you need database access the client
doesn't expose. Point `DB` at the actual database file; it is NOT reliably at
`~/.config/shelley/shelley.db`.

```bash
DB="${SHELLEY_DB:-$HOME/.config/shelley/shelley.db}"

# List recent conversations
sqlite3 "$DB" "SELECT conversation_id, slug, datetime(created_at, 'localtime') as created, datetime(updated_at, 'localtime') as updated FROM conversations ORDER BY updated_at DESC LIMIT 20;"

# Get messages from a conversation
sqlite3 "$DB" "SELECT CASE type WHEN 'user' THEN 'User' ELSE 'Agent' END, substr(json_extract(llm_data, '$.Content[0].Text'), 1, 500) FROM messages WHERE conversation_id='CONVERSATION_ID' AND type IN ('user', 'agent') AND json_extract(llm_data, '$.Content[0].Type') = 2 AND json_extract(llm_data, '$.Content[0].Text') != '' ORDER BY sequence_id;"

# Search conversations by slug
sqlite3 "$DB" "SELECT conversation_id, slug FROM conversations WHERE slug LIKE '%SEARCH_TERM%';"
```
