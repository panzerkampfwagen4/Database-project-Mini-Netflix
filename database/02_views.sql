-- Tier-Gated Catalog Views (Enforces BR-01 inside the DBMS)
CREATE OR REPLACE VIEW v_catalog_tier_free AS
SELECT title_id, title_name, content_type, maturity_rating, minimum_tier_id, synopsis
FROM catalog_title
WHERE minimum_tier_id <= 0;

CREATE OR REPLACE VIEW v_catalog_tier_basic AS
SELECT title_id, title_name, content_type, maturity_rating, minimum_tier_id, synopsis
FROM catalog_title
WHERE minimum_tier_id <= 1;

CREATE OR REPLACE VIEW v_catalog_tier_standard AS
SELECT title_id, title_name, content_type, maturity_rating, minimum_tier_id, synopsis
FROM catalog_title
WHERE minimum_tier_id <= 2;

CREATE OR REPLACE VIEW v_catalog_tier_premium AS
SELECT title_id, title_name, content_type, maturity_rating, minimum_tier_id, synopsis
FROM catalog_title
WHERE minimum_tier_id <= 3;

-- Parental Control View (Enforces BR-03 inside the DBMS)
CREATE OR REPLACE VIEW v_catalog_kids AS
SELECT title_id, title_name, content_type, maturity_rating, minimum_tier_id, synopsis
FROM catalog_title
WHERE maturity_rating IN ('G', 'PG');