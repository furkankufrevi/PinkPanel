ALTER TABLE domains ADD COLUMN parent_id INTEGER DEFAULT NULL REFERENCES domains(id) ON DELETE CASCADE;
ALTER TABLE domains ADD COLUMN separate_dns INTEGER NOT NULL DEFAULT 0;
CREATE INDEX idx_domains_parent_id ON domains(parent_id);
