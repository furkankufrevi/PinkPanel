DROP INDEX IF EXISTS idx_domains_parent_id;
ALTER TABLE domains DROP COLUMN separate_dns;
ALTER TABLE domains DROP COLUMN parent_id;
