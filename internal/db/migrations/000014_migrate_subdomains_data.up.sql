INSERT INTO domains (name, document_root, parent_id, status, php_version, separate_dns)
SELECT sub.name || '.' || dom.name, sub.document_root, sub.domain_id, 'active', dom.php_version, 0
FROM subdomains sub
JOIN domains dom ON sub.domain_id = dom.id;

DROP TABLE IF EXISTS subdomains;
