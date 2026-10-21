-- 1. SUBSCRIPTION TIER
CREATE TABLE subscription_tier (
    tier_id TINYINT NOT NULL,
    tier_name VARCHAR(50) NOT NULL,
    monthly_price DECIMAL(10, 2) NOT NULL,
    max_concurrent_streams INT NOT NULL,
    max_video_quality VARCHAR(20) NOT NULL,
    PRIMARY KEY (tier_id),
    CONSTRAINT uk_subscription_tier_name UNIQUE (tier_name),
    CONSTRAINT chk_tier_id CHECK (tier_id IN (0, 1, 2, 3)),
    CONSTRAINT chk_concurrent_streams CHECK (max_concurrent_streams IN (1, 2, 4))
);

-- 2. ACCOUNT
CREATE TABLE account (
    account_id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    email VARCHAR(320) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    billing_status VARCHAR(20) NOT NULL,
    current_tier_id TINYINT NOT NULL,
    pending_tier_id TINYINT NULL,
    pending_tier_effective_date DATETIME NULL,
    registration_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    next_billing_date DATE NOT NULL,
    cancelled_at DATETIME NULL,
    PRIMARY KEY (account_id),
    CONSTRAINT uk_account_email UNIQUE (email),
    CONSTRAINT fk_account_current_tier FOREIGN KEY (current_tier_id) REFERENCES subscription_tier(tier_id),
    CONSTRAINT fk_account_pending_tier FOREIGN KEY (pending_tier_id) REFERENCES subscription_tier(tier_id)
);

-- 3. PROFILE
CREATE TABLE profile (
    profile_id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    account_id INT UNSIGNED NOT NULL,
    profile_name VARCHAR(255) NOT NULL,
    is_kids_profile BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (profile_id),
    CONSTRAINT uk_account_profile_name UNIQUE (account_id, profile_name),
    CONSTRAINT uk_account_profile_composite UNIQUE (account_id, profile_id),
    CONSTRAINT fk_profile_account FOREIGN KEY (account_id) REFERENCES account(account_id) ON DELETE CASCADE
);

-- 4. CATALOG TITLE
CREATE TABLE catalog_title (
    title_id CHAR(36) NOT NULL DEFAULT (UUID()),
    minimum_tier_id TINYINT NOT NULL DEFAULT 0,
    title_name VARCHAR(250) NOT NULL,
    maturity_rating ENUM('G', 'PG', 'PG-13', 'R', 'NC-17') NOT NULL DEFAULT 'G',
    content_type ENUM('Movie', 'Series') NOT NULL,
    synopsis TEXT NULL,
    PRIMARY KEY (title_id),
    CONSTRAINT fk_catalog_tier FOREIGN KEY (minimum_tier_id) REFERENCES subscription_tier(tier_id)
);

-- 5. STREAMABLE MEDIA UNIT (Consolidated)
CREATE TABLE streamable_media_unit (
    media_unit_id CHAR(36) NOT NULL DEFAULT (UUID()),
    duration_seconds INT UNSIGNED NOT NULL,
    video_resolution VARCHAR(20) NOT NULL,
    is_demo_or_trailer BOOLEAN NOT NULL DEFAULT FALSE,
    storage_asset_url VARCHAR(1000) NOT NULL,
    PRIMARY KEY (media_unit_id)
);

-- 6. MOVIE TITLE (Subclass)
CREATE TABLE movie_title (
    title_id CHAR(36) NOT NULL,
    media_unit_id CHAR(36) NOT NULL,
    PRIMARY KEY (title_id),
    CONSTRAINT uk_movie_media_unit UNIQUE (media_unit_id),
    CONSTRAINT fk_movie_catalog FOREIGN KEY (title_id) REFERENCES catalog_title(title_id) ON DELETE CASCADE,
    CONSTRAINT fk_movie_media FOREIGN KEY (media_unit_id) REFERENCES streamable_media_unit(media_unit_id)
);

-- 7. TV SERIES (Subclass)
CREATE TABLE tv_series (
    title_id CHAR(36) NOT NULL,
    total_seasons INT UNSIGNED NOT NULL DEFAULT 1,
    PRIMARY KEY (title_id),
    CONSTRAINT fk_tv_series_catalog FOREIGN KEY (title_id) REFERENCES catalog_title(title_id) ON DELETE CASCADE
);

-- 8. SEASON
CREATE TABLE season (
    season_id CHAR(36) NOT NULL DEFAULT (UUID()),
    series_id CHAR(36) NOT NULL,
    season_number INT UNSIGNED NOT NULL,
    release_year YEAR NULL,
    PRIMARY KEY (season_id),
    CONSTRAINT uq_season_number UNIQUE (series_id, season_number),
    CONSTRAINT fk_season_series FOREIGN KEY (series_id) REFERENCES tv_series(title_id) ON DELETE CASCADE
);

-- 9. EPISODE
CREATE TABLE episode (
    episode_id CHAR(36) NOT NULL DEFAULT (UUID()),
    season_id CHAR(36) NOT NULL,
    media_unit_id CHAR(36) NOT NULL,
    episode_number INT UNSIGNED NOT NULL,
    episode_title VARCHAR(255) NOT NULL,
    PRIMARY KEY (episode_id),
    CONSTRAINT uk_episode_media UNIQUE (media_unit_id),
    CONSTRAINT uq_episode_number UNIQUE (season_id, episode_number),
    CONSTRAINT fk_episode_season FOREIGN KEY (season_id) REFERENCES season(season_id) ON DELETE CASCADE,
    CONSTRAINT fk_episode_media FOREIGN KEY (media_unit_id) REFERENCES streamable_media_unit(media_unit_id)
);

-- 10. WATCHLIST
CREATE TABLE watchlist (
    profile_id INT UNSIGNED NOT NULL,
    title_id CHAR(36) NOT NULL,
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (profile_id, title_id),
    CONSTRAINT fk_watchlist_profile FOREIGN KEY (profile_id) REFERENCES profile(profile_id) ON DELETE CASCADE,
    CONSTRAINT fk_watchlist_title FOREIGN KEY (title_id) REFERENCES catalog_title(title_id) ON DELETE CASCADE
);

-- 11. TITLE RATING
CREATE TABLE title_rating (
    profile_id INT UNSIGNED NOT NULL,
    title_id CHAR(36) NOT NULL,
    score TINYINT UNSIGNED NOT NULL,
    rated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (profile_id, title_id),
    CONSTRAINT chk_title_rating_score CHECK (score BETWEEN 1 AND 10),
    CONSTRAINT fk_title_rating_profile FOREIGN KEY (profile_id) REFERENCES profile(profile_id) ON DELETE CASCADE,
    CONSTRAINT fk_title_rating_title FOREIGN KEY (title_id) REFERENCES catalog_title(title_id) ON DELETE CASCADE
);

-- 12. WATCH HISTORY
CREATE TABLE watch_history (
    profile_id INT UNSIGNED NOT NULL,
    media_unit_id CHAR(36) NOT NULL,
    watched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    playback_stop_seconds INT UNSIGNED NOT NULL DEFAULT 0,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (profile_id, media_unit_id, watched_at),
    CONSTRAINT fk_watch_history_profile FOREIGN KEY (profile_id) REFERENCES profile(profile_id) ON DELETE CASCADE,
    CONSTRAINT fk_watch_history_media FOREIGN KEY (media_unit_id) REFERENCES streamable_media_unit(media_unit_id) ON DELETE CASCADE
);

-- 13. ACTIVE SESSION (With Tenant-Scoped Composite FK)
CREATE TABLE active_session (
    session_id CHAR(36) NOT NULL DEFAULT (UUID()),
    account_id INT UNSIGNED NOT NULL,
    prof_id INT UNSIGNED NOT NULL,
    ip_address VARCHAR(46) NOT NULL,
    device_fingerprint CHAR(64) NULL,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_heartbeat_at TIMESTAMP NULL DEFAULT NULL,
    session_status ENUM('ACTIVE', 'EXPIRED', 'REVOKED') NOT NULL DEFAULT 'ACTIVE',
    PRIMARY KEY (session_id),
    CONSTRAINT fk_active_session_account_profile 
        FOREIGN KEY (account_id, prof_id) 
        REFERENCES profile(account_id, profile_id) ON DELETE CASCADE
);
