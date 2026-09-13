# Database Project: On-Demand Streaming Platform - Mini-Netflix

The proposed system is an On-Demand Streaming Platform (Mini-Netflix). The platform is a centralized multimedia delivery service designed to manage, categorize, and stream digital video content to end-users. 

To accurately manage this catalog, the system categorizes "Content" into two distinct hierarchical types:  

* **Movies:** Standalone video files characterized by a single duration and video resolution.   

* **TV Series Episodes:** Serialized media units that belong to a broader TV Series container, requiring complex tracking of seasons and episode numbers. 

> This repository accompanies the report:  
> **On-Demand Streaming Platform - Mini-Netflix**  
> Nhan D. Tran (n25dece106@student.ptithcm.edu.vn), Lam Vo (n25dece055@student.ptithcm.edu.vn), Hung M. To (n25dece092@student.ptithcm.edu.vn)

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

* **BR-01** — Subscription Access: A profile can access content only when the content's required subscription tier is less than or equal to the account's current tier.
* **BR-02** — Concurrent Streaming: The number of active streaming sessions for an account cannot exceed the maximum concurrent streams allowed by its subscription tier.
* **BR-03** — Profile Data Isolation: Watch history, watchlists, and ratings are managed independently for each profile. Activities performed by one profile must not affect another profile under the same account.
* **BR-04** — Kids Content Restriction: A Kids Profile must not be allowed to access content classified as R or TV-MA.
* **BR-05** — Profile Limit: Each account must have at least 1 and no more than 5 profiles.
* **BR-06** — Referential Integrity: Deleting an account must automatically remove its dependent profiles and related activity records, preventing orphaned data through appropriate cascading rules.


---

## Database Design

### 3.1 Conceptual Model

The conceptual model represents the main entities and relationships required to support the streaming platform. It organizes accounts, profiles, subscription tiers, content, genres, viewing activities, and streaming sessions while defining their cardinalities and dependencies. The model uses specialization to distinguish Movie and Episode as subtypes of Content, and associative entities to resolve many-to-many relationships such as content genres, watchlists, watch history, and profile ratings. This structure provides the foundation for mapping the system into a normalized relational schema.

### 3.2 Entity Overview

The database consists of the following main entities:

* **Subscription_Tier:** Defines available membership plans and their streaming limits and video quality.
* **Account:** Represents the primary customer account responsible for subscription and billing.
* **Profile:** Represents an individual user profile belonging to an account, with separate viewing preferences and activity data.
* **Content:** The superclass representing all streamable media in the platform.
* **Movie:** A standalone type of content with its own duration and video resolution.
* **TV_Series:** Represents a series that groups related episodes.
* **Episode:** A content subtype representing an individual episode within a TV series.
* **Genre:** Stores content classification categories.
* **Content_Genre:** Resolves the many-to-many relationship between content and genres.
* **Watchlist:** Stores content saved by profiles for later viewing.
* **Watch_History:** Records a profile's viewing activity and playback progress.
* **Profile_Ratings:** Stores ratings given by profiles to content.
* **Active_Session:** Tracks currently active streaming sessions associated with an account and profile.

### 3.3 Logical Schema

The logical schema translates the conceptual model into a set of relational tables with clearly defined primary keys, foreign keys, and constraints. It preserves the relationships between accounts, profiles, subscription tiers, content, genres, and viewing activities while supporting content specialization through the Content, Movie, and Episode tables. Many-to-many relationships are resolved using associative tables such as Content_Genre, Watchlist, Watch_History, and Profile_Ratings. This schema provides the structural basis for the database implementation and normalization process.

### 3.4 Relational Mapping

The conceptual entities are mapped into relational tables using primary keys to uniquely identify records and foreign keys to maintain relationships between related entities. The Content superclass is mapped together with its specialized Movie and Episode tables, while many-to-many relationships are represented through associative tables such as **Content_Genre**, **Watchlist**, **Watch_History**, and **Profile_Ratings**. Composite keys are used where necessary to uniquely identify relationship records, ensuring that the relational schema preserves the structure and constraints defined in the conceptual model.

### 3.5 Normalization



---

## Data Dictionary


---

## Database Implementation

### 5.1 Technology Stack

### 5.2 Project Structure

### 5.3 DDL

### 5.4 Constraints

### 5.5 Views

### 5.6 Indexes

### 5.7 Triggers

---

## Setup & Quickstart

### 6.1 Requirements

### 6.2 Installation

### 6.3 Database Initialization

### 6.4 Running Example Queries


---

## Verification & Testing

### 7.1 Constraint Tests

### 7.2 Referential Integrity Tests

### 7.3 Business Rule Tests

### 7.4 RBAC Tests

---

## Example Queries

### 8.1 Content Discovery

### 8.2 Watch History

### 8.3 Subscription Validation

### 8.4 Active Session Validation

---

## Security & Access Control

### 9.1 RBAC

### 9.2 GRANT / REVOKE

### 9.3 Data Integrity

---

## Reproducibility


---

## References