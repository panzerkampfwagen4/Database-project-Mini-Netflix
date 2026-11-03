package main

import (
	"context"
	"database/sql"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

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
}

type CreateMovieRequest struct {
	TitleName       string `json:"title_name"`
	MaturityRating  string `json:"maturity_rating"` // 'G', 'PG', 'PG-13', 'R', 'NC-17'
	MinimumTierID   int    `json:"minimum_tier_id"`
	Synopsis        string `json:"synopsis"`
	DurationSeconds int    `json:"duration_seconds"`
	VideoResolution string `json:"video_resolution"` // '480p', '720p', '1080p', '4K UHD'
	StorageAssetURL string `json:"storage_asset_url"`
}

type CreateProfileRequest struct {
	AccountID     int    `json:"account_id"`
	ProfileName   string `json:"profile_name"`
	IsKidsProfile bool   `json:"is_kids_profile"`
}

type WatchlistRequest struct {
	ProfileID int    `json:"profile_id"`
	TitleID   string `json:"title_id"`
}

type TitleRatingRequest struct {
	ProfileID int    `json:"profile_id"`
	TitleID   string `json:"title_id"`
	Score     int    `json:"score"` // 1-10
}

type StartSessionRequest struct {
	AccountID         int    `json:"account_id"`
	ProfileID         int    `json:"profile_id"`
	DeviceFingerprint string `json:"device_fingerprint"`
}

type UpdatePlaybackRequest struct {
	ProfileID           int    `json:"profile_id"`
	MediaUnitID         string `json:"media_unit_id"`
	PlaybackStopSeconds int    `json:"playback_stop_seconds"`
}

type CardRegistrationRequest struct {
	AccountID      int    `json:"account_id"`
	CardholderName string `json:"cardholder_name"`
	CardNumber     string `json:"card_number"` // e.g., "4242424242424242"
	ExpMonth       int    `json:"exp_month"`
	ExpYear        int    `json:"exp_year"`
	CVV            string `json:"cvv"`         // e.g., "123"
}

// ============================================================================
// Helper Utilities
// ============================================================================

func writeJSON(w http.ResponseWriter, status int, payload APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, APIResponse{Success: false, Message: msg})
}

// ============================================================================
// HTTP Handlers
// ============================================================================

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

	// Retrieve Profile and Account Tier Context
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

	// Build Dynamic Parameterized Query
	query := `
		SELECT 
			ct.title_id, 
			ct.title_name, 
			ct.content_type, 
			ct.maturity_rating, 
			ct.minimum_tier_id, 
			ct.synopsis,
			smu.duration_seconds,
			smu.video_resolution
		FROM catalog_title ct
		LEFT JOIN movie_title mt ON ct.title_id = mt.title_id
		LEFT JOIN streamable_media_unit smu ON mt.media_unit_id = smu.media_unit_id
		WHERE ct.minimum_tier_id <= ?
	`
	args := []any{currentTierID}

	// BR-04: Kids Profile exclusion
	if isKids {
		query += " AND ct.maturity_rating NOT IN ('R', 'NC-17') "
	}

	// Content Type filter ('Movie' / 'Series')
	if contentType != "" {
		query += " AND ct.content_type = ? "
		args = append(args, contentType)
	}

	// Title search query
	if search != "" {
		query += " AND ct.title_name LIKE ? "
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY ct.title_name ASC LIMIT 50;"

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

	// Generate UUIDs for catalog and media entities
	var titleID, mediaID string
	if err := tx.QueryRow("SELECT UUID(), UUID()").Scan(&titleID, &mediaID); err != nil {
		writeError(w, http.StatusInternalServerError, "UUID generation error")
		return
	}

	// 1. Insert into catalog_title (Superclass)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO catalog_title (title_id, minimum_tier_id, title_name, maturity_rating, content_type, synopsis)
		VALUES (?, ?, ?, ?, 'Movie', ?)
	`, titleID, req.MinimumTierID, req.TitleName, req.MaturityRating, req.Synopsis)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to insert catalog_title: %v", err))
		return
	}

	// 2. Insert into streamable_media_unit
	_, err = tx.ExecContext(ctx, `
		INSERT INTO streamable_media_unit (media_unit_id, duration_seconds, video_resolution, is_demo_or_trailer, storage_asset_url)
		VALUES (?, ?, ?, FALSE, ?)
	`, mediaID, req.DurationSeconds, req.VideoResolution, req.StorageAssetURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to insert media_unit: %v", err))
		return
	}

	// 3. Link via movie_title (Option 8A Specialized Subclass)
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

// 3. POST /api/profiles: Create Profile (Enforces BR-05: 1 <= Count <= 5)
func profilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CreateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// Check Profile threshold (BR-05)
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
}

// 4. POST & GET /api/watchlist: M:N Associative Entity Management
func watchlistHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		profileIDStr := r.URL.Query().Get("profile_id")
		if profileIDStr == "" {
			writeError(w, http.StatusBadRequest, "Missing profile_id parameter")
			return
		}

		rows, err := db.Query(`
			SELECT ct.title_id, ct.title_name, ct.content_type, ct.maturity_rating, w.added_at
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
			TitleID        string    `json:"title_id"`
			TitleName      string    `json:"title_name"`
			ContentType    string    `json:"content_type"`
			MaturityRating string    `json:"maturity_rating"`
			AddedAt        time.Time `json:"added_at"`
		}

		watchlist := make([]WatchlistItem, 0)
		for rows.Next() {
			var wi WatchlistItem
			if err := rows.Scan(&wi.TitleID, &wi.TitleName, &wi.ContentType, &wi.MaturityRating, &wi.AddedAt); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
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

		_, err := db.Exec("INSERT INTO watchlist (profile_id, title_id) VALUES (?, ?)", req.ProfileID, req.TitleID)
		if err != nil {
			writeError(w, http.StatusConflict, fmt.Sprintf("Could not add to watchlist: %v", err))
			return
		}

		writeJSON(w, http.StatusCreated, APIResponse{Success: true, Message: "Title added to watchlist"})

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

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Rating recorded successfully"})
}

// 6. POST /api/sessions/start: Starts active stream (Enforces BR-02 Concurrency Limit)
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

	// 1. Verify stream cap from subscription_tier (BR-02)
	var maxStreams int
	err := db.QueryRow(`
		SELECT st.max_concurrent_streams
		FROM account a
		JOIN subscription_tier st ON a.current_tier_id = st.tier_id
		WHERE a.account_id = ?
	`, req.AccountID).Scan(&maxStreams)
	if err != nil {
		writeError(w, http.StatusNotFound, "Account or tier not found")
		return
	}

	// 2. Count active sessions for this account
	var activeCount int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM active_session 
		WHERE account_id = ? AND session_status = 'ACTIVE'
	`, req.AccountID).Scan(&activeCount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to audit active sessions")
		return
	}

	if activeCount >= maxStreams {
		writeError(w, http.StatusConflict, fmt.Sprintf(
			"Stream limit exceeded: Current tier allows %d concurrent streams (BR-02)", maxStreams,
		))
		return
	}

	// 3. Generate session ID and insert into active_session
	var sessionID string
	if err := db.QueryRow("SELECT UUID()").Scan(&sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, "UUID generation error")
		return
	}

	clientIP := r.RemoteAddr
	if colonIdx := strings.LastIndex(clientIP, ":"); colonIdx != -1 {
		clientIP = clientIP[:colonIdx]
	}

	// Note: Schema column name is prof_id
	insertQuery := `
		INSERT INTO active_session (session_id, account_id, prof_id, ip_address, device_fingerprint, session_status)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE');
	`
	_, err = db.Exec(insertQuery, sessionID, req.AccountID, req.ProfileID, clientIP, req.DeviceFingerprint)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize session: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "Streaming session active",
		Data:    map[string]string{"session_id": sessionID},
	})
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

	// Determine media duration to calculate completion flag (>= 90%)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CardRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// 1. Sanitize & extract safe display data
	cleanNumber := strings.ReplaceAll(req.CardNumber, " ", "")
	if len(cleanNumber) < 4 {
		http.Error(w, "Invalid card length", http.StatusBadRequest)
		return
	}
	lastFour := cleanNumber[len(cleanNumber)-4:]

	// 2. Identify brand (simple mock check)
	cardBrand := "UNKNOWN"
	if strings.HasPrefix(cleanNumber, "4") {
		cardBrand = "VISA"
	} else if strings.HasPrefix(cleanNumber, "5") {
		cardBrand = "MASTERCARD"
	}

	// 3. Mint a mock token to represent the gateway vault reference
	tokenBytes := make([]byte, 16)
	_, _ = rand.Read(tokenBytes)
	mockToken := fmt.Sprintf("tok_mock_%s", hex.EncodeToString(tokenBytes))

	// 4. Store ONLY safe fields in MySQL (Raw card number & CVV are discarded here)
	query := `
		INSERT INTO credit_card 
		(account_id, cardholder_name, card_brand, last_four, exp_month, exp_year, payment_token) 
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`
	_, err := db.Exec(query, req.AccountID, req.CardholderName, cardBrand, lastFour, req.ExpMonth, req.ExpYear, mockToken)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":   true,
		"message":   "Card tokenized successfully",
		"brand":     cardBrand,
		"last_four": lastFour,
		"token":     mockToken,
	})
}

// ============================================================================
// Server Initialization
// ============================================================================

func main() {
	// Configure MySQL credentials (replace with your MySQL user/pass)
	// dsn := "root:P@nzerkampf4@tcp(127.0.0.1:3306)/mininetflix?parseTime=true"
	dsn := "root:your_mysql_password@tcp(127.0.0.1:3306)/mininetflix?parseTime=true"

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("DSN parsing failed: %v", err)
	}
	defer db.Close()

	// Connection Pool Settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("Database unreachable: %v", err)
	}
	log.Println("Connected to MySQL 'mininetflix' successfully.")

	// Register HTTP Routes (Universal multiplexer compatibility)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/catalog", catalogHandler)
	mux.HandleFunc("/api/movies", createMovieHandler)
	mux.HandleFunc("/api/profiles", profilesHandler)
	mux.HandleFunc("/api/watchlist", watchlistHandler)
	mux.HandleFunc("/api/ratings", ratingsHandler)
	mux.HandleFunc("/api/sessions/start", startSessionHandler)
	mux.HandleFunc("/api/history/progress", historyProgressHandler)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("Mini-Netflix REST API running on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server stopped: %v", err)
	}
}
