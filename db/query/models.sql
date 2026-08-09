-- name: GetModels :many
SELECT * FROM models ORDER BY created_at ASC;

-- name: GetEnabledModels :many
SELECT * FROM models WHERE enabled = 1 ORDER BY created_at ASC;

-- name: GetModel :one
SELECT * FROM models WHERE model_id = ?;

-- name: CreateModel :one
INSERT INTO models (model_id, display_name, provider_type, endpoint, api_key, model_name, max_tokens, tags, reasoning_effort, image_support, reasoning_support, reasoning_map, enabled, context_window, input_price, output_price, cache_read_price, cache_write_price)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateModel :one
UPDATE models
SET display_name = ?,
    provider_type = ?,
    endpoint = ?,
    api_key = ?,
    model_name = ?,
    max_tokens = ?,
    tags = ?,
    reasoning_effort = ?,
    image_support = ?,
    reasoning_support = ?,
    reasoning_map = ?,
    enabled = ?,
    context_window = ?,
    input_price = ?,
    output_price = ?,
    cache_read_price = ?,
    cache_write_price = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE model_id = ?
RETURNING *;

-- name: DeleteModel :exec
DELETE FROM models WHERE model_id = ?;
