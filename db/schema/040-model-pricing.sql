-- Pricing columns: USD per million tokens, populated from /v1/models on import.
ALTER TABLE models ADD COLUMN input_price REAL NOT NULL DEFAULT 0;
ALTER TABLE models ADD COLUMN output_price REAL NOT NULL DEFAULT 0;
ALTER TABLE models ADD COLUMN cache_read_price REAL NOT NULL DEFAULT 0;
ALTER TABLE models ADD COLUMN cache_write_price REAL NOT NULL DEFAULT 0;
