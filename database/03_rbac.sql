-- ============================================================================
-- A. Create Functional & Tier-Based Roles
-- ============================================================================
CREATE ROLE IF NOT EXISTS
    'role_dba_superuser',
    'role_catalog_admin',
    'role_billing_service',
    'role_subscriber_base',
    'role_tier_free',
    'role_tier_basic',
    'role_tier_standard',
    'role_tier_premium',
    'role_kids_profile';

-- ============================================================================
-- B. Superuser / DBA Role
-- ============================================================================
GRANT ALL PRIVILEGES ON mini_netflix.* TO 'role_dba_superuser' WITH GRANT OPTION;

-- ============================================================================
-- C. Content Admin Role (Catalog & Media Management Only)
-- ============================================================================
GRANT SELECT, INSERT, UPDATE, DELETE ON mini_netflix.catalog_title         TO 'role_catalog_admin';
GRANT SELECT, INSERT, UPDATE, DELETE ON mini_netflix.movie_title           TO 'role_catalog_admin';
GRANT SELECT, INSERT, UPDATE, DELETE ON mini_netflix.tv_series             TO 'role_catalog_admin';
GRANT SELECT, INSERT, UPDATE, DELETE ON mini_netflix.season                TO 'role_catalog_admin';
GRANT SELECT, INSERT, UPDATE, DELETE ON mini_netflix.episode               TO 'role_catalog_admin';
GRANT SELECT, INSERT, UPDATE, DELETE ON mini_netflix.streamable_media_unit TO 'role_catalog_admin';
GRANT SELECT                         ON mini_netflix.subscription_tier     TO 'role_catalog_admin';

-- ============================================================================
-- D. Billing Service Role (PCI-DSS & Audit Ledger Isolation)
-- ============================================================================
GRANT SELECT, INSERT ON mini_netflix.credit_card         TO 'role_billing_service';
GRANT SELECT, INSERT ON mini_netflix.payment_transaction TO 'role_billing_service';
GRANT SELECT (account_id, email, billing_status, current_tier_id, next_billing_date),
      UPDATE (billing_status, current_tier_id, pending_tier_id, next_billing_date, cancelled_at)
      ON mini_netflix.account TO 'role_billing_service';

-- ============================================================================
-- E. Base Subscriber Role (Column-Level Masking of Sensitive Fields)
-- ============================================================================
-- Allow reading public account metadata while hiding password_hash
GRANT SELECT (account_id, email, billing_status, current_tier_id, registration_date, next_billing_date)
      ON mini_netflix.account TO 'role_subscriber_base';

-- Allow reading masked card metadata while hiding payment_token (BR-06)
GRANT SELECT (card_id, account_id, cardholder_name, card_brand, last_four, exp_month, exp_year, is_default),
      INSERT (account_id, cardholder_name, card_brand, last_four, exp_month, exp_year, payment_token),
      DELETE ON mini_netflix.credit_card TO 'role_subscriber_base';

-- Profile, Watchlist, Ratings, History, and Session privileges
GRANT SELECT, INSERT, UPDATE, DELETE ON mini_netflix.profile        TO 'role_subscriber_base';
GRANT SELECT, INSERT, DELETE         ON mini_netflix.watchlist      TO 'role_subscriber_base';
GRANT SELECT, INSERT, UPDATE, DELETE ON mini_netflix.title_rating   TO 'role_subscriber_base';
GRANT SELECT, INSERT, UPDATE         ON mini_netflix.watch_history  TO 'role_subscriber_base';
GRANT SELECT, INSERT, UPDATE, DELETE ON mini_netflix.active_session TO 'role_subscriber_base';
GRANT SELECT                         ON mini_netflix.subscription_tier TO 'role_subscriber_base';

-- ============================================================================
-- F. Subscription Tier Sub-Roles (Role Inheritance + View Gating)
-- ============================================================================
GRANT 'role_subscriber_base' TO 'role_tier_free', 'role_tier_basic', 'role_tier_standard', 'role_tier_premium';

GRANT SELECT ON mini_netflix.v_catalog_tier_free     TO 'role_tier_free';
GRANT SELECT ON mini_netflix.v_catalog_tier_basic    TO 'role_tier_basic';
GRANT SELECT ON mini_netflix.v_catalog_tier_standard TO 'role_tier_standard';
GRANT SELECT ON mini_netflix.v_catalog_tier_premium  TO 'role_tier_premium';
GRANT SELECT ON mini_netflix.v_catalog_kids          TO 'role_kids_profile';

-- 1. Prevent subscribers from bypassing tier views via direct base-table queries
REVOKE SELECT ON mini_netflix.catalog_title FROM 'role_subscriber_base';
REVOKE SELECT ON mini_netflix.catalog_title FROM 'role_tier_free', 'role_tier_basic', 'role_tier_standard';

-- 2. Enforce immutable financial audit trail (no role except DBA may alter/delete past transactions)
REVOKE UPDATE, DELETE ON mini_netflix.payment_transaction FROM 'role_billing_service';
REVOKE UPDATE, DELETE ON mini_netflix.payment_transaction FROM 'role_subscriber_base';

-- 3. Dynamic Downgrade / Suspension Enforcement (e.g., when a user downgrades from Premium to Free)
REVOKE 'role_tier_premium' FROM 'david_free_user'@'%';
GRANT  'role_tier_free'    TO   'david_free_user'@'%';
SET DEFAULT ROLE 'role_tier_free' TO 'david_free_user'@'%';

-- 4. Immediate Streaming Revocation upon Account Suspension (BR-05)
REVOKE INSERT, UPDATE ON mini_netflix.active_session FROM 'emma_suspended_user'@'%';

-- Create database users mapped to organizational and tier roles
CREATE USER IF NOT EXISTS 'dba_admin'@'%'        IDENTIFIED BY 'StrongDbaPass#2026';
CREATE USER IF NOT EXISTS 'content_editor'@'%'   IDENTIFIED BY 'EditorPass#2026';
CREATE USER IF NOT EXISTS 'alice_premium'@'%'    IDENTIFIED BY 'AlicePass#2026';
CREATE USER IF NOT EXISTS 'david_free_user'@'%'  IDENTIFIED BY 'DavidPass#2026';

-- Bind roles to users
GRANT 'role_dba_superuser' TO 'dba_admin'@'%';
GRANT 'role_catalog_admin' TO 'content_editor'@'%';
GRANT 'role_tier_premium'  TO 'alice_premium'@'%';
GRANT 'role_tier_free'     TO 'david_free_user'@'%';

-- Activate default roles immediately upon connection
SET DEFAULT ROLE ALL TO
    'dba_admin'@'%',
    'content_editor'@'%',
    'alice_premium'@'%',
    'david_free_user'@'%';

FLUSH PRIVILEGES;