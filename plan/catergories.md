# Budget-Server: How It Works & Categories Plan

## Summary

The **budget-server** is a Go HTTP API that manages **accounts** and **transactions** with a PostgreSQL backend. It uses a clear layering: **HTTP API → Handlers → Operator (for writes) or Storage Reader (for reads) → Storage (DB)**. Writes (create account, create transaction) go through a **queue-based operator** so they run in a transaction and don’t block the HTTP server; reads (list accounts, list transactions) hit the storage reader directly.

**Current resources**

- **Accounts**: Create (POST /v1/accounts), List (GET /v1/accounts). Stored in `accounts` table; create goes through the operator.
- **Transactions**: Create (POST /v1/transaction), List (POST /v1/transaction/list). Stored in `transactions` table with `account_id` and `category_id` (UUID). Create goes through the operator; it also updates the account balance in the same transaction.
- **Categories**: Not implemented yet. Transactions already store and filter by `category_id`, but there is no `categories` table or API. Implementing categories is the next step.

---

## Architecture Layers

| Layer | Responsibility | Key types / paths |
|-------|----------------|-------------------|
| **Entry** | `main.go` | Loads config, builds `Storage` and `OperatorDelegator`, starts REST server. |
| **API** | `api/routes.go` | CORS, logging middleware, Huma API setup, route registration. |
| **Handlers** | `internal/handlers/v1/{account,transaction,status}/` | Parse request, call Operator (writes) or Storage.Read() (reads), return HTTP response. |
| **Operator** | `internal/operator/` | Queue of actions; workers take an action, open a DB write transaction, run the action, commit or rollback. |
| **Actions** | `internal/operator/actions/` | `IAction.Perform(ctx, writer)` — create account, create transaction, etc. |
| **Storage** | `internal/storage/` | `Storage` exposes `Read()` (Reader) and `Write(ctx)` (Writer over a transaction). Reader/Writer are split by domain (account, transaction). |
| **DB** | PostgreSQL + Bob ORM | Migrations in `scripts/db_migrations/migrations/`; generated Bob code in `internal/storage/sqlconfig/bobgen/`. |

---

## Request Flow

### Read path (e.g. List Accounts, List Transactions)

- HTTP request → middleware (CORS, logging) → **Handler**.
- Handler uses **Storage.Read()** (e.g. `r.Storage.Read().Accounts`, `r.Storage.Read().Transactions`) to query the DB (no transaction needed for simple reads).
- Handler maps domain models to API DTOs and returns JSON.

### Write path (e.g. Create Account, Create Transaction)

- HTTP request → middleware → **Handler**.
- Handler builds an **Action** (e.g. `actions.CreateAccount`, `actions.CreateTransaction`) and calls **Operator.Process(ctx, action)**.
- **OperatorDelegator** puts the action on a single shared **channel** (queue). One of the **worker Operators** (default 4) picks it up.
- Worker calls **Storage.Write(ctx)** to start a DB transaction and get a **Writer**.
- Worker runs **action.Perform(ctx, writer)**. The action uses only `writer` (e.g. `writer.Account.Create`, `writer.Transaction.Insert`, `writer.Account.UpdateBalance`).
- Worker commits or rolls back the transaction, then sends the result back on the action’s response channel.
- Handler returns HTTP status (e.g. 201) or error.

So: **writes are serialized through the operator queue and run inside a single DB transaction per action.**

---

## Diagram: How the Budget-Server Works

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                    main.go                                   │
│  config → NewStorage() → NewOperatorDelegator(storage, 4) → api.Rest.Serve() │
└─────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              api/routes.go (Rest)                            │
│  CORS → Logging → Mux (Huma)                                                 │
│  Routes: GET /status, GET /v1/accounts, POST /v1/accounts,                   │
│          POST /v1/transaction, POST /v1/transaction/list                    │
└─────────────────────────────────────────────────────────────────────────────┘
                    │                                    │
        ┌───────────┴───────────┐            ┌──────────┴──────────┐
        │  READ (list)          │            │  WRITE (create)      │
        │  Handler uses         │            │  Handler uses        │
        │  Storage.Read()       │            │  Operator.Process()  │
        └───────────┬───────────┘            └──────────┬──────────┘
                    │                                    │
                    ▼                                    ▼
┌───────────────────────────────┐    ┌──────────────────────────────────────┐
│  Storage.Read()               │    │  OperatorDelegator                    │
│  ┌─────────────────────────┐  │    │  queue (chan ActionItem, cap 1000)    │
│  │ Reader                  │  │    │  numWorkers (e.g. 4)                   │
│  │  .Accounts  (list)      │  │    └──────────────────┬───────────────────┘
│  │  .Transactions (list)  │  │                       │
│  └───────────┬─────────────┘  │                       ▼
└──────────────┼────────────────┘    ┌──────────────────────────────────────┐
               │                      │  Operator (per worker)                │
               │                      │  for item := range queue {            │
               │                      │    writer, _ := storage.Write(ctx)    │
               │                      │    item.action.Perform(ctx, writer)   │
               │                      │    writer.Commit() / Rollback()       │
               │                      │  }                                    │
               │                      └──────────────────┬───────────────────┘
               │                                         │
               │                                         ▼
               │                      ┌──────────────────────────────────────┐
               │                      │  Actions (Perform(ctx, writer))      │
               │                      │  CreateAccount → writer.Account.Create│
               │                      │  CreateTransaction →                 │
               │                      │    writer.Account.FindByIDForUpdate   │
               │                      │    writer.Transaction.Insert          │
               │                      │    writer.Account.UpdateBalance       │
               │                      └──────────────────┬───────────────────┘
               │                                         │
               ▼                                         ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  Storage.Write(ctx) → NewWriter(tx)                                          │
│  Writer: .Account (IAccountWriter), .Transaction (ITransactionWriter)        │
│  tx = db.Begin(); commit/rollback in Operator                               │
└─────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  PostgreSQL                                                                  │
│  tables: transactions (id, account_id, category_id, amount, ...),           │
│         accounts (id, name, type, sub_type, balance, ...)                    │
│  categories table: NOT YET (to be added)                                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Data Flow Summary

| Operation | Handler | Uses | Storage / Operator |
|-----------|---------|------|--------------------|
| GET /status | status.Handler | Operator (only for reference) | — |
| GET /v1/accounts | ListAccountsHandler | Storage.Read().Accounts | Reader.List(filter) |
| POST /v1/accounts | CreateAccountHandler | Operator.Process(CreateAccount) | Writer.Account.Create(...) |
| POST /v1/transaction/list | ListTransactionsHandler | Storage.Read().Transactions | Reader.List(filter), filter by AccountID/CategoryID optional |
| POST /v1/transaction | CreateTransactionHandler | Operator.Process(CreateTransaction) | Writer.Account.FindByIDForUpdate, Writer.Transaction.Insert, Writer.Account.UpdateBalance |

---

## Implementing Categories — Current State

- **Transactions** already have `category_id` (UUID) in DB and in all transaction types (create, list, filter). List transactions supports optional `CategoryID` in `TransactionFilter`.
- There is **no** `categories` table and **no** category CRUD or list API. Any UUID can be sent as `categoryID` when creating a transaction (no referential check).

---

# Implementation Plan: Categories

## Requirements

### Category model

Each category has:

| Property | Description | Type / notes |
|----------|-------------|--------------|
| **Name** | Display name of the category | Text, required |
| **Is group** | Whether this category is a group (can have child categories) | Boolean; **immutable** — set only at creation, cannot be changed on update |
| **Parent** | The group this category belongs to | **Required** when `is_group = false` (every category must be in a group). Optional when `is_group = true` (root groups have NULL). `parent_id` (UUID) references another category; when set, parent must exist and have `is_group = true` (validated on create and update). |
| **Should be budgeted** | Whether the category participates in budgeting | Boolean (no budgeting logic implemented yet) |
| **Is disabled** | If true, category cannot be used for new transactions | Boolean |
| **Type** | Direction of the category (e.g. expense vs income) | SMALLINT; stored as `category_type` in DB |

### Category hierarchy (parent–child)

- **Single table, self-referential**: One `categories` table with an optional `parent_id` column that references `categories(id)`. No separate “group” table.
- **Semantics**:
  - **Top-level**: Only **groups** can be top-level. A row with `parent_id` NULL must have `is_group = true` (a root group).
  - **Category in a group**: Any category that is **not** a group (`is_group = false`) **must** have `parent_id` set — i.e. **every category must at least be in a group**. The parent must exist and have `is_group = true`.
- **Rules**:
  1. **Every category must be in a group.** If `is_group = false`, then `parent_id` is **required** (cannot be NULL). Only groups (`is_group = true`) may have `parent_id` NULL (root groups). Enforce on create and update.
  2. If `parent_id` is set (on create or update), the referenced row must exist and must have `is_group = true` (enforce on create and update).
  3. **`is_group` is immutable.** It is set once at creation and cannot be changed. Do not allow updates to `is_group`.
  4. **Transactions are not assignable to a group.** A transaction’s `category_id` must reference a category with `is_group = false` (a leaf category under a group). Reject with a clear error if the category has `is_group = true`.
  5. Optional: disallow cycles (parent_id chain never leads back to the same row). For MVP, allowing one level (parent → child) is enough; deeper nesting can share the same schema.
- **List behavior**: List categories can return a flat list with `parent_id` (and optional `parent_name`) so the client can build a tree, or the API can return a tree. Flat list with `parent_id` is simpler for MVP and keeps the API flexible.

### Validation

- **Create or update category — must be in a group**: **Every category must at least be in a group.** When creating or updating a category with `is_group = false`, `parent_id` is **required** (cannot be NULL). Reject with a clear error (e.g. “category must be in a group; parentID is required for non-group categories”). On update, do not allow clearing `parent_id` for a non-group category.
- **Create or update category — parent**: When creating or editing a category, if `parent_id` is set (or is being set/changed on update), validate that the parent category **exists** and has **`is_group = true`**. Reject with a clear error if the parent is missing or is not a group (e.g. “parent category not found” or “parent must be a group”).
- **Create or update category — is_group immutable**: **`is_group` is immutable.** It is set once at creation and must not be changeable on update. The API must not accept or must ignore `is_group` on update; if the client sends a different value than the existing one, reject with a clear error (e.g. “is_group cannot be changed”).
- **New transactions** must validate that the referenced category exists, is **valid**, is **enabled** (not disabled), and is **not a group** (`is_group = false`). Reject the request with a clear error if the category is missing, invalid, disabled, or is a group (parent). Only leaf or standalone categories may be assigned to transactions.

### Editing categories

- Categories must be **editable** after creation. Provide an **update** endpoint (e.g. PATCH or PUT `/v1/categories/{id}`) so clients can change mutable fields: name, parent (parent_id), should_be_budgeted, is_disabled, category_type. `is_group` is immutable and must not be changeable on update. Parent validation applies when setting or changing parent_id.

### Database relations (Primary keys and Foreign keys)

- **Primary keys**: Every table has a primary key.
  - `categories.id` — UUID, PRIMARY KEY.
  - `accounts.id` — UUID, PRIMARY KEY (existing).
  - `transactions.id` — UUID, PRIMARY KEY (existing).

- **Foreign keys (required)**:
  - **`categories.parent_id`** → `categories(id)` (nullable). Self-referential: a category’s parent must be an existing category. Enforces that parent exists at the DB level.
  - **`transactions.category_id`** → `categories(id)` (NOT NULL). Every transaction must reference an existing category. The application additionally enforces that the category is not disabled and is not a group; the FK ensures the category exists.

- **Migration order**: Create the `categories` table (with its self-referential `parent_id` FK) first. Then add `ALTER TABLE transactions ADD CONSTRAINT ... FOREIGN KEY (category_id) REFERENCES categories(id)`. If `transactions` was created before `categories` existed, the `category_id` column already exists; the new migration only adds the FK constraint. The category direction column is **`category_type`** (SMALLINT).

---

## Pros and Cons

### Pros

- **Single source of truth**: Categories table gives a consistent set of options for transactions and future budgeting.
- **Flexibility**: “Is group” and “should be budgeted” set up for future hierarchy and budget features without changing the schema later. A single self-referential `parent_id` keeps the model simple and supports one or more levels of grouping without a separate table.
- **Expense vs income**: Makes it easy to filter or report by direction and prevents mixing semantics (e.g. using an “income” category on an expense transaction) if enforced later.
- **Soft disable**: “Is disabled” lets you hide or retire categories without deleting history; existing transactions keep their `category_id`, new ones cannot use it.
- **Validation in create transaction**: Ensures data integrity and gives users clear errors (e.g. “Category not found” or “Category is disabled”) instead of silent bad references.

### Cons

- **Stricter API**: Create transaction can fail for invalid/disabled category; clients must handle 4xx and surface messages. Slightly more work for clients.
- **Extra read in write path**: Create transaction must look up the category (in the same transaction as the insert) to validate; one more query per create. Acceptable for typical volume.
- **Schema surface**: More columns and indexes to maintain; migrations and Bob codegen must be run when adding categories.

### Other considerations

- **Every category must be in a group**: Non-group categories (`is_group = false`) must have `parent_id` set. Enforce on create (reject if is_group = false and parent_id is missing) and on update (reject if trying to clear parent_id for a non-group category). Only groups may be top-level (parent_id NULL).
- **Parent validation on create/update**: When creating or updating a category with `parent_id` set (or when changing `parent_id` on update), validate that the parent exists and has `is_group = true`. Return a clear error otherwise (e.g. “parent category not found” or “parent must be a group”).
- **Immutability of is_group**: Do not allow `is_group` to be changed after creation. On update, either omit `is_group` from the request body (and do not update it) or reject the request if the client sends a value that differs from the stored value. This avoids converting a group into a leaf (or vice versa) and keeps hierarchy semantics consistent.
- **Transactions and groups**: Transactions must not be assignable to a group. Validation in create-transaction must require `is_group = false` for the category; return a clear error (e.g. “category is a group; transactions must use a leaf or standalone category”) if the category is a group.
- **Type and transaction amount**: Today, transaction amount sign might already imply direction. Consider whether category type (expense/income) should match transaction sign, or be advisory for reporting only; document the rule and enforce in validation if desired.
- **Ordering / display**: List categories may need a stable sort (e.g. by parent then name, or a future `sort_order`). Flat list with `parent_id` lets the client sort or build a tree.
- **Future budgeting**: “Should be budgeted” is a flag only; no budgeting logic in this phase. Later you can add budget amounts, periods, and rules that only consider categories where `should_be_budgeted = true`. Group roll-ups (e.g. sum children into parent) can use the parent_id relationship.
- **FK constraints**: (1) `transactions.category_id` → `categories(id)` for referential integrity. (2) `categories.parent_id` → `categories(id)` so the DB enforces that parent exists; application enforces “parent must have is_group = true” and “not disabled” where needed.

---

# Implementation Phases and Steps

Phases are ordered by dependency: **Database → Storage → Operator/Actions → API/Handlers**. Each phase is sized as a reviewable PR (~200–500 lines) and includes implementation and testing tasks. Tasks are commit-sized.

---

## Phase 1: Database schema and Bob codegen

**Goal:** Add the `categories` table and the `transactions.category_id` FK. No application code yet; storage and API will consume this in later phases.

**Dependencies:** None.

| # | Task | Description |
|---|------|-------------|
| 1.1 | Migration: create `categories` table | Add migration `000003_create_categories_table.up.sql`: `id` (UUID PK), `name` (TEXT NOT NULL), `is_group` (BOOLEAN NOT NULL), `parent_id` (UUID NULL, FK → `categories(id)`), `should_be_budgeted` (BOOLEAN NOT NULL), `is_disabled` (BOOLEAN NOT NULL), `category_type` (SMALLINT NOT NULL), `created_at` (TIMESTAMPTZ). Add `.down.sql` to drop the table. |
| 1.2 | Migration: add FK from `transactions` to `categories` | Add migration `000004_add_transactions_category_fk.up.sql`: `ALTER TABLE transactions ADD CONSTRAINT fk_transactions_category_id FOREIGN KEY (category_id) REFERENCES categories(id)`. Handle existing rows if needed (e.g. backfill or allow NULL temporarily per your policy). Add `.down.sql` to drop the constraint. |
| 1.3 | Run migrations and Bob codegen | Run migration tool and `bobgen` (or equivalent) so `internal/storage/sqlconfig/bobgen/` gets generated code for `categories` (e.g. `categories.bob.go`). |
| 1.4 | Tests | Add or extend migration/integration tests: apply up/down migrations for 000003 and 000004; optionally assert Bob-generated types compile and table exists. |

---

## Phase 2: Storage layer — category Reader, Writer, and model

**Goal:** Expose category read/write operations through the existing Storage pattern. No HTTP API yet.

**Dependencies:** Phase 1 (categories table and Bob codegen exist).

| # | Task | Description |
|---|------|-------------|
| 2.1 | Category model and filter | In `internal/storage/category/` (or equivalent): define domain struct `Category` (id, name, is_group, parent_id, parent_name optional, should_be_budgeted, is_disabled, category_type, created_at) and `CategoryFilter` for list (e.g. limit, offset, optional parent_id, is_disabled). Map Bob row types to this model. |
| 2.2 | Category Reader | Implement `category.Reader` with `List(ctx, filter)` returning a flat list of categories (with `parent_id`; optionally `parent_name` if joined). Follow `account.Reader` / `transaction.Reader` pattern. Add `GetByID(ctx, id)` for single-category fetch (used later for validation). |
| 2.3 | Category Writer interface and implementation | Define `ICategoryWriter` with `Create(ctx, ...)` and `Update(ctx, id, ...)` (mutable fields only; no `is_group` on update). Implement `category.Writer` using Bob, with create and update. Enforce in implementation or caller: non-group requires `parent_id`; parent must exist (DB FK handles existence; app can validate `is_group` in action layer). |
| 2.4 | Wire category into Storage Reader and Writer | In `internal/storage/reader.go`: add `Categories *category.Reader` and construct it in `NewReader`. In `internal/storage/writer.go`: add `Category ICategoryWriter` and construct in `NewWriter`; add `NewWriterForTest` support for category mock. |
| 2.5 | Tests | Unit tests for `category.Reader` (List, GetByID) and `category.Writer` (Create, Update) using test DB or mocks. Integration test that Storage.Read().Categories and Storage.Write(ctx).Category work end-to-end. |

---

## Phase 3: Operator and actions — CreateCategory, UpdateCategory, and transaction category validation

**Goal:** Writes for categories go through the operator; create/update category validation (parent, is_group, "must be in a group") lives here. CreateTransaction validates category (exists, enabled, not a group).

**Dependencies:** Phase 2 (Storage exposes category Reader/Writer).

| # | Task | Description |
|---|------|-------------|
| 3.1 | CreateCategory action | Add `internal/operator/actions/create_category.go`: build category row from input (name, is_group, parent_id, should_be_budgeted, is_disabled, category_type). Validate: if `is_group == false`, require `parent_id`; if `parent_id` set, load parent and ensure it exists and `is_group == true`. Call `writer.Category.Create(...)`. |
| 3.2 | UpdateCategory action | Add `internal/operator/actions/update_category.go`: load existing category by id; reject if not found. Validate: do not allow changing `is_group`; if updating `parent_id`, require parent exists and has `is_group == true`; for non-group category do not allow clearing `parent_id`. Call `writer.Category.Update(...)` with mutable fields only. |
| 3.3 | CreateTransaction: category validation | In `CreateTransaction.Perform`: before inserting the transaction, use `writer.Category` (or a read within the same tx) to resolve the category by id. Reject with clear errors if: category missing, disabled, or is a group. Only then call `writer.Transaction.Insert` and account balance update. |
| 3.4 | Tests | Unit tests for CreateCategory (success, validation failures: missing parent_id for non-group, parent not a group, parent not found). Unit tests for UpdateCategory (success, is_group immutable, parent validation, cannot clear parent_id for non-group). Unit tests for CreateTransaction with invalid/disabled/group category (expect specific errors). |

---

## Phase 4: API and handlers — category list, create, update, and route wiring

**Goal:** REST API for categories and correct error responses for create-transaction when category is invalid.

**Dependencies:** Phase 3 (actions and validation in place).

| # | Task | Description |
|---|------|-------------|
| 4.1 | Category API types and list handler | Define Huma request/response types for list categories (e.g. query params: limit, offset; response: flat list with id, name, is_group, parent_id, parent_name?, should_be_budgeted, is_disabled, category_type). Implement `ListCategoriesHandler` using `Storage.Read().Categories.List(ctx, filter)`. Register GET (or POST list) route, e.g. `GET /v1/categories` or `POST /v1/categories/list`. |
| 4.2 | Create category handler | Define request body for create (name, is_group, parent_id optional for groups, should_be_budgeted, is_disabled, category_type). Implement `CreateCategoryHandler`: parse body, call `Operator.Process(ctx, actions.CreateCategory{...})`, return 201 with created category or 4xx with validation error message. Register `POST /v1/categories`. |
| 4.3 | Update category handler | Define request body for update (name, parent_id, should_be_budgeted, is_disabled, category_type; no `is_group`). Implement `UpdateCategoryHandler`: parse id from path and body, call `Operator.Process(ctx, actions.UpdateCategory{...})`, return 200 or 4xx. Register `PATCH /v1/categories/{id}` or `PUT /v1/categories/{id}`. |
| 4.4 | Create transaction error mapping | Ensure create-transaction handler maps operator/validation errors to appropriate HTTP status and body (e.g. 400/404 with "Category not found", "Category is disabled", "Category is a group; use a leaf category"). |
| 4.5 | Tests | Handler tests for list categories (empty and non-empty). Handler tests for create category (success, validation errors). Handler tests for update category (success, is_group immutable rejected, parent validation errors). Handler test for create transaction with invalid/disabled/group category returns expected status and message. |

---

## Phase summary

| Phase | Focus | Approx. size | Delivers |
|-------|--------|--------------|----------|
| 1 | DB | ~100–150 lines (migrations + config) | Categories table, transactions FK, Bob models |
| 2 | Storage | ~200–400 lines | Category Reader/Writer, Storage wiring, model |
| 3 | Operator/Actions | ~200–350 lines | CreateCategory, UpdateCategory, CreateTransaction category checks |
| 4 | API/Handlers | ~250–450 lines | List/Create/Update category endpoints, create-transaction error responses |

Each phase should be merged before starting the next so that DB → storage → operator → API dependencies stay clear and reviews stay focused.
