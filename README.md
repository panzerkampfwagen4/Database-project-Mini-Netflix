# Mini-Netflix: On-Demand Streaming Database Project

Mini-Netflix is a Databases course project (class INT1313_CLC): a MySQL relational database, a Go HTTP API, and a browser interface served by the API. It demonstrates catalog browsing, subscription tiers, accounts and profiles, kids profiles, watchlists, ratings, simulated playback sessions, and billing. It does **not** stream real video or process real payments.

## Project and team

| Field | Details |
| --- | --- |
| Project | **#2: On-Demand Streaming Platform (Mini-Netflix)** |
| Course | **Databases** |
| Class code | **INT1313_CLC** |
| Institution | Posts and Telecommunications Institute of Technology (PTIT), Faculty of Information Technology 2, Ho Chi Minh City |
| Team | No Man's Land (NML) |

| Student (full name) | Student ID | Student email |
| --- | --- | --- |
| Nhan Duy Tran | N25DECE106 | n25dece106@student.ptithcm.edu.vn |
| Lam Vo | N25DECE055 | n25dece055@student.ptithcm.edu.vn |
| Hung Minh To | N25DECE092 | n25dece092@student.ptithcm.edu.vn |

## Repository contents

| Path | Purpose |
| --- | --- |
| [Go API source](./mininetflix-api/main.go) | Complete Go server and REST API; embeds and serves the browser UI. |
| [Web UI source](./mininetflix-api/static/index.html) | Single-page HTML, CSS, and JavaScript client. |
| [Go module](./mininetflix-api/go.mod), [checksums](./mininetflix-api/go.sum) | Dependency specification. |
| [Schema](./database/01_schema.sql) | MySQL tables, keys, and constraints. |
| [Views](./database/02_views.sql) | Tier-filtered and kids catalog views. |
| [RBAC example](./database/03_rbac.sql) | Database roles and grants; see limitations below. |
| [Seed dataset](./database/04_seed.sql) | Committed catalog and demo data needed to run the app. |
| [SQL examples](./database/05_tests.sql) | Incomplete, intentionally failing constraint examples; not a setup migration. |
| [Detailed report PDF](./Paper/Database%20Report.pdf), [editable DOCX](./Paper/Database%20Report.docx) | Project scope, design, normalization, data dictionary, implementation, and verification. |
| [Additional report PDF](./Paper/On-Demand%20Streaming%20Platform%20-%20Mini-Netflix.pdf) | Project report copy. |
| [Legacy TiDB sync script](./util_scripts/sync_seed_tidb.py) | Historical importer; not part of the installation procedure. |

The committed dataset includes titles, movies, series, seasons, episodes, media units, tiers, accounts, profiles, watchlists, ratings, history, and sample active sessions. Media URLs and payment tokens are demo data.

## Requirements

- Go **1.22+** (as specified in the [module](./mininetflix-api/go.mod)).
- MySQL **8.0** server and MySQL command-line client, or a compatible TiDB instance. Other database engines are not covered by these instructions.
- A free local port (default **10000**) and credentials allowed to create/import a database.

## Local installation

Run these commands from the **repository root**. Use a **new, disposable database**: the [seed](./database/04_seed.sql) truncates existing project tables before inserting demo records.

1. Clone and enter the repository:

   ```bash
   git clone <repository-URL>
   cd Database-project-Mini-Netflix
   ```

2. Create the database:

   ```bash
   mysql -u root -p -e "CREATE DATABASE mini_netflix CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
   ```

3. Start MySQL **from the repository root**. Type the following `SOURCE` statements at the `mysql>` prompt in this order:

   ```text
   mysql -u root -p mini_netflix
   mysql> SOURCE database/01_schema.sql;
   mysql> SOURCE database/02_views.sql;
   mysql> SOURCE database/04_seed.sql;
   mysql> SELECT COUNT(*) AS titles FROM catalog_title;
   mysql> SELECT account_id, email, current_tier_id, billing_status FROM account ORDER BY account_id;
   mysql> EXIT;
   ```

   Confirm the title count is greater than zero and demo accounts appear. `SOURCE` paths are relative to the directory from which you started `mysql`; use `/` in paths on Windows too. Do **not** run [the RBAC file](./database/03_rbac.sql) or [SQL examples](./database/05_tests.sql) during installation.

4. Set your own database connection and run the server. **PowerShell:**

   ```powershell
   $env:DB_USER = "root"
   $env:DB_PASSWORD = "<your-local-MySQL-password>"
   $env:DB_HOST = "127.0.0.1"
   $env:DB_PORT = "3306"
   $env:DB_NAME = "mini_netflix"
   $env:PORT = "10000"
   cd mininetflix-api
   go run .
   ```

   **Bash:**

   ```bash
   export DB_USER=root
   export DB_PASSWORD='<your-local-MySQL-password>'
   export DB_HOST=127.0.0.1 DB_PORT=3306 DB_NAME=mini_netflix PORT=10000
   cd mininetflix-api
   go run .
   ```

   Replace the password placeholder with your local MySQL password. Do not commit credentials. The server also accepts a complete `DB_DSN` or `DATABASE_DSN` in [go-sql-driver/mysql format](https://github.com/go-sql-driver/mysql#dsn-data-source-name); these take precedence over individual `DB_*` variables. For remote databases, add the provider's required TLS settings. `PORT` defaults to **10000**.

5. Open `http://localhost:10000/`. Check `http://localhost:10000/health`: success returns `"success": true` and `"message": "Connected"`. The database label in that response is fixed text, not a version probe.

The process can start even if the DB is unreachable; the UI and API then fail until the connection is fixed. Restart the server after changing its environment variables.

## Operation and evaluation

1. Sign in with a seeded demo account such as `alice@ptithcm.edu.vn`; the UI's default demo password is `password`. Choose a profile, then browse/search/filter the catalog. The seed includes adult and kids profiles across Free, Basic, Standard, and Premium accounts. Seeded password hashes are placeholders: the current demo login accepts any nonempty password for these accounts.
2. Add a title to **My List**, switch profiles, and compare their lists.
3. Select **Play Stream** on an eligible title. It creates a **simulated** database session. Use the active-stream controls to stop it before testing a plan's concurrent-stream cap.
4. Open **Account & Billing** to inspect or change demo account state. These actions write to the database; reimporting the seed resets demo records.
5. Use **Rules Verification** and **Dataset Provenance** in the UI, and consult the [detailed report](./Paper/Database%20Report.pdf) for design decisions and project verification.

Read-only API smoke checks from a second terminal:

```bash
curl http://localhost:10000/health
curl http://localhost:10000/api/profiles
curl "http://localhost:10000/api/catalog?profile_id=1&limit=5"
```

`profile_id=1` is present in the committed seed; newly created accounts may have other IDs. The API returns JSON. Routes registered in [`main.go`](./mininetflix-api/main.go) include `/api/catalog`, `/api/profiles`, `/api/watchlist`, `/api/ratings`, `/api/sessions/*`, `/api/accounts/*`, `/api/cards`, and `/api/auth/*`. Ratings use a **1–10** scale, as enforced by the schema and API.

## Deployment

1. Provision a **new** MySQL 8.0 or compatible TiDB database. Import [schema](./database/01_schema.sql), [views](./database/02_views.sql), then [seed](./database/04_seed.sql) into the same database. Keep its credentials private.
2. Deploy [`mininetflix-api/`](./mininetflix-api/) to a Go 1.22+ host, with that folder as its working/root directory. Build with `go build -o server .` and start `./server` (or `server.exe` on Windows).
3. Set `DB_DSN`/`DATABASE_DSN` **or** `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT`, `DB_NAME`. Let the host provide `PORT` if it does so automatically. Configure TLS and network access as required by the database provider.
4. Check `https://<your-host>/health`, then open the root page. No separate frontend server is required because the Go binary embeds the HTML.

There is no Dockerfile or infrastructure manifest in the repository. This is the manual deployment path supported by the committed files. Local evaluation does not depend on any hosted demo.

## Data, verification, and limitations

- The [seed SQL](./database/04_seed.sql) is the dataset necessary to reproduce this project. The catalog is described as derived from the [Netflix Movies and TV Shows dataset](https://www.kaggle.com/datasets/shivamb/netflix-shows). The original raw download and seed-generation program are **not** committed; the exact project records are in the seed SQL.
- [`05_tests.sql`](./database/05_tests.sql) contains examples of a rating check-constraint violation and duplicate watchlist rejection. It is incomplete and intentionally raises errors. Run individual statements only in a disposable copy if you want to inspect them.
- [`03_rbac.sql`](./database/03_rbac.sql) illustrates database roles, but is not required by the Go application. It references some users before creating them and includes example passwords; review and adapt before using it for a separate RBAC demonstration.
- [`sync_seed_tidb.py`](./util_scripts/sync_seed_tidb.py) is legacy: it expects a missing `seed.sql`, includes old connection settings, and truncates tables. Use the SQL files and MySQL client above.
- Authentication, authorization, card tokens, and billing are **course-project demonstrations**, not production security/payment integrations. The seed uses placeholder password hashes and source files contain demo credentials. Do not use real customer/card data or a public production DB. Rotate cloud credentials committed in source before sharing or deploying.

If the app reports database errors, confirm that `mini_netflix` exists, the three setup files completed, the DB environment variables target that database, and `/health` succeeds. If Go cannot resolve modules, run `go mod download` inside [`mininetflix-api/`](./mininetflix-api/) and retry.
