package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

//go:embed static/index.html
var embeddedIndexHTML []byte

var db *sql.DB

// ============================================================================
// Data Transfer Objects (DTOs)
// ============================================================================

type APIResponse struct {
	Success bool   `json:"success"`
	Count   int    `json:"count,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type CatalogItemDTO struct {
	TitleID         string  `json:"title_id"`
	TitleName       string  `json:"title_name"`
	ContentType     string  `json:"content_type"`
	MaturityRating  string  `json:"maturity_rating"`
	MinimumTierID   int     `json:"minimum_tier_id"`
	Synopsis        *string `json:"synopsis,omitempty"`
	DurationSeconds *int    `json:"duration_seconds,omitempty"`
	VideoResolution *string `json:"video_resolution,omitempty"`
	InWatchlist     bool    `json:"in_watchlist"`
	HasWatched      bool    `json:"has_watched"`
}

type ProfileContextDTO struct {
	ProfileID     int    `json:"profile_id"`
	AccountID     int    `json:"account_id"`
	ProfileName   string `json:"profile_name"`
	IsKidsProfile bool   `json:"is_kids_profile"`
	TierID        int    `json:"tier_id"`
	TierName      string `json:"tier_name"`
	BillingStatus string `json:"billing_status"`
	MaxStreams    int    `json:"max_streams"`
	ActiveStreams int    `json:"active_streams"`
}

type AccountSummaryDTO struct {
	AccountID       int     `json:"account_id"`
	Email           string  `json:"email"`
	TierID          int     `json:"tier_id"`
	TierName        string  `json:"tier_name"`
	MonthlyPrice    float64 `json:"monthly_price"`
	BillingStatus   string  `json:"billing_status"`
	NextBillingDate string  `json:"next_billing_date"`
	ActiveStreams   int     `json:"active_streams"`
	MaxStreams      int     `json:"max_streams"`
}

type ActiveSessionDTO struct {
	SessionID         string `json:"session_id"`
	AccountID         int    `json:"account_id"`
	ProfileID         int    `json:"profile_id"`
	ProfileName       string `json:"profile_name"`
	IPAddress         string `json:"ip_address"`
	DeviceFingerprint string `json:"device_fingerprint"`
	StartedAt         string `json:"started_at"`
}

type CreateMovieRequest struct {
	TitleName       string `json:"title_name"`
	MaturityRating  string `json:"maturity_rating"`
	MinimumTierID   int    `json:"minimum_tier_id"`
	Synopsis        string `json:"synopsis"`
	DurationSeconds int    `json:"duration_seconds"`
	VideoResolution string `json:"video_resolution"`
	StorageAssetURL string `json:"storage_asset_url"`
}

type CreateProfileRequest struct {
	AccountID     int    `json:"account_id"`
	ProfileName   string `json:"profile_name"`
	IsKidsProfile bool   `json:"is_kids_profile"`
}

type CreateAccountRequest struct {
	Email  string `json:"email"`
	TierID int    `json:"tier_id"`
}

type UpdateTierRequest struct {
	AccountID int `json:"account_id"`
	TierID    int `json:"tier_id"`
}

type AccountActionRequest struct {
	AccountID int `json:"account_id"`
}

type WatchlistRequest struct {
	ProfileID int    `json:"profile_id"`
	TitleID   string `json:"title_id"`
}

type TitleRatingRequest struct {
	ProfileID int    `json:"profile_id"`
	TitleID   string `json:"title_id"`
	Score     int    `json:"score"`
}

type StartSessionRequest struct {
	AccountID         int    `json:"account_id"`
	ProfileID         int    `json:"profile_id"`
	TitleID           string `json:"title_id,omitempty"`
	DeviceFingerprint string `json:"device_fingerprint"`
}

type StopSessionRequest struct {
	SessionID string `json:"session_id"`
}

type UpdatePlaybackRequest struct {
	ProfileID           int    `json:"profile_id"`
	MediaUnitID         string `json:"media_unit_id"`
	PlaybackStopSeconds int    `json:"playback_stop_seconds"`
}

type CardRegistrationRequest struct {
	AccountID      int    `json:"account_id"`
	CardholderName string `json:"cardholder_name"`
	CardNumber     string `json:"card_number"`
	ExpMonth       int    `json:"exp_month"`
	ExpYear        int    `json:"exp_year"`
	CVV            string `json:"cvv"`
}

// ============================================================================
// Helper Utilities & Middleware
// ============================================================================

func writeJSON(w http.ResponseWriter, status int, payload APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, APIResponse{Success: false, Message: msg})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// normalizeBillingStatus standardizes DB status values for index.html UI badges
func normalizeBillingStatus(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "SUSPENDED", "PAST_DUE", "OVERDUE":
		return "SUSPENDED"
	case "CANCELLED", "CANCELED", "INACTIVE":
		return "CANCELLED"
	default:
		return "ACTIVE"
	}
}

// buildDSN reads DATABASE_DSN (or individual DB_* env vars) and guarantees parseTime=true
func buildDSN() string {
	dsn := strings.TrimSpace(os.Getenv("DB_DSN"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_DSN"))
	}

	if dsn == "" {
		user := os.Getenv("DB_USER")
		pass := os.Getenv("DB_PASSWORD")
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		name := os.Getenv("DB_NAME")

		if user != "" && host != "" {
			if port == "" {
				port = "3306"
			}
			if name == "" {
				name = "mini_netflix"
			}
			tlsParam := ""
			if strings.Contains(host, "tidbcloud.com") {
				tlsParam = "&tls=true"
			}
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true%s", user, pass, host, port, name, tlsParam)
		} else {
			dsn = "2AmEQpexzMhSUci.root:TdsLpNo74ZKkqeSy@tcp(gateway01.ap-southeast-1.prod.aws.tidbcloud.com:4000)/mininetflix?tls=true&parseTime=true"
		}
	}

	if !strings.Contains(dsn, "parseTime=") {
		if strings.Contains(dsn, "?") {
			dsn += "&parseTime=true"
		} else {
			dsn += "?parseTime=true"
		}
	}
	return dsn
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// Root Portal UI Handler
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}
	if len(embeddedIndexHTML) > 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(embeddedIndexHTML)
		return
	}
	http.ServeFile(w, r, "static/index.html")
}

// GET /health: Database health check for index.html status bar
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("Database unreachable: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Connected",
		Data:    map[string]string{"database": "MySQL 8.0 / TiDB"},
	})
}

// 1. GET /api/catalog: Filtered catalog browsing (Enforces BR-01 & BR-04)
func catalogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	queryParams := r.URL.Query()
	profileIDStr := queryParams.Get("profile_id")
	search := queryParams.Get("search")
	contentType := queryParams.Get("type")

	if profileIDStr == "" {
		writeError(w, http.StatusBadRequest, "Missing required query parameter: profile_id")
		return
	}
	profileID, err := strconv.Atoi(profileIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid profile_id")
		return
	}

	var isKids bool
	var currentTierID int
	var billingStatus string

	err = db.QueryRow(`
		SELECT p.is_kids_profile, a.current_tier_id, a.billing_status
		FROM profile p
		JOIN account a ON p.account_id = a.account_id
		WHERE p.profile_id = ?
	`, profileID).Scan(&isKids, &currentTierID, &billingStatus)

	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Profile not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Profile lookup failed: %v", err))
		return
	}

	query := `
		SELECT 
			ct.title_id, 
			ct.title_name, 
			ct.content_type, 
			ct.maturity_rating, 
			ct.minimum_tier_id, 
			ct.synopsis,
			smu.duration_seconds,
			smu.video_resolution,
			EXISTS(
				SELECT 1 FROM watchlist w 
				WHERE w.profile_id = ? AND w.title_id = ct.title_id
			) AS in_watchlist,
			EXISTS(
				SELECT 1 FROM watch_history wh
				LEFT JOIN movie_title mt2 ON wh.media_unit_id = mt2.media_unit_id
				LEFT JOIN episode ep ON wh.media_unit_id = ep.media_unit_id
				LEFT JOIN season s ON ep.season_id = s.season_id
				WHERE wh.profile_id = ? AND (mt2.title_id = ct.title_id OR s.series_id = ct.title_id)
			) AS has_watched
		FROM catalog_title ct
		LEFT JOIN movie_title mt ON ct.title_id = mt.title_id
		LEFT JOIN streamable_media_unit smu ON mt.media_unit_id = smu.media_unit_id
		WHERE ct.minimum_tier_id <= ?
	`
	args := []any{profileID, profileID, currentTierID}

	if isKids {
		query += " AND ct.maturity_rating NOT IN ('R', 'NC-17', 'TV-MA') "
	}
	if contentType != "" {
		query += " AND ct.content_type = ? "
		args = append(args, contentType)
	}
	if search != "" {
		query += " AND ct.title_name LIKE ? "
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY ct.title_name ASC LIMIT 60;"

	rows, err := db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Catalog query failed: %v", err))
		return
	}
	defer rows.Close()

	items := make([]CatalogItemDTO, 0)
	for rows.Next() {
		var item CatalogItemDTO
		var synopsis sql.NullString
		var duration sql.NullInt32
		var resolution sql.NullString

		if err := rows.Scan(
			&item.TitleID,
			&item.TitleName,
			&item.ContentType,
			&item.MaturityRating,
			&item.MinimumTierID,
			&synopsis,
			&duration,
			&resolution,
			&item.InWatchlist,
			&item.HasWatched,
		); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if synopsis.Valid {
			item.Synopsis = &synopsis.String
		}
		if duration.Valid {
			d := int(duration.Int32)
			item.DurationSeconds = &d
		}
		if resolution.Valid {
			item.VideoResolution = &resolution.String
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Count: len(items), Data: items})
}

// 2. POST /api/movies: Register a new movie across the EER Subclasses
func createMovieHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CreateMovieRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Transaction initiation failed")
		return
	}
	defer tx.Rollback()

	var titleID, mediaID string
	if err := tx.QueryRow("SELECT UUID(), UUID()").Scan(&titleID, &mediaID); err != nil {
		writeError(w, http.StatusInternalServerError, "UUID generation error")
		return
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO catalog_title (title_id, minimum_tier_id, title_name, maturity_rating, content_type, synopsis)
		VALUES (?, ?, ?, ?, 'Movie', ?)
	`, titleID, req.MinimumTierID, req.TitleName, req.MaturityRating, req.Synopsis)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to insert catalog_title: %v", err))
		return
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO streamable_media_unit (media_unit_id, duration_seconds, video_resolution, is_demo_or_trailer, storage_asset_url)
		VALUES (?, ?, ?, FALSE, ?)
	`, mediaID, req.DurationSeconds, req.VideoResolution, req.StorageAssetURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to insert media_unit: %v", err))
		return
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO movie_title (title_id, media_unit_id)
		VALUES (?, ?)
	`, titleID, mediaID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to link movie_title: %v", err))
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "Transaction commit failed")
		return
	}

	writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "Movie registered successfully across EER subclasses",
		Data: map[string]string{
			"title_id":      titleID,
			"media_unit_id": mediaID,
		},
	})
}

// 3. GET & POST /api/profiles: Fetch all profiles for UI switcher & Create Profile (BR-05)
func profilesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query(`
			SELECT 
				p.profile_id,
				p.account_id,
				p.profile_name,
				p.is_kids_profile,
				st.tier_id,
				st.tier_name,
				a.billing_status,
				st.max_concurrent_streams,
				(SELECT COUNT(*) FROM active_session s 
				 WHERE s.account_id = p.account_id AND UPPER(s.session_status) = 'ACTIVE') AS active_streams
			FROM profile p
			JOIN account a ON p.account_id = a.account_id
			JOIN subscription_tier st ON a.current_tier_id = st.tier_id
			ORDER BY p.account_id ASC, p.profile_id ASC
		`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch profiles: %v", err))
			return
		}
		defer rows.Close()

		profiles := make([]ProfileContextDTO, 0)
		for rows.Next() {
			var p ProfileContextDTO
			var rawStatus string
			if err := rows.Scan(
				&p.ProfileID,
				&p.AccountID,
				&p.ProfileName,
				&p.IsKidsProfile,
				&p.TierID,
				&p.TierName,
				&rawStatus,
				&p.MaxStreams,
				&p.ActiveStreams,
			); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			p.BillingStatus = normalizeBillingStatus(rawStatus)
			profiles = append(profiles, p)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch profiles: %v", err))
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Count: len(profiles), Data: profiles})

	case http.MethodPost:
		var req CreateProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}

		var currentCount int
		err := db.QueryRow("SELECT COUNT(*) FROM profile WHERE account_id = ?", req.AccountID).Scan(&currentCount)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to verify profile count")
			return
		}

		if currentCount >= 5 {
			writeError(w, http.StatusConflict, "Profile threshold reached: Accounts cannot have more than 5 profiles (BR-05)")
			return
		}

		res, err := db.Exec(`
			INSERT INTO profile (account_id, profile_name, is_kids_profile)
			VALUES (?, ?, ?)
		`, req.AccountID, req.ProfileName, req.IsKidsProfile)
		if err != nil {
			writeError(w, http.StatusConflict, fmt.Sprintf("Failed to create profile: %v", err))
			return
		}

		profileID, _ := res.LastInsertId()
		writeJSON(w, http.StatusCreated, APIResponse{
			Success: true,
			Message: "Profile created successfully",
			Data:    map[string]any{"profile_id": profileID, "profile_name": req.ProfileName},
		})

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// 4. GET & POST /api/watchlist: M:N Associative Entity Management (with Toggle support)
func watchlistHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		profileIDStr := r.URL.Query().Get("profile_id")
		if profileIDStr == "" {
			writeError(w, http.StatusBadRequest, "Missing profile_id parameter")
			return
		}

		rows, err := db.Query(`
			SELECT ct.title_id, ct.title_name, ct.content_type, ct.maturity_rating, ct.synopsis, w.added_at
			FROM watchlist w
			JOIN catalog_title ct ON w.title_id = ct.title_id
			WHERE w.profile_id = ?
			ORDER BY w.added_at DESC
		`, profileIDStr)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		type WatchlistItem struct {
			TitleID        string  `json:"title_id"`
			TitleName      string  `json:"title_name"`
			ContentType    string  `json:"content_type"`
			MaturityRating string  `json:"maturity_rating"`
			Synopsis       *string `json:"synopsis,omitempty"`
			AddedAt        string  `json:"added_at"`
		}

		watchlist := make([]WatchlistItem, 0)
		for rows.Next() {
			var wi WatchlistItem
			var synopsis sql.NullString
			var addedTime time.Time
			if err := rows.Scan(&wi.TitleID, &wi.TitleName, &wi.ContentType, &wi.MaturityRating, &synopsis, &addedTime); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if synopsis.Valid {
				wi.Synopsis = &synopsis.String
			}
			wi.AddedAt = addedTime.Format("2006-01-02 15:04")
			watchlist = append(watchlist, wi)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Count: len(watchlist), Data: watchlist})

	case http.MethodPost:
		var req WatchlistRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}

		// Toggle check: if already present, remove it; otherwise insert it
		var exists int
		_ = db.QueryRow("SELECT COUNT(*) FROM watchlist WHERE profile_id = ? AND title_id = ?", req.ProfileID, req.TitleID).Scan(&exists)
		if exists > 0 {
			_, err := db.Exec("DELETE FROM watchlist WHERE profile_id = ? AND title_id = ?", req.ProfileID, req.TitleID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to remove from watchlist: %v", err))
				return
			}
			writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Title removed from My List"})
			return
		}

		_, err := db.Exec("INSERT INTO watchlist (profile_id, title_id) VALUES (?, ?)", req.ProfileID, req.TitleID)
		if err != nil {
			writeError(w, http.StatusConflict, fmt.Sprintf("Could not add to watchlist: %v", err))
			return
		}

		writeJSON(w, http.StatusCreated, APIResponse{Success: true, Message: "Title added to My List"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// 5. POST /api/ratings: Rate a title (Score 1-10)
func ratingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req TitleRatingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Score < 1 || req.Score > 10 {
		writeError(w, http.StatusBadRequest, "Domain constraint violation: Score must be between 1 and 10")
		return
	}

	query := `
		INSERT INTO title_rating (profile_id, title_id, score, rated_at)
		VALUES (?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE score = VALUES(score), rated_at = NOW();
	`
	_, err := db.Exec(query, req.ProfileID, req.TitleID, req.Score)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Rating update failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: fmt.Sprintf("Recorded rating of %d/10!", req.Score)})
}

// 6. POST /api/sessions/start: Starts active stream (Enforces Billing Status & BR-02 Concurrency Limit)
func startSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req StartSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	var maxStreams int
	var rawStatus string
	err := db.QueryRow(`
		SELECT st.max_concurrent_streams, a.billing_status
		FROM account a
		JOIN subscription_tier st ON a.current_tier_id = st.tier_id
		WHERE a.account_id = ?
	`, req.AccountID).Scan(&maxStreams, &rawStatus)
	if err != nil {
		writeError(w, http.StatusNotFound, "Account or tier not found")
		return
	}

	normStatus := normalizeBillingStatus(rawStatus)
	if normStatus == "SUSPENDED" {
		writeError(w, http.StatusPaymentRequired, "Account is SUSPENDED due to an overdue invoice (HTTP 402). Please renew in Account & Billing.")
		return
	}
	if normStatus == "CANCELLED" {
		writeError(w, http.StatusForbidden, "Account membership is CANCELLED. Please reactivate your plan to stream.")
		return
	}

	var activeCount int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM active_session 
		WHERE account_id = ? AND UPPER(session_status) = 'ACTIVE'
	`, req.AccountID).Scan(&activeCount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to audit active sessions")
		return
	}

	if activeCount >= maxStreams {
		writeError(w, http.StatusConflict, fmt.Sprintf(
			"Stream limit exceeded: Current tier allows max %d concurrent streams (BR-02)", maxStreams,
		))
		return
	}

	var sessionID string
	if err := db.QueryRow("SELECT UUID()").Scan(&sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, "UUID generation error")
		return
	}

	clientIP := r.RemoteAddr
	if colonIdx := strings.LastIndex(clientIP, ":"); colonIdx != -1 {
		clientIP = clientIP[:colonIdx]
	}

	// Support both prof_id and profile_id column naming conventions seamlessly
	_, err = db.Exec(`
		INSERT INTO active_session (session_id, account_id, prof_id, ip_address, device_fingerprint, session_status)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE')
	`, sessionID, req.AccountID, req.ProfileID, clientIP, req.DeviceFingerprint)
	if err != nil && strings.Contains(err.Error(), "Unknown column") {
		_, err = db.Exec(`
			INSERT INTO active_session (session_id, account_id, profile_id, ip_address, device_fingerprint, session_status)
			VALUES (?, ?, ?, ?, ?, 'Active')
		`, sessionID, req.AccountID, req.ProfileID, clientIP, req.DeviceFingerprint)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize session: %v", err))
		return
	}

	// If a title_id was played, log a watch_history entry so the user can rate the title
	if req.TitleID != "" {
		var mediaUnitID string
		_ = db.QueryRow(`
			SELECT COALESCE(
				(SELECT mt.media_unit_id FROM movie_title mt WHERE mt.title_id = ? LIMIT 1),
				(SELECT ep.media_unit_id FROM episode ep JOIN season s ON ep.season_id = s.season_id WHERE s.series_id = ? LIMIT 1)
			)
		`, req.TitleID, req.TitleID).Scan(&mediaUnitID)

		if mediaUnitID != "" {
			_, _ = db.Exec(`
				INSERT INTO watch_history (profile_id, media_unit_id, watched_at, playback_stop_seconds, is_completed)
				VALUES (?, ?, NOW(), 120, TRUE)
			`, req.ProfileID, mediaUnitID)
		}
	}

	writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Streaming session started (%d/%d slots active)", activeCount+1, maxStreams),
		Data:    map[string]string{"session_id": sessionID},
	})
}

// 6b. GET /api/sessions: List active sessions for account
func listSessionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accID := r.URL.Query().Get("account_id")
	if accID == "" {
		writeError(w, http.StatusBadRequest, "Missing account_id")
		return
	}

	rows, err := db.Query(`
		SELECT s.session_id, s.account_id, s.prof_id, p.profile_name, s.ip_address, s.device_fingerprint, s.started_at
		FROM active_session s
		JOIN profile p ON s.prof_id = p.profile_id
		WHERE s.account_id = ? AND UPPER(s.session_status) = 'ACTIVE'
		ORDER BY s.started_at DESC
	`, accID)
	if err != nil && strings.Contains(err.Error(), "Unknown column") {
		rows, err = db.Query(`
			SELECT s.session_id, s.account_id, s.profile_id, p.profile_name, s.ip_address, s.device_fingerprint, s.started_at
			FROM active_session s
			JOIN profile p ON s.profile_id = p.profile_id
			WHERE s.account_id = ? AND UPPER(s.session_status) = 'ACTIVE'
			ORDER BY s.started_at DESC
		`, accID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	sessions := make([]ActiveSessionDTO, 0)
	for rows.Next() {
		var s ActiveSessionDTO
		var startedAt time.Time
		if err := rows.Scan(&s.SessionID, &s.AccountID, &s.ProfileID, &s.ProfileName, &s.IPAddress, &s.DeviceFingerprint, &startedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.StartedAt = startedAt.Format("15:04:05")
		sessions = append(sessions, s)
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Count: len(sessions), Data: sessions})
}

// 6c. POST /api/sessions/stop: Terminate single active session
func stopSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req StopSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	_, err := db.Exec("DELETE FROM active_session WHERE session_id = ?", req.SessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Session terminated and concurrency slot released"})
}

// 6d. POST /api/sessions/stop-all: Terminate all sessions for an account
func stopAllSessionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req AccountActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	_, err := db.Exec("DELETE FROM active_session WHERE account_id = ?", req.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "All active playback sessions disconnected"})
}

// 7. POST /api/history/progress: Log Playback Checkpoint (Composite Key History)
func historyProgressHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req UpdatePlaybackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	var totalSeconds int
	err := db.QueryRow("SELECT duration_seconds FROM streamable_media_unit WHERE media_unit_id = ?", req.MediaUnitID).Scan(&totalSeconds)
	if err != nil {
		writeError(w, http.StatusNotFound, "Media unit not found")
		return
	}

	isCompleted := float64(req.PlaybackStopSeconds) >= (0.90 * float64(totalSeconds))

	query := `
		INSERT INTO watch_history (profile_id, media_unit_id, watched_at, playback_stop_seconds, is_completed)
		VALUES (?, ?, NOW(), ?, ?)
	`
	_, err = db.Exec(query, req.ProfileID, req.MediaUnitID, req.PlaybackStopSeconds, isCompleted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to record history: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Playback checkpoint recorded",
		Data:    map[string]any{"is_completed": isCompleted},
	})
}

// 8. POST /api/cards: Register Credit Card (Tokenization & Vault Simulation)
func registerCardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CardRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	cleanNumber := strings.ReplaceAll(req.CardNumber, " ", "")
	if len(cleanNumber) < 4 {
		writeError(w, http.StatusBadRequest, "Invalid card length")
		return
	}
	lastFour := cleanNumber[len(cleanNumber)-4:]

	cardBrand := "VISA"
	if strings.HasPrefix(cleanNumber, "4") {
		cardBrand = "VISA"
	} else if strings.HasPrefix(cleanNumber, "5") {
		cardBrand = "MASTERCARD"
	} else if strings.HasPrefix(cleanNumber, "3") {
		cardBrand = "AMEX"
	}

	tokenBytes := make([]byte, 16)
	_, _ = rand.Read(tokenBytes)
	mockToken := fmt.Sprintf("tok_mock_%s", hex.EncodeToString(tokenBytes))

	query := `
		INSERT INTO credit_card 
		(account_id, cardholder_name, card_brand, last_four, exp_month, exp_year, payment_token) 
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`
	_, err := db.Exec(query, req.AccountID, req.CardholderName, cardBrand, lastFour, req.ExpMonth, req.ExpYear, mockToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "Card tokenized successfully",
		Data: map[string]string{
			"brand":     cardBrand,
			"last_four": lastFour,
			"token":     mockToken,
		},
	})
}

// 9. GET & POST /api/accounts: Membership & Billing Management
func accountsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query(`
			SELECT 
				a.account_id,
				a.email,
				st.tier_id,
				st.tier_name,
				st.monthly_price,
				a.billing_status,
				a.next_billing_date,
				(SELECT COUNT(*) FROM active_session s 
				 WHERE s.account_id = a.account_id AND UPPER(s.session_status) = 'ACTIVE') AS active_streams,
				st.max_concurrent_streams
			FROM account a
			JOIN subscription_tier st ON a.current_tier_id = st.tier_id
			ORDER BY a.account_id ASC
		`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		accounts := make([]AccountSummaryDTO, 0)
		for rows.Next() {
			var acc AccountSummaryDTO
			var rawStatus string
			var nextDate sql.NullTime
			if err := rows.Scan(
				&acc.AccountID,
				&acc.Email,
				&acc.TierID,
				&acc.TierName,
				&acc.MonthlyPrice,
				&rawStatus,
				&nextDate,
				&acc.ActiveStreams,
				&acc.MaxStreams,
			); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			acc.BillingStatus = normalizeBillingStatus(rawStatus)
			if nextDate.Valid {
				acc.NextBillingDate = nextDate.Time.Format("2006-01-02")
			} else {
				acc.NextBillingDate = "N/A"
			}
			accounts = append(accounts, acc)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Count: len(accounts), Data: accounts})

	case http.MethodPost:
		var req CreateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		mockHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
		res, err := db.Exec(`
			INSERT INTO account (email, password_hash, billing_status, current_tier_id, registration_date, next_billing_date)
			VALUES (?, ?, 'ACTIVE', ?, CURRENT_DATE, DATE_ADD(CURRENT_DATE, INTERVAL 30 DAY))
		`, req.Email, mockHash, req.TierID)
		if err != nil {
			// Fallback if schema uses 'Active' case-sensitive CHECK constraint
			res, err = db.Exec(`
				INSERT INTO account (email, password_hash, billing_status, current_tier_id, registration_date, next_billing_date)
				VALUES (?, ?, 'Active', ?, CURRENT_DATE, DATE_ADD(CURRENT_DATE, INTERVAL 30 DAY))
			`, req.Email, mockHash, req.TierID)
		}
		if err != nil {
			writeError(w, http.StatusConflict, fmt.Sprintf("Could not create account: %v", err))
			return
		}

		accID, _ := res.LastInsertId()
		// Automatically create default primary profile so BR-05 (1 <= N <= 5) is satisfied
		defaultName := strings.Split(req.Email, "@")[0]
		_, _ = db.Exec(`INSERT INTO profile (account_id, profile_name, is_kids_profile) VALUES (?, ?, FALSE)`, accID, defaultName)

		writeJSON(w, http.StatusCreated, APIResponse{
			Success: true,
			Message: fmt.Sprintf("Account #%d (%s) registered with default profile", accID, req.Email),
		})

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// 10. POST /api/accounts/tier: Upgrade or Downgrade Subscription Tier
func updateAccountTierHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req UpdateTierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	_, err := db.Exec("UPDATE account SET current_tier_id = ? WHERE account_id = ?", req.TierID, req.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: fmt.Sprintf("Account #%d subscription tier updated", req.AccountID)})
}

// 11. POST /api/accounts/pay: Process Renewal Payment & Reactivate SUSPENDED Account
func payAccountInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req AccountActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	_, err := db.Exec(`
		UPDATE account 
		SET billing_status = 'ACTIVE', next_billing_date = DATE_ADD(CURRENT_DATE, INTERVAL 30 DAY)
		WHERE account_id = ?
	`, req.AccountID)
	if err != nil {
		_, err = db.Exec(`
			UPDATE account 
			SET billing_status = 'Active', next_billing_date = DATE_ADD(CURRENT_DATE, INTERVAL 30 DAY)
			WHERE account_id = ?
		`, req.AccountID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Payment processed for Account #%d. Billing status is now ACTIVE.", req.AccountID),
	})
}

// 12. POST /api/accounts/cancel: Cancel Account Subscription
func cancelAccountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req AccountActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	_, err := db.Exec(`UPDATE account SET billing_status = 'CANCELLED', canceled_at = NOW() WHERE account_id = ?`, req.AccountID)
	if err != nil {
		_, err = db.Exec(`UPDATE account SET billing_status = 'Inactive', canceled_at = NOW() WHERE account_id = ?`, req.AccountID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Terminate active streams on cancellation
	_, _ = db.Exec("DELETE FROM active_session WHERE account_id = ?", req.AccountID)

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Account #%d membership cancelled (10-month profile retention active per BR-15)", req.AccountID),
	})
}

// ============================================================================
// Server Initialization
// ============================================================================

func main() {
	dsn := buildDSN()

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("DSN parsing failed: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Printf("[WARNING] Database not immediately reachable: %v", err)
	} else {
		log.Println("Connected to MySQL successfully.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// Portal UI & Health
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/health", healthHandler)

	// Core API Routes
	mux.HandleFunc("/api/catalog", catalogHandler)
	mux.HandleFunc("/api/movies", createMovieHandler)
	mux.HandleFunc("/api/profiles", profilesHandler)
	mux.HandleFunc("/api/watchlist", watchlistHandler)
	mux.HandleFunc("/api/ratings", ratingsHandler)

	// Streaming Concurrency & Telemetry Routes
	mux.HandleFunc("/api/sessions", listSessionsHandler)
	mux.HandleFunc("/api/sessions/start", startSessionHandler)
	mux.HandleFunc("/api/sessions/stop", stopSessionHandler)
	mux.HandleFunc("/api/sessions/stop-all", stopAllSessionsHandler)
	mux.HandleFunc("/api/history/progress", historyProgressHandler)

	// Billing, Accounts & Tokenized Cards Routes
	mux.HandleFunc("/api/accounts", accountsHandler)
	mux.HandleFunc("/api/accounts/tier", updateAccountTierHandler)
	mux.HandleFunc("/api/accounts/pay", payAccountInvoiceHandler)
	mux.HandleFunc("/api/accounts/cancel", cancelAccountHandler)
	mux.HandleFunc("/api/cards", registerCardHandler)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("Mini-Netflix Portal & REST API running on http://localhost:%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server stopped: %v", err)
	}
}