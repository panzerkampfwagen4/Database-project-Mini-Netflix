# Database Project: On-Demand Streaming Platform - Mini-Netflix

The proposed system is an On-Demand Streaming Platform (Mini-Netflix). The platform is a centralized multimedia delivery service designed to manage, categorize, and stream digital video content to end-users. 

To accurately manage this catalog, the system categorizes "Content" into two distinct hierarchical types:  

* **Movies:** Standalone video files characterized by a single duration and video resolution.   
* **TV Series Episodes:** Serialized media units that belong to a broader TV Series container, requiring complex tracking of seasons and episode numbers. 

> This repository accompanies the report:  
> **On-Demand Streaming Platform - Mini-Netflix**  
> Nhan D. Tran (n25dece106@student.ptithcm.edu.vn), Lam Vo (n25dece055@student.ptithcm.edu.vn), Hung M. To (n25dece092@student.ptithcm.edu.vn)  
> **Faculty of Information Technology - Posts and Telecommunications Institute of Technology (PTIT HCMC)**

---

## 🌟 Live Demo & Evaluation Portal

Giảng viên và người đánh giá có thể truy cập trực tiếp hệ thống kiểm thử trực tuyến (Live GUI & REST API) mà không cần cài đặt bất kỳ môi trường nào trên máy:

* 🌐 **Live Web Portal:** [https://database-project-mini-netflix.onrender.com](https://database-project-mini-netflix.onrender.com) *(hoặc domain deploy của nhóm)*
* 🩺 **Health & Keep-Alive Endpoint:** `GET /health` (Được giám sát tự động 24/7 qua UptimeRobot đảm bảo server không sleep/ngủ đông)
* 🧪 **Interactive Testing Suite:** Tích hợp sẵn giao diện chuyển đổi Profile linh hoạt và các bộ test 1-click cho toàn bộ Business Rules (BR-01 đến BR-05).

---

## System Overview

This project models the database backend of an on-demand streaming platform inspired by Netflix. It manages user accounts, profiles, subscription plans, digital content, and viewing activities while enforcing access control, profile isolation, and data integrity through a structured relational database.

---

## Project Objective

The main objective of this project is to design and implement a structured, normalized relational database for an on-demand streaming platform. The database aims to maintain data integrity, support subscription-based access control, manage user profiles and viewing activities, and enforce key business rules through relational constraints.

---

## Business Requirements

### 2.1 Business Context

The system is designed for an on-demand streaming platform where users can subscribe to a membership plan and access a centralized catalog of movies and TV series. A single account can manage billing and subscription information while supporting multiple individual profiles, each with its own viewing history, watchlist, and ratings. The platform also needs to manage different content types, genres, subscription-based access levels, concurrent streaming sessions, and parental restrictions. The database therefore serves as the core system for organizing these relationships and enforcing the business rules required to keep user data isolated, content access controlled, and day-to-day streaming operations consistent.

### 2.2 User Roles 

The system defines three main roles with clearly separated responsibilities:

* **Database Administrator (DBA):** Responsible for managing the database infrastructure, including access authorization, security, performance monitoring, and overall database maintenance. The DBA operates at the backend level and does not represent a normal streaming user.
* **Account Owner:** The primary customer account responsible for subscription and billing management. An Account Owner can manage the account's membership tier and create or remove profiles associated with the account.
* **Profile:** The end-user identity used for watching content. Each profile belongs to an Account and manages its own watch history, watchlist, ratings, language preferences, and other viewing settings. A profile may also be designated as a **Kids Profile**, subject to age-appropriate content restrictions.

### 2.3 Core Business Rules

The database enforces the following core business rules to maintain consistent access control, user isolation, and data integrity:

* **BR-01 — Subscription Access:** A profile can access content only when the content's required subscription tier is less than or equal to the account's current tier (`ct.minimum_tier_id <= a.current_tier_id`).
* **BR-02 — Concurrent Streaming:** The number of active streaming sessions for an account cannot exceed the maximum concurrent streams allowed by its subscription tier (`active_sessions < st.max_concurrent_streams`).
* **BR-03 — Profile Data Isolation:** Watch history, watchlists, and ratings are managed independently for each profile. Activities performed by one profile must not affect another profile under the same account (enforced by composite foreign keys `(account_id, profile_id)`).
* **BR-04 — Kids Content Restriction:** A Kids Profile must not be allowed to access content classified as R or NC-17 (`ct.maturity_rating NOT IN ('R', 'NC-17')`).
* **BR-05 — Profile Limit:** Each account must have at least 1 and no more than 5 profiles (`COUNT(profile) <= 5`).
* **BR-06 — Referential Integrity:** Deleting an account must automatically remove its dependent profiles and related activity records, preventing orphaned data through appropriate cascading rules (`ON DELETE CASCADE`).

---

## Database Design

### 3.1 Conceptual Model

The conceptual model represents the main entities and relationships required to support the streaming platform. It organizes accounts, profiles, subscription tiers, content, genres, viewing activities, and streaming sessions while defining their cardinalities and dependencies. The model uses specialization to distinguish Movie and Episode as subtypes of Content, and associative entities to resolve many-to-many relationships such as watchlists, watch history, and profile ratings.

### 3.2 Entity Overview

The database consists of the following main entities:

* **Subscription_Tier:** Defines available membership plans (Free, Basic, Standard, Premium), streaming limits, and video quality.
* **Account:** Represents the primary customer account responsible for subscription and billing.
* **Profile:** Represents an individual user profile belonging to an account, with separate viewing preferences and activity data.
* **Catalog_Title:** The superclass representing all streamable media catalog items.
* **Streamable_Media_Unit:** Technical streaming asset storing video resolution, runtime duration in seconds, and storage URL.
* **Movie_Title:** Specialization subclass linking a standalone `catalog_title` to a `streamable_media_unit`.
* **TV_Series:** Specialization subclass representing a serialized TV show containing seasons.
* **Season & Episode:** Hierarchical decomposition of TV Series into distinct seasons and streamable episode media units.
* **Watchlist:** Stores titles saved by profiles for later viewing.
* **Watch_History:** Records a profile's viewing activity, playback stop checkpoint, and 90% completion flag.
* **Title_Rating:** Stores ratings (scores 1-10) given by profiles to catalog titles.
* **Active_Session:** Tracks currently active streaming sessions associated with an account and profile.

---

## 📊 Dataset Provenance & Data Engineering (Hiệu chỉnh dữ liệu)

### 4.1 Nguồn dữ liệu (Dataset Citation)
Đồ án sử dụng bộ dữ liệu thực tế:
> **Netflix Movies and TV Shows Dataset**  
> HuggingFace Repository: [`hugginglearners/netflix-shows`](https://huggingface.co/datasets/hugginglearners/netflix-shows)  
> Dữ liệu gốc gồm 8,807 bản ghi (6,131 Movies và 2,676 TV Shows).

### 4.2 Báo cáo chuyển đổi & Hiệu chỉnh đồng bộ với `schema.sql`
Để đảm bảo **không làm thay đổi kiến trúc CSDL đã chuẩn hóa trong `schema.sql`** và tài liệu báo cáo của nhóm, nhóm đã áp dụng quy trình chuẩn hóa dữ liệu (ETL / Data Normalization) qua script [`scripts/generate_seed.py`](file:///d:/Repository/Database-project-Mini-Netflix/scripts/generate_seed.py) như sau:

| Thuộc tính Dataset gốc | Thực thể CSDL đích | Phương pháp hiệu chỉnh (Transformation Logic) |
| :--- | :--- | :--- |
| `type` (`"Movie"`, `"TV Show"`) | `catalog_title.content_type` | Map `"TV Show"` $\rightarrow$ `'Series'` để khớp 100% với `ENUM('Movie', 'Series')`. |
| `rating` (`TV-MA`, `TV-14`, `TV-PG`, `TV-Y`...) | `catalog_title.maturity_rating` | Chuẩn hóa hệ mã truyền hình sang mã tương đương trong `ENUM('G', 'PG', 'PG-13', 'R', 'NC-17')`:<br>&bull; `TV-MA`, `UR` $\rightarrow$ `'R'` (Dành cho người lớn)<br>&bull; `TV-14` $\rightarrow$ `'PG-13'` (Khán giả từ 14 tuổi)<br>&bull; `TV-PG`, `NR` $\rightarrow$ `'PG'` (Có phụ huynh hướng dẫn)<br>&bull; `TV-Y`, `TV-Y7`, `TV-G` $\rightarrow$ `'G'` (Mọi lứa tuổi / Trẻ em) |
| `duration` (`"90 min"`, `"2 Seasons"`) | `streamable_media_unit.duration_seconds` & `tv_series.total_seasons` | &bull; **Movie:** Trích xuất số phút và quy đổi ra giây (`seconds = min * 60`).<br>&bull; **Series:** Trích xuất số Season lưu vào `tv_series.total_seasons`, đồng thời tự động phân rã thành các bảng con `season` và `episode` với thời lượng tập tương ứng. |
| *(Dữ liệu nền tảng)* | `subscription_tier` | Khởi tạo 4 gói cước chuẩn: `Free` (Tier 0), `Basic` (Tier 1), `Standard` (Tier 2), `Premium` (Tier 3). |
| *(Dữ liệu phân quyền)* | `catalog_title.minimum_tier_id` | Phân bổ cấp gói yêu cầu từ 0 đến 3 trên danh mục để phục vụ kiểm thử **BR-01**. |
| *(Dữ liệu người dùng mẫu)* | `account`, `profile`, `watchlist`, `title_rating` | Khởi tạo 5 tài khoản sinh viên PTIT với đầy đủ Adult Profile, Kids Profile (kiểm thử **BR-04**), giới hạn Stream (kiểm thử **BR-02**) và hạn mức 5 profiles (kiểm thử **BR-05**). |

---

## Database Implementation

### 5.1 Technology Stack
* **Database Management System:** MySQL 8.0+ / TiDB Cloud Serverless (Wire-compatible MySQL).
* **Backend Runtime:** Go (Golang 1.22+) RESTful API microservice.
* **Frontend Portal:** Single-Page Web Application (Embedded HTML5/CSS3/Vanilla JS) styled with Netflix modern dark theme.
* **Keep-Alive Service:** UptimeRobot 24/7 Automated Health Ping Monitor.

### 5.2 Project Structure
```text
Database-project-Mini-Netflix/
├── mininetflix-api/
│   ├── static/
│   │   └── index.html         # Web GUI Evaluation Portal (nhúng trực tiếp vào Go binary)
│   ├── go.mod                 # Go module definition
│   ├── go.sum                 # Go dependencies checksums
│   └── main.go                # REST API handlers, CORS middleware, DSN config & DB pool
├── scripts/
│   └── generate_seed.py       # Script ETL chuyển đổi dataset HuggingFace sang seed.sql
├── schema.sql                 # Cấu trúc CSDL (13 bảng, DDL, Khóa ngoại, Ràng buộc CHECK/UNIQUE)
├── seed.sql                   # Bộ dữ liệu mẫu đầy đủ từ dataset Netflix
├── netflix_titles.csv         # File dataset gốc từ HuggingFace
├── Database Report.pdf        # Báo cáo đồ án CSDL hoàn chỉnh
└── README.md                  # Hướng dẫn chi tiết cho Giảng viên & Hội đồng
```

---

## Setup & Quickstart

### 6.1 Khởi chạy trực tiếp (Không cần cài đặt)
Giảng viên chỉ cần mở trình duyệt và truy cập:
👉 **[https://database-project-mini-netflix.onrender.com](https://database-project-mini-netflix.onrender.com)**

### 6.2 Khởi chạy cục bộ (Local Setup)

#### Bước 1: Khởi tạo CSDL MySQL
Đăng nhập vào MySQL và thực thi nạp cấu trúc cùng dữ liệu mẫu:
```bash
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS mininetflix;"
mysql -u root -p mininetflix < schema.sql
mysql -u root -p mininetflix < seed.sql
```

#### Bước 2: Khởi động Backend API & Web Portal
Cấu hình biến môi trường và chạy ứng dụng Go:
```bash
cd mininetflix-api

# Cách 1: Sử dụng chuỗi DSN đầy đủ
export DB_DSN="root:password@tcp(127.0.0.1:3306)/mininetflix?parseTime=true&charset=utf8mb4"

# Cách 2: Sử dụng các biến thành phần (hoặc để mặc định)
export DB_USER="root"
export DB_PASSWORD="your_password"
export DB_HOST="127.0.0.1"
export DB_PORT="3306"
export DB_NAME="mininetflix"
export PORT="8080"

go run main.go
```

Mở trình duyệt tại địa chỉ: `http://localhost:8080`

---

## Verification & Testing (Kiểm thử Quy tắc Nghiệp vụ)

Hệ thống cung cấp sẵn các bộ dữ liệu mẫu trong [`seed.sql`](file:///d:/Repository/Database-project-Mini-Netflix/seed.sql) để kiểm chứng 5 Business Rules:

### 7.1 Test BR-01: Subscription Access Control
* **Kịch bản:** Tài khoản của David (Account ID `4`, Profile ID `7`) đang ở gói **Free (Tier 0)**. Khi gọi API catalog, hệ thống chỉ trả về các tựa phim có `minimum_tier_id <= 0`.
* **cURL kiểm tra:**
  ```bash
  curl -X GET "http://localhost:8080/api/catalog?profile_id=7"
  ```
* **Kết quả mong đợi:** Tất cả các bản ghi trả về đều có `minimum_tier_id = 0`, không xuất hiện phim thuộc Tier 1, 2, 3.

### 7.2 Test BR-02: Concurrent Streaming Limit
* **Kịch bản:** Tài khoản của Bob (Account ID `2`) sử dụng gói **Standard (Tier 2, tối đa 2 streams đồng thời)**. Khi cố gắng khởi tạo stream thứ 3, server sẽ từ chối và trả về HTTP `409 Conflict`.
* **cURL kiểm tra:**
  ```bash
  # Stream 1 (Thành công - HTTP 201)
  curl -X POST "http://localhost:8080/api/sessions/start" -H "Content-Type: application/json" -d '{"account_id":2,"profile_id":4,"device_fingerprint":"dev-1"}'

  # Stream 2 (Thành công - HTTP 201)
  curl -X POST "http://localhost:8080/api/sessions/start" -H "Content-Type: application/json" -d '{"account_id":2,"profile_id":4,"device_fingerprint":"dev-2"}'

  # Stream 3 (Bị chặn - HTTP 409 Conflict)
  curl -X POST "http://localhost:8080/api/sessions/start" -H "Content-Type: application/json" -d '{"account_id":2,"profile_id":4,"device_fingerprint":"dev-3"}'
  ```

### 7.3 Test BR-03: Profile Data Isolation
* **Kịch bản:** Watchlist của Profile `1` (Alice Main) hoàn toàn độc lập với Profile `4` (Bob Main).
* **cURL kiểm tra:**
  ```bash
  curl -X GET "http://localhost:8080/api/watchlist?profile_id=1"
  curl -X GET "http://localhost:8080/api/watchlist?profile_id=4"
  ```

### 7.4 Test BR-04: Kids Profile Restrictions
* **Kịch bản:** Alice Junior (Profile ID `2`) có `is_kids_profile = true`. Khi truy vấn danh mục, hệ thống tự động loại trừ toàn bộ nội dung dán nhãn `R` và `NC-17`.
* **cURL kiểm tra:**
  ```bash
  curl -X GET "http://localhost:8080/api/catalog?profile_id=2"
  ```
* **Kết quả:** Danh mục trả về chỉ gồm `G`, `PG`, `PG-13`; không có bất kỳ phim `R` hay `NC-17`.

### 7.5 Test BR-05: Account Profile Threshold (1 <= N <= 5)
* **Kịch bản:** Khi tài khoản đã đạt đủ 5 profile, việc gửi request tạo thêm profile sẽ bị chặn với mã lỗi HTTP `409 Conflict`.
* **cURL kiểm tra:**
  ```bash
  curl -X POST "http://localhost:8080/api/profiles" -H "Content-Type: application/json" \
    -d '{"account_id":1,"profile_name":"Extra Profile","is_kids_profile":false}'
  ```

---

## REST API Reference

| Method | Endpoint | Description | Request Query / Body |
| :--- | :--- | :--- | :--- |
| `GET` | `/` hoặc `/ui` | Giao diện Evaluation Portal tương tác trực quan | - |
| `GET` | `/health` | Kiểm tra tình trạng server & kết nối MySQL (Keep-alive) | - |
| `GET` | `/api/catalog` | Lấy danh mục phim (áp dụng lọc theo BR-01 & BR-04) | `?profile_id=1&type=Movie&search=Star` |
| `POST` | `/api/movies` | Thêm phim mới vào mô hình phân cấp EER | `{"title_name":"...","maturity_rating":"PG-13","minimum_tier_id":1,"synopsis":"...","duration_seconds":5400,"video_resolution":"1080p","storage_asset_url":"https://..."}` |
| `GET` | `/api/profiles` | Danh sách tất cả profile kèm thông tin gói cước | - |
| `POST` | `/api/profiles` | Tạo profile mới (ràng buộc BR-05 $\le 5$) | `{"account_id":1,"profile_name":"Bob","is_kids_profile":false}` |
| `GET` | `/api/watchlist` | Xem danh sách phim đã lưu của Profile | `?profile_id=1` |
| `POST` | `/api/watchlist` | Thêm phim vào Watchlist | `{"profile_id":1,"title_id":"..."}` |
| `POST` | `/api/ratings` | Đánh giá phim (Điểm 1 - 10) | `{"profile_id":1,"title_id":"...","score":9}` |
| `POST` | `/api/sessions/start` | Khởi tạo phiên xem phim trực tuyến (BR-02) | `{"account_id":1,"profile_id":1,"device_fingerprint":"chrome-win"}` |
| `GET` | `/api/sessions` | Xem danh sách các phiên xem phim đang hoạt động | `?account_id=1` |
| `POST` | `/api/history/progress` | Ghi nhận tiến độ xem phim & tính cờ hoàn thành | `{"profile_id":1,"media_unit_id":"...","playback_stop_seconds":5000}` |

---

## Security & Data Integrity

1. **Ràng buộc toàn vẹn quan hệ (Referential Integrity):** Toàn bộ các bảng liên quan (Profile, Watchlist, Ratings, Watch History, Active Sessions) đều được thiết lập khóa ngoại liên kết với `ON DELETE CASCADE` đảm bảo loại bỏ triệt để bản ghi mồ côi (orphaned records).
2. **Kiểm soát tính hợp lệ (Domain Constraints):** Sử dụng các ràng buộc `CHECK` nghiêm ngặt (`tier_id IN (0,1,2,3)`, `max_concurrent_streams IN (1,2,4)`, `score BETWEEN 1 AND 10`).
3. **Phân vùng dữ liệu (Multi-Tenant Composite Keys):** Phiên làm việc `active_session` tham chiếu khóa ngoại kép `(account_id, prof_id) REFERENCES profile(account_id, profile_id)` đảm bảo tính cô lập tuyệt đối, tránh tình trạng phiên stream gán sai tài khoản sở hữu.

---

## References

1. HuggingFace Datasets: [Netflix Movies and TV Shows Dataset (`hugginglearners/netflix-shows`)](https://huggingface.co/datasets/hugginglearners/netflix-shows).
2. MySQL 8.0 Reference Manual: [Constraint & Foreign Key Referencing](https://dev.mysql.com/doc/refman/8.0/en/).
3. TiDB Cloud Documentation: [MySQL Wire Compatibility & Serverless Driver](https://docs.pingcap.com/tidbcloud/).
4. Go Database/SQL Documentation: [Connection Pooling & Prepared Statements](https://pkg.go.dev/database/sql).
