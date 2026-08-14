---
name: previous-conversations
description: Use when the user references a previous conversation, asks you to continue earlier work, or you need to look up what was discussed before.
---

Prefer the CLI client — it talks to the running server, so you never need to
know where the SQLite DB lives. Fall back to `sqlite3` only if the server is
down or you need raw SQL.

## List recent conversations

```bash
shelley client list -limit 20
```

## Get messages from a conversation

```bash
shelley client read CONVERSATION_ID
```

Use `read -wait CONVERSATION_ID` to stream until the agent turn ends.

## Search conversations

```bash
shelley client search SEARCH_TERM
```

Searches slugs and message content. Use `list -archived` for archived
conversations.

## Raw SQL fallback

```bash
DB="${SHELLEY_DB:-$HOME/.config/shelley/shelley.db}"

# List recent conversations
sqlite3 "$DB" "SELECT conversation_id, slug, datetime(created_at, 'localtime') as created, datetime(updated_at, 'localtime') as updated FROM conversations ORDER BY updated_at DESC LIMIT 20;"

# Get messages from a conversation
sqlite3 "$DB" "SELECT CASE type WHEN 'user' THEN 'User' ELSE 'Agent' END, substr(json_extract(llm_data, '$.Content[0].Text'), 1, 500) FROM messages WHERE conversation_id='CONVERSATION_ID' AND type IN ('user', 'agent') AND json_extract(llm_data, '$.Content[0].Type') = 2 AND json_extract(llm_data, '$.Content[0].Text') != '' ORDER BY sequence_id;"

# Search conversations by slug
sqlite3 "$DB" "SELECT conversation_id, slug FROM conversations WHERE slug LIKE '%SEARCH_TERM%';"
```
