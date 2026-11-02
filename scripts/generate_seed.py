import csv
import uuid
import re

def escape_sql(val):
    if val is None:
        return "NULL"
    s = str(val).replace("\\", "\\\\").replace("'", "''")
    return f"'{s}'"

def map_rating(rating_str):
    rating_str = rating_str.strip() if rating_str else ""
    valid = {'G', 'PG', 'PG-13', 'R', 'NC-17'}
    if rating_str in valid:
        return rating_str
    
    mapping = {
        'TV-MA': 'R',
        'TV-14': 'PG-13',
        'TV-PG': 'PG',
        'TV-Y': 'G',
        'TV-Y7': 'G',
        'TV-Y7-FV': 'G',
        'TV-G': 'G',
        'NR': 'PG',
        'UR': 'R'
    }
    return mapping.get(rating_str, 'PG')

def generate_seed_sql(csv_path='netflix_titles.csv', output_sql='seed.sql', max_titles=350):
    print(f"Reading {csv_path}...")
    
    movies_data = []
    series_data = []

    with open(csv_path, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f)
        for row in reader:
            title = row['title'].strip()[:245]
            raw_type = row['type'].strip()
            rating = row['rating'].strip()
            duration = row['duration'].strip() if row['duration'] else ""
            desc = row['description'].strip()
            release_year = row['release_year'].strip()

            # Handle edge case where duration was shifted to rating
            if 'min' in rating and not duration:
                duration = rating
                rating = 'R'

            mapped_rat = map_rating(rating)

            if raw_type == 'Movie':
                # parse duration "90 min"
                m = re.search(r'(\d+)', duration)
                minutes = int(m.group(1)) if m else 90
                duration_sec = minutes * 60
                movies_data.append({
                    'title': title,
                    'rating': mapped_rat,
                    'duration_sec': duration_sec,
                    'synopsis': desc,
                    'release_year': release_year if release_year.isdigit() else '2020'
                })
            else:
                m = re.search(r'(\d+)', duration)
                seasons = int(m.group(1)) if m else 1
                series_data.append({
                    'title': title,
                    'rating': mapped_rat,
                    'seasons': max(1, min(seasons, 5)), # cap at 5 for sample
                    'synopsis': desc,
                    'release_year': release_year if release_year.isdigit() else '2021'
                })

    print(f"Found {len(movies_data)} movies and {len(series_data)} TV series.")
    
    # Pick a balanced subset
    selected_movies = movies_data[:max_titles * 2 // 3]
    selected_series = series_data[:max_titles // 3]

    print(f"Selected {len(selected_movies)} movies and {len(selected_series)} series for seed.sql.")

    resolutions = ['1080p', '4K UHD', '720p', '1080p']

    out = []
    out.append("-- ============================================================================")
    out.append("-- MINI-NETFLIX DATABASE SEED DATA")
    out.append("-- Derived from Netflix Movies and TV Shows Dataset (HuggingFace/Kaggle)")
    out.append("-- Synthesized & mapped to match schema.sql relational model")
    out.append("-- ============================================================================")
    out.append("SET FOREIGN_KEY_CHECKS = 0;")
    out.append("TRUNCATE TABLE active_session;")
    out.append("TRUNCATE TABLE watch_history;")
    out.append("TRUNCATE TABLE title_rating;")
    out.append("TRUNCATE TABLE watchlist;")
    out.append("TRUNCATE TABLE episode;")
    out.append("TRUNCATE TABLE season;")
    out.append("TRUNCATE TABLE tv_series;")
    out.append("TRUNCATE TABLE movie_title;")
    out.append("TRUNCATE TABLE streamable_media_unit;")
    out.append("TRUNCATE TABLE catalog_title;")
    out.append("TRUNCATE TABLE profile;")
    out.append("TRUNCATE TABLE account;")
    out.append("TRUNCATE TABLE subscription_tier;")
    out.append("SET FOREIGN_KEY_CHECKS = 1;")
    out.append("")

    # 1. SUBSCRIPTION TIERS
    out.append("-- 1. SUBSCRIPTION TIERS")
    out.append("INSERT INTO subscription_tier (tier_id, tier_name, monthly_price, max_concurrent_streams, max_video_quality) VALUES")
    out.append("  (0, 'Free', 0.00, 1, '480p'),")
    out.append("  (1, 'Basic', 9.99, 1, '720p'),")
    out.append("  (2, 'Standard', 15.49, 2, '1080p'),")
    out.append("  (3, 'Premium', 19.99, 4, '4K UHD');")
    out.append("")

    # 2. ACCOUNTS
    out.append("-- 2. ACCOUNTS (Testing various tiers and statuses)")
    out.append("INSERT INTO account (account_id, email, password_hash, billing_status, current_tier_id, next_billing_date) VALUES")
    out.append("  (1, 'alice.premium@ptithcm.edu.vn', '$2a$12$eXampleHashedPasswordAlice', 'ACTIVE', 3, DATE_ADD(CURRENT_DATE(), INTERVAL 25 DAY)),")
    out.append("  (2, 'bob.standard@ptithcm.edu.vn', '$2a$12$eXampleHashedPasswordBob', 'ACTIVE', 2, DATE_ADD(CURRENT_DATE(), INTERVAL 14 DAY)),")
    out.append("  (3, 'charlie.basic@ptithcm.edu.vn', '$2a$12$eXampleHashedPasswordCharlie', 'ACTIVE', 1, DATE_ADD(CURRENT_DATE(), INTERVAL 10 DAY)),")
    out.append("  (4, 'david.free@ptithcm.edu.vn', '$2a$12$eXampleHashedPasswordDavid', 'ACTIVE', 0, DATE_ADD(CURRENT_DATE(), INTERVAL 30 DAY)),")
    out.append("  (5, 'emma.cancelled@ptithcm.edu.vn', '$2a$12$eXampleHashedPasswordEmma', 'SUSPENDED', 1, CURRENT_DATE());")
    out.append("")

    # 3. PROFILES
    out.append("-- 3. PROFILES (Adult vs Kids profiles to test BR-03, BR-04, BR-05)")
    out.append("INSERT INTO profile (profile_id, account_id, profile_name, is_kids_profile) VALUES")
    out.append("  (1, 1, 'Alice (Main)', FALSE),")
    out.append("  (2, 1, 'Alice Junior (Kids)', TRUE),")
    out.append("  (3, 1, 'Alice Family', FALSE),")
    out.append("  (4, 2, 'Bob (Main)', FALSE),")
    out.append("  (5, 2, 'Bob Junior (Kids)', TRUE),")
    out.append("  (6, 3, 'Charlie (Basic)', FALSE),")
    out.append("  (7, 4, 'David (Free)', FALSE),")
    out.append("  (8, 5, 'Emma (Suspended)', FALSE);")
    out.append("")

    # 4. MOVIES
    out.append("-- 4. CATALOG TITLES & STREAMABLE MEDIA UNITS (MOVIES)")
    catalog_inserts = []
    media_inserts = []
    movie_links = []
    
    first_few_title_ids = []
    first_few_media_ids = []

    for i, m in enumerate(selected_movies):
        t_id = str(uuid.uuid4())
        m_id = str(uuid.uuid4())
        if len(first_few_title_ids) < 10:
            first_few_title_ids.append(t_id)
            first_few_media_ids.append(m_id)
        
        # Distribute minimum_tier_id across 0, 1, 2, 3
        tier_req = i % 4
        res = resolutions[i % len(resolutions)]
        url = f"https://cdn.mininetflix.internal/assets/movies/{m_id}.mp4"

        catalog_inserts.append(
            f"({escape_sql(t_id)}, {tier_req}, {escape_sql(m['title'])}, '{m['rating']}', 'Movie', {escape_sql(m['synopsis'])})"
        )
        media_inserts.append(
            f"({escape_sql(m_id)}, {m['duration_sec']}, '{res}', FALSE, {escape_sql(url)})"
        )
        movie_links.append(
            f"({escape_sql(t_id)}, {escape_sql(m_id)})"
        )

    # 5. SERIES
    series_catalog_inserts = []
    tv_series_inserts = []
    season_inserts = []
    episode_inserts = []

    for j, s in enumerate(selected_series):
        t_id = str(uuid.uuid4())
        if len(first_few_title_ids) < 15:
            first_few_title_ids.append(t_id)
        tier_req = (j + 1) % 4
        series_catalog_inserts.append(
            f"({escape_sql(t_id)}, {tier_req}, {escape_sql(s['title'])}, '{s['rating']}', 'Series', {escape_sql(s['synopsis'])})"
        )
        tv_series_inserts.append(
            f"({escape_sql(t_id)}, {s['seasons']})"
        )

        for sn in range(1, s['seasons'] + 1):
            s_id = str(uuid.uuid4())
            season_inserts.append(
                f"({escape_sql(s_id)}, {escape_sql(t_id)}, {sn}, {s['release_year']})"
            )
            # Create 2 episodes for season 1
            if sn == 1:
                for ep in [1, 2]:
                    ep_media_id = str(uuid.uuid4())
                    ep_id = str(uuid.uuid4())
                    ep_url = f"https://cdn.mininetflix.internal/assets/episodes/{ep_media_id}.mp4"
                    media_inserts.append(
                        f"({escape_sql(ep_media_id)}, 2700, '1080p', FALSE, {escape_sql(ep_url)})"
                    )
                    episode_inserts.append(
                        f"({escape_sql(ep_id)}, {escape_sql(s_id)}, {escape_sql(ep_media_id)}, {ep}, 'Episode {ep}')"
                    )

    # Write Batched SQL
    def write_batches(table, cols, rows, batch_size=50):
        res = []
        for idx in range(0, len(rows), batch_size):
            chunk = rows[idx:idx+batch_size]
            res.append(f"INSERT INTO {table} ({cols}) VALUES\n  " + ",\n  ".join(chunk) + ";\n")
        return res

    out.extend(write_batches("catalog_title", "title_id, minimum_tier_id, title_name, maturity_rating, content_type, synopsis", catalog_inserts + series_catalog_inserts))
    out.extend(write_batches("streamable_media_unit", "media_unit_id, duration_seconds, video_resolution, is_demo_or_trailer, storage_asset_url", media_inserts))
    out.extend(write_batches("movie_title", "title_id, media_unit_id", movie_links))
    out.extend(write_batches("tv_series", "title_id, total_seasons", tv_series_inserts))
    out.extend(write_batches("season", "season_id, series_id, season_number, release_year", season_inserts))
    out.extend(write_batches("episode", "episode_id, season_id, media_unit_id, episode_number, episode_title", episode_inserts))

    # 6. WATCHLIST
    out.append("-- 6. WATCHLIST (Sample entries for profiles)")
    if len(first_few_title_ids) >= 6:
        out.append(f"INSERT INTO watchlist (profile_id, title_id) VALUES")
        out.append(f"  (1, '{first_few_title_ids[0]}'),")
        out.append(f"  (1, '{first_few_title_ids[1]}'),")
        out.append(f"  (1, '{first_few_title_ids[2]}'),")
        out.append(f"  (4, '{first_few_title_ids[3]}'),")
        out.append(f"  (6, '{first_few_title_ids[4]}');\n")

    # 7. TITLE RATINGS
    out.append("-- 7. TITLE RATINGS (Scores 1-10)")
    if len(first_few_title_ids) >= 6:
        out.append(f"INSERT INTO title_rating (profile_id, title_id, score) VALUES")
        out.append(f"  (1, '{first_few_title_ids[0]}', 10),")
        out.append(f"  (1, '{first_few_title_ids[1]}', 8),")
        out.append(f"  (4, '{first_few_title_ids[2]}', 9),")
        out.append(f"  (6, '{first_few_title_ids[3]}', 7);\n")

    # 8. WATCH HISTORY
    out.append("-- 8. WATCH HISTORY")
    if len(first_few_media_ids) >= 3:
        out.append(f"INSERT INTO watch_history (profile_id, media_unit_id, watched_at, playback_stop_seconds, is_completed) VALUES")
        out.append(f"  (1, '{first_few_media_ids[0]}', NOW() - INTERVAL 2 DAY, 5400, TRUE),")
        out.append(f"  (1, '{first_few_media_ids[1]}', NOW() - INTERVAL 1 DAY, 1800, FALSE),")
        out.append(f"  (4, '{first_few_media_ids[2]}', NOW() - INTERVAL 3 HOUR, 3600, TRUE);\n")

    # 9. ACTIVE SESSIONS
    out.append("-- 9. ACTIVE SESSIONS")
    out.append(f"INSERT INTO active_session (session_id, account_id, prof_id, ip_address, device_fingerprint, session_status) VALUES")
    out.append(f"  (UUID(), 1, 1, '127.0.0.1', 'web-chrome-desktop-fp-001', 'ACTIVE');\n")

    with open(output_sql, 'w', encoding='utf-8') as f:
        f.write("\n".join(out))

    print(f"Successfully generated {output_sql} with {len(out)} lines!")

if __name__ == '__main__':
    generate_seed_sql()

