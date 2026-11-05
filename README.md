# On-Demand Streaming Platform - Mini-Netflix Database System

An enterprise-grade relational database architecture and evaluation platform modeling an on-demand video streaming service. The system implements multi-tenant profile isolation, subscription-based content gating, real-time concurrency tracking, and transactional billing workflows compliant with third normal form (3NF).

> This repository accompanies the report:  
> **On-Demand Streaming Platform - Mini-Netflix**  
> Nhan D. Tran (n25dece106@student.ptithcm.edu.vn), Lam Vo (n25dece055@student.ptithcm.edu.vn), Hung M. To (n25dece092@student.ptithcm.edu.vn)  
> **Faculty of Information Technology - Posts and Telecommunications Institute of Technology (PTIT HCMC)**

---

## 1. Domain Concepts and Business Rules

### 1.1 Core Domain Concepts

The Mini-Netflix architecture models the real-world operational domain of large-scale video streaming providers. The system separates account ownership from consumption identity, and decouples high-level catalog metadata from low-level streamable media files.

1. **Subscription Tiers:** Hierarchical service agreements defining access boundaries. Each tier specifies a monthly rate, video resolution ceiling, and maximum concurrent streams:
   * **Free (Tier 0):** Up to 480p resolution, strictly 1 concurrent stream, access restricted to baseline content.
   * **Basic (Tier 1):** Up to 720p resolution, strictly 1 concurrent stream.
   * **Standard (Tier 2):** Up to 1080p resolution, up to 2 concurrent streams.
   * **Premium (Tier 3):** Up to 4K UHD resolution, up to 4 concurrent streams.

2. **Accounts and Profiles:**
   * **Account:** Represents the paying entity and legal subscriber. Holds login credentials, billing status (`ACTIVE`, `SUSPENDED`, `CANCELLED`), subscription tier, and tokenized payment records.
   * **Profile:** Represents an individual viewing identity belonging to an Account (up to 5 profiles per account). Maintains isolated watch history, custom watchlists, and title ratings. Can be flagged as a **Kids Profile**.

3. **Content Hierarchy and Media Units:**
   * **Catalog Title:** Abstract container holding common metadata across all titles (title name, maturity rating, minimum tier requirement, synopsis).
   * **Movies (`movie_title`):** Standalone content linked 1:1 with a media stream unit.
   * **TV Series (`tv_series`):** Multi-tiered serialized content structured into `season` entities (season number, release date) and `episode` entities (episode number, duration), each referencing its dedicated `streamable_media_unit`.
   * **Streamable Media Unit (`streamable_media_unit`):** Concrete technical entity storing storage asset URLs, technical resolution, runtime, and audio codec information.

---

### 1.2 Formal Business Rules

The relational schema and API layer strictly enforce the following business rules:

* **BR-01 (Subscription Tier Gating):** A user cannot stream content if their account's subscription tier is lower than the title's `minimum_tier_id`. Attempts to access higher-tier content return an authorization failure indicating the required upgrade.
* **BR-02 (Concurrent Stream Cap Enforcement):** Active streams across all profiles in an account cannot exceed the account's tier limit (`max_concurrent_streams`). Any playback initiation request that breaches this limit is rejected unless an existing stream session is explicitly stopped.
* **BR-03 (Parental Content Restriction):** Profiles designated as `is_kids_profile = TRUE` can only access content with maturity ratings `G`, `PG`, or `TV-Y`. Titles rated `PG-13`, `R`, `TV-MA`, or `NC-17` are filtered from queries and blocked from playback.
* **BR-04 (Watchlist and Rating Integrity):** A profile cannot add duplicate titles to its watchlist (enforced by a composite primary key `(profile_id, title_id)`). A title rating can only be recorded between 1 and 5 stars.
* **BR-05 (Billing Lifecycle and Profile Retention):** An account marked `SUSPENDED` (due to missed renewal) cannot initiate playback until an invoice is settled. When an account is `CANCELLED`, all active sessions are immediately terminated, while profile configurations and viewing data are preserved for a 10-month retention window.
* **BR-06 (Tokenized Payment Isolation):** Credit card numbers are never stored in plain text. The system records card brand, last four digits, expiration dates, and a cryptographic payment gateway token (`payment_token`), ensuring PCI-DSS compliance in transactional auditing.

---

## 2. Dataset Provenance and Synthesis

### 2.1 Dataset Source and Preprocessing

The primary catalog data is derived from the canonical Netflix Movies and TV Shows dataset hosted on Hugging Face and Kaggle:

* **Primary Dataset Source:** [Hugging Face: hugginglearners/netflix-shows](https://huggingface.co/datasets/hugginglearners/netflix-shows)
* **Secondary Mirror:** [Kaggle: Netflix Movies and TV Shows](https://www.kaggle.com/datasets/shivamb/netflix-shows)

The raw source dataset contains 8,807 unstructured entries with comma-separated cast lists, unnormalized durations (mixing minutes and season counts), and missing technical identifiers. 

The ingestion pipeline (`scripts/generate_seed.py` and `scripts/sync_seed_tidb.py`) processed and normalized this raw data into a strictly typed, Third Normal Form (3NF) relational model:
1. Parsing UUIDs and separating movies from multi-season television shows.
2. Synthesizing seasons and episode counts for television series with sequential episode identifiers.
3. Generating corresponding technical `streamable_media_unit` records with accurate duration seconds and resolutions based on maturity tiers.
4. Synthesizing representative user accounts across multiple universities, realistic profile distributions, watchlists, ratings, and active stream sessions.

---

### 2.2 Entity Count and Data Volume Summary

The production database is hosted on a cloud-native TiDB cluster (MySQL 8.0 wire-compatible) in AWS Singapore (`ap-southeast-1`). The table below outlines the exact record distribution:

| Database Table | Entity Description | Record Count |
| :--- | :--- | :---: |
| `catalog_title` | Unified catalog headers (Movies and TV Series) | 8,786 |
| `movie_title` | Standalone feature films | 6,121 |
| `tv_series` | Serialized television series containers | 2,665 |
| `season` | Television show season records | 4,709 |
| `episode` | Individual episodes within seasons | 5,330 |
| `streamable_media_unit` | Concrete streamable media assets (URLs, runtimes) | 11,451 |
| `subscription_tier` | Plan tiers (Free, Basic, Standard, Premium) | 4 |
| `account` | Customer accounts with credentials and billing state | 5 |
| `profile` | Individual viewing personas (Adult vs. Kids) | 8 |
| `watchlist` | Profile bookmark records | 8 |
| `title_rating` | Profile rating reviews (1 to 5 scale) | 6 |
| `watch_history` | Playback resumption and timestamp tracking | 5 |
| `active_session` | Real-time concurrent streaming leases | Dynamic |
| `credit_card` | Tokenized PCI-DSS payment methods | Dynamic |
| `payment_transaction` | Invoiced billing ledger entries | Dynamic |
| **Total System Entities** | **Normalized relational rows across all tables** | **36,000+** |

---

## 3. Interactive Web Application Overview

The project provides a fully featured, single-page web application inspired by the authentic Netflix desktop interface. The application serves as both an interactive client for catalog browsing and an administrative evaluation portal for verifying database constraints.

* **Live Deployment:** [https://database-project-mini-netflix.onrender.com](https://database-project-mini-netflix.onrender.com)
* **Health and Telemetry Endpoint:** `GET https://database-project-mini-netflix.onrender.com/health`

```
+---------------------------------------------------------------------------------------------------+
|  MINI-NETFLIX   Home  My List  Account & Billing  Rules Verification  Provenance  API Docs  [Profile] |
+---------------------------------------------------------------------------------------------------+
|                                                                                                   |
|  FEATURED TITLE BILLBOARD                                                                         |
|  '76 (Movie | 118 min | 4K UHD | PG-13)                                                           |
|  [ Play Stream ]   [ + My List ]                                                                  |
|                                                                                                   |
+---------------------------------------------------------------------------------------------------+
|  EXPLORE CATALOG                                         [Filter: All Titles | Movies | TV Series] |
|  [ Card 1: 16:9 ]    [ Card 2: 16:9 ]    [ Card 3: 16:9 ]    [ Card 4: 16:9 ]    [ Card 5: 16:9 ]  |
+---------------------------------------------------------------------------------------------------+
```

### 3.1 Interface Components

1. **Top Navigation Bar:**
   * **Navigation Links:** Instant filtering across `Home`, `My List` (bookmarked items for the selected profile), `Account & Billing`, `Rules Verification`, `Dataset Provenance`, and `API Reference`.
   * **Real-Time Stream Meter:** Displays the live concurrent stream utilization for the active account (e.g., `Streams: 1 / 2`).
   * **Profile Context Selector:** Dropdown menu allowing instant switching between pre-seeded accounts and profiles to demonstrate rule enforcement in real time.

2. **Billboard (Hero Section):**
   * Automatically showcases a high-tier featured title with synopsis, runtime, video quality badge, maturity rating, and direct playback controls.

3. **Catalog Grid with Real-Time Filtering:**
   * Displays 16:9 cinematic cards with tier indicators, runtime/resolution tags, and quick-action buttons (`Play Stream`, `Toggle Watchlist`).
   * Supports live text search and content-type toggle buttons (`All Titles`, `Movies`, `TV Series`).

4. **Account & Billing Modal:**
   * Displays account tier, renewal date, and billing status (`ACTIVE`, `SUSPENDED`, `CANCELLED`).
   * Provides controls for tier upgrading/downgrading, simulated subscription payment, account cancellation, and credit card registration with mock tokenization.

5. **Rules Verification Modal:**
   * An automated visual test runner presenting the status of all five business rules (BR-01 through BR-05) based on the currently selected profile.

---

## 4. Quickstart and Evaluation Guide

This guide is designed for evaluators and instructors to verify all relational constraints and business rules via both the interactive web portal and direct REST API testing in Postman.

### 4.1 Evaluation Scenarios (Interactive Web Portal)

Evaluators can follow these step-by-step scenarios on the live portal:

#### Test Scenario 1: Subscription Tier Gating (BR-01)
1. In the top-right profile selector, switch to **David (Free Tier)** (Account #4).
2. Browse the catalog and locate a title marked **Premium Tier Required** (or click Play on the featured hero title).
3. **Observed Result:** The system blocks playback and displays an access error stating: *"Access Denied: Title requires Premium tier. Account is currently on Free tier."*
4. Open **Account & Billing**, change the tier dropdown to **Premium ($19.99/mo)**, and confirm.
5. Click **Play Stream** on the same title.
6. **Observed Result:** The stream initializes successfully.

#### Test Scenario 2: Concurrent Stream Cap Enforcement (BR-02)
1. Switch to **Charlie (Basic Tier)** (Account #3). Basic tier allows strictly **1 stream**.
2. Click **Play Stream** on any eligible movie. The stream meter updates to `Streams: 1 / 1`.
3. Attempt to start a second stream on another title.
4. **Observed Result:** The API rejects the request with HTTP 403: *"Concurrent streaming limit reached (1/1 streams active for Basic plan). Stop an existing stream to start a new one."*
5. Click the Stream Meter pill in the navbar to open the Active Streams manager, click **Terminate Session**, and retry. The second stream now succeeds.

#### Test Scenario 3: Parental Controls for Kids Profiles (BR-03)
1. Select **Alice (Main)** (Account #1, Adult profile). Observe titles rated `R`, `TV-MA`, and `PG-13` present in the catalog.
2. In the profile selector, switch to **Alice (Kids)** (`is_kids_profile = TRUE`).
3. **Observed Result:** The catalog immediately re-renders, displaying exclusively family-safe titles (`G`, `PG`, `TV-Y`). All mature titles are completely hidden and unqueryable.

#### Test Scenario 4: Profile Isolation & Watchlist Management (BR-04)
1. Under **Alice (Main)**, click the **+ My List** button on two distinct movies.
2. Click the **My List** navigation tab to verify the items appear.
3. Switch profile to **Bob (Main)** (Account #2) and click **My List**.
4. **Observed Result:** Bob's list is completely empty or contains only his own bookmarks. Data isolation is maintained across profiles.

#### Test Scenario 5: Account Suspension and Payment Workflow (BR-05 / BR-06)
1. Switch profile to **Emma (Suspended)** (Account #5). Notice the `SUSPENDED` account badge.
2. Attempt to start any playback stream.
3. **Observed Result:** Playback is blocked: *"Account subscription is SUSPENDED due to past due invoice."*
4. Open **Account & Billing**, click **Pay Due Invoice & Reactivate**.
5. The system records a mock transaction, updates `billing_status` to `ACTIVE`, and recalculates `next_billing_date` 30 days ahead. Playback is restored immediately.

---

### 4.2 Direct Postman and REST API Testing Guide

Instructors and evaluators who prefer testing directly against the backend REST API using **Postman**, **cURL**, or any HTTP client can execute the test suite below.

#### Base URL Configuration
* **Live Cloud Deployment:** `https://database-project-mini-netflix.onrender.com`
* **Local Development Server:** `http://localhost:8080`

In Postman, define an environment variable `baseUrl = https://database-project-mini-netflix.onrender.com` and set the request header `Content-Type: application/json` for all POST requests.

---

#### Postman Test Case 1: System Health and Database Engine Verification
Verifies connectivity to the cloud TiDB / MySQL cluster and confirms connection pooling state.

* **Method:** `GET`
* **Endpoint:** `{{baseUrl}}/health`
* **cURL Command:**
  ```bash
  curl -X GET "https://database-project-mini-netflix.onrender.com/health"
  ```
* **Expected Response (HTTP 200 OK):**
  ```json
  {
    "success": true,
    "message": "Connected",
    "data": {
      "database": "MySQL 8.0 / TiDB"
    }
  }
  ```

---

#### Postman Test Case 2: Content Catalog Retrieval & Parental Restriction (BR-03)
Demonstrates relational filtering based on the profile context. Querying with an adult profile returns all mature content, while querying with a kids profile strictly excludes R and TV-MA titles.

* **Method:** `GET`
* **Endpoint (Adult Profile):** `{{baseUrl}}/api/catalog?profile_id=1&limit=5`
* **Endpoint (Kids Profile):** `{{baseUrl}}/api/catalog?profile_id=2&limit=5`
* **cURL Command (Kids Profile Verification):**
  ```bash
  curl -X GET "https://database-project-mini-netflix.onrender.com/api/catalog?profile_id=2&limit=5"
  ```
* **Expected Response (HTTP 200 OK):**
  ```json
  {
    "success": true,
    "count": 5,
    "data": [
      {
        "title_id": "0007b302-126b-7e30-890a-30258bfed41e",
        "title_name": "Elf Pets: Santa's Reindeer Rescue",
        "content_type": "Movie",
        "maturity_rating": "G",
        "minimum_tier_id": 0,
        "video_resolution": "480p"
      }
    ]
  }
  ```
  *Notice that `maturity_rating` never contains R, TV-MA, or PG-13 when `profile_id=2`.*

---

#### Postman Test Case 3: Start Stream & Tier / Concurrency Gating (BR-01, BR-02, BR-05)
Initiates a streaming lease. If the account's tier is below the title requirement or active streams reach the plan cap, the transaction is aborted.

* **Method:** `POST`
* **Endpoint:** `{{baseUrl}}/api/sessions/start`
* **Headers:** `Content-Type: application/json`
* **cURL Command (Testing Tier Gating with Free Account #4):**
  ```bash
  curl -X POST "https://database-project-mini-netflix.onrender.com/api/sessions/start" \
    -H "Content-Type: application/json" \
    -d '{
      "account_id": 4,
      "profile_id": 7,
      "title_id": "00281697-f03c-54ad-903f-60a01ac8038f",
      "device_fingerprint": "postman-evaluator"
    }'
  ```
* **Expected Response when Tier Insufficient (HTTP 403 Forbidden):**
  ```json
  {
    "success": false,
    "message": "Title requires Premium tier (Account is currently Free)"
  }
  ```
* **Expected Response on Success (HTTP 200 OK):**
  ```json
  {
    "success": true,
    "message": "Playback session started",
    "data": {
      "session_id": "sess_a8b9c0d1e2f3",
      "max_streams": 4,
      "active_streams": 1
    }
  }
  ```

---

#### Postman Test Case 4: Terminate Stream Session (Concurrency Release)
Terminates an active playback session lease, freeing capacity for concurrent playback.

* **Method:** `POST`
* **Endpoint:** `{{baseUrl}}/api/sessions/stop`
* **Headers:** `Content-Type: application/json`
* **cURL Command:**
  ```bash
  curl -X POST "https://database-project-mini-netflix.onrender.com/api/sessions/stop" \
    -H "Content-Type: application/json" \
    -d '{
      "session_id": "sess_a8b9c0d1e2f3"
    }'
  ```
* **Expected Response (HTTP 200 OK):**
  ```json
  {
    "success": true,
    "message": "Session terminated"
  }
  ```

---

#### Postman Test Case 5: Upgrade Subscription Tier (BR-01 Resolution)
Modifies the account subscription plan directly in the database.

* **Method:** `POST`
* **Endpoint:** `{{baseUrl}}/api/accounts/tier`
* **Headers:** `Content-Type: application/json`
* **Payload:**
  ```json
  {
    "account_id": 4,
    "tier_id": 3
  }
  ```
* **cURL Command:**
  ```bash
  curl -X POST "https://database-project-mini-netflix.onrender.com/api/accounts/tier" \
    -H "Content-Type: application/json" \
    -d '{"account_id": 4, "tier_id": 3}'
  ```
* **Expected Response (HTTP 200 OK):**
  ```json
  {
    "success": true,
    "message": "Account #4 subscription tier updated"
  }
  ```

---

#### Postman Test Case 6: Invoiced Payment & Account Reactivation (BR-05)
Processes a simulated billing settlement for suspended accounts (such as Account #5), updating `billing_status` to `ACTIVE` and advancing `next_billing_date` by 30 days.

* **Method:** `POST`
* **Endpoint:** `{{baseUrl}}/api/accounts/pay`
* **Headers:** `Content-Type: application/json`
* **Payload:**
  ```json
  {
    "account_id": 5
  }
  ```
* **cURL Command:**
  ```bash
  curl -X POST "https://database-project-mini-netflix.onrender.com/api/accounts/pay" \
    -H "Content-Type: application/json" \
    -d '{"account_id": 5}'
  ```
* **Expected Response (HTTP 200 OK):**
  ```json
  {
    "success": true,
    "message": "Payment processed for Account #5. Billing status is now ACTIVE."
  }
  ```

---

#### Postman Test Case 7: Tokenized Card Registration & Ledger Auditing (BR-06)
Registers a credit card without persisting raw primary account numbers, generating a synthetic gateway payment token.

* **Method:** `POST`
* **Endpoint:** `{{baseUrl}}/api/cards`
* **Headers:** `Content-Type: application/json`
* **Payload:**
  ```json
  {
    "account_id": 1,
    "cardholder_name": "Alice Smith",
    "card_number": "4242424242424242",
    "exp_month": 12,
    "exp_year": 2028,
    "cvv": "123"
  }
  ```
* **cURL Command:**
  ```bash
  curl -X POST "https://database-project-mini-netflix.onrender.com/api/cards" \
    -H "Content-Type: application/json" \
    -d '{
      "account_id": 1,
      "cardholder_name": "Alice Smith",
      "card_number": "4242424242424242",
      "exp_month": 12,
      "exp_year": 2028,
      "cvv": "123"
    }'
  ```
* **Expected Response (HTTP 201 Created):**
  ```json
  {
    "success": true,
    "message": "Card registered and initial transaction processed successfully",
    "data": {
      "card_id": 1,
      "brand": "VISA",
      "last_four": "4242",
      "payment_token": "tok_mock_8f91c7a2b3e4d5",
      "transaction_status": "SUCCESS"
    }
  }
  ```

---

#### Postman Test Case 8: Profile Watchlist Toggle (BR-04)
Toggles a catalog title in the profile's isolated watchlist. Adding an existing item automatically removes it, preserving uniqueness without primary key collisions.

* **Method:** `POST`
* **Endpoint:** `{{baseUrl}}/api/watchlist`
* **Headers:** `Content-Type: application/json`
* **Payload:**
  ```json
  {
    "profile_id": 1,
    "title_id": "0007b302-126b-7e30-890a-30258bfed41e"
  }
  ```
* **cURL Command:**
  ```bash
  curl -X POST "https://database-project-mini-netflix.onrender.com/api/watchlist" \
    -H "Content-Type: application/json" \
    -d '{"profile_id": 1, "title_id": "0007b302-126b-7e30-890a-30258bfed41e"}'
  ```
* **Expected Response (HTTP 200 OK):**
  ```json
  {
    "success": true,
    "message": "Added to watchlist"
  }
  ```

---

### 4.3 Local Development and Deployment

The project can be executed locally using the native Go runtime or Docker.

#### Prerequisites
* Go 1.22 or higher (or Docker)
* Access to a MySQL 8.0 or TiDB database cluster

#### Environment Variables
| Variable | Description | Default Fallback |
| :--- | :--- | :--- |
| `PORT` | Listening HTTP port | `8080` |
| `DATABASE_DSN` | Full MySQL connection DSN | Built-in TiDB Cloud Singapore gateway |
| `DB_DSN` | Alternative alias for connection DSN | Uses `DATABASE_DSN` |

#### Running the Server
```bash
# Clone the repository
git clone https://github.com/panzerkampfwagen4/Database-project-Mini-Netflix.git
cd Database-project-Mini-Netflix

# Run the API server directly (dependencies resolve automatically via Go modules)
go run mininetflix-api/main.go
```
The server will start on `http://localhost:8080` with the complete web GUI embedded.

---

## 5. Repository Architecture

```
Database-project-Mini-Netflix/
|-- schema.sql                 # Complete DDL definitions for 15 relational tables with 3NF constraints
|-- seed.sql                   # Production seed dataset (39,928 SQL lines, 8,786 catalog titles)
|-- README.md                  # Comprehensive technical documentation and evaluation manual
|-- mininetflix-api/           # RESTful API and Web Platform backend
|   |-- go.mod                 # Go module definition and dependencies (mysql driver)
|   |-- go.sum                 # Cryptographic checksums of Go dependencies
|   |-- main.go                # Server implementation (handlers, transaction logic, connection pool)
|   +-- static/
|       +-- index.html         # Authentic single-page Netflix web client with embedded styles & logic
+-- scripts/                   # Data pipelines and database synchronization tools
    |-- generate_seed.py       # Raw dataset ingestion and SQL seed generator
    +-- sync_seed_tidb.py      # Automated bulk migration and verification script for TiDB Cloud
```

### Module Descriptions
* **`schema.sql`:** Contains table definitions, primary keys, foreign key cascades, unique constraints, and check constraints across all 15 relational entities.
* **`seed.sql`:** Structured data load script containing 8,786 titles, 11,451 media streams, test accounts across multiple subscription tiers, profiles, and initial stream sessions.
* **`mininetflix-api/main.go`:** Self-contained backend server providing JSON endpoints (`/api/catalog`, `/api/sessions`, `/api/accounts`, `/api/watchlist`, `/api/ratings`, `/api/cards`) and serving the single-page application via `go:embed`.
* **`mininetflix-api/static/index.html`:** Clean single-page application implementing real-time catalog search, profile state switching, stream concurrency inspectors, and billing dialogues without external framework overhead.
* **`scripts/`:** Python automation scripts responsible for dataset transformation, schema normalization, and automated TiDB Cloud ingestion.
