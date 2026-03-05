package db

const Schema = `
CREATE TABLE IF NOT EXISTS specs (
	id          TEXT PRIMARY KEY,
	name        TEXT NOT NULL,
	type        TEXT NOT NULL,
	status      TEXT NOT NULL DEFAULT 'draft',
	body        TEXT NOT NULL,
	tags        TEXT,
	created_at  TEXT NOT NULL,
	updated_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS edges (
	from_id     TEXT NOT NULL REFERENCES specs(id),
	to_id       TEXT NOT NULL REFERENCES specs(id),
	relation    TEXT NOT NULL,
	created_at  TEXT NOT NULL,
	PRIMARY KEY (from_id, to_id, relation)
);

CREATE TABLE IF NOT EXISTS code_links (
	spec_id     TEXT NOT NULL REFERENCES specs(id),
	file_path   TEXT NOT NULL,
	symbol      TEXT,
	link_type   TEXT NOT NULL,
	scope       TEXT NOT NULL DEFAULT 'file',
	start_line  INTEGER,
	start_col   INTEGER,
	end_line    INTEGER,
	end_col     INTEGER,
	created_at  TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_code_links_unique
	ON code_links (spec_id, file_path, link_type, COALESCE(start_line, 0));

CREATE TABLE IF NOT EXISTS history (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	spec_id     TEXT NOT NULL REFERENCES specs(id),
	changed_at  TEXT NOT NULL,
	field       TEXT NOT NULL,
	old_value   TEXT,
	new_value   TEXT
);
`
