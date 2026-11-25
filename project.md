# Spesifikasi Teknis & Rancangan Lengkap: BakeOrder (UAS Kelompok 1)

> **Repositori GitHub:** https://github.com/Tullannghoda/UAS_PBL_K1.git  
> **Tujuan Dokumen:** Dokumen spesifikasi teknis serinci-rincinya (*single source of truth*) untuk seluruh anggota tim dan asisten AI masing-masing saat mengimplementasikan kode per branch.  
> **Catatan Scope Dosen:** Cakupan endpoint dipersempit ke **inti-intinya saja (15 endpoint)** dengan data katalog memakai **SQL seeder**, namun **seluruh Modul 1–12 tetap diterapkan** dan pekerjaan dibagi merata ke **3 anggota** lewat Pull Request & micro commits.

---

## 1. Aturan Arsitektur & Konvensi Kode Wajib (Konteks untuk AI & Tim)

Setiap anggota (dan AI yang membantu) **wajib** mematuhi konvensi proyek berikut agar kode dari 3 orang menyatu bersih tanpa konflik gaya:

1. **Bahasa & Framework:** Go (`go 1.26+`), `github.com/gofiber/fiber/v2`, PostgreSQL (`github.com/jackc/pgx/v5/pgxpool`), MongoDB (`go.mongodb.org/mongo-driver/v2`), Redis + Asynq (`github.com/hibiken/asynq`), JWT (`github.com/golang-jwt/jwt/v5`), Validator (`github.com/go-playground/validator/v10`), Swagger (`github.com/swaggo/swag` + `github.com/gofiber/swagger`).
2. **Clean Architecture Sesuai Modul Praktikum:**
   - **DILARANG keras** membuat folder, file, struct, atau istilah `controller` maupun `handler`.
   - `app/model`: hanya struct domain, request DTO (dengan tag `json` & `validate`), serta response DTO bernama (jangan kembalikan `fiber.Map` pada endpoint yang didokumentasikan Swagger). Tidak boleh mengimpor `fiber` atau driver database.
   - `app/repository`: definisikan **interface** terlebih dahulu beserta implementasi struct privat (`*PostgresRepository` / `*MongoRepository`). Repository PostgreSQL menerima `DBTX` dan menyediakan method `WithTx(tx DBTX)` untuk transaksi.
   - `app/service`: logika bisnis murni dipisah ke file `*_rules.go` (mudah di-unit-test tanpa DB/Fiber), sedangkan method HTTP Fiber (`func (s *XService) Method(c *fiber.Ctx) error`) ada di `*_service.go`.
3. **Standar Error & Response:**
   - Seluruh error bisnis mengembalikan `*helper.AppError` (via `helper.BadRequest`, `Unauthorized`, `Forbidden`, `NotFound`, `Conflict`, `Validation`, `NewError`, `Internal`) agar diformat seragam oleh centralized `ErrorHandler` di `config/app.go`.
   - Response sukses memakai `helper.Success`, `helper.Created` (mengisi header `Location`), `helper.SuccessCursor`, atau `helper.NoContent`.
4. **Keamanan:** Jangan pernah commit `.env`, `logs/`, atau `uploads/` (tambahkan ke `.gitignore`, gunakan `.env.example`).

---

## 2. Pemetaan Seluruh Modul Perkuliahan (Modul 1–12)

| Modul | Topik Perkuliahan | Implementasi Konkret di BakeOrder |
| --- | --- | --- |
| **Modul 1** | Persiapan & Sintaks Go Fiber | Server HTTP Fiber v2, flag `-role=api \| worker \| all`, pembacaan `.env` |
| **Modul 2** | REST API & HTTP Deep Dive | Method `GET`/`POST`/`PATCH`, status `200`/`201`/`400`/`401`/`403`/`404`/`409`/`422`, validasi struct, centralized `ErrorHandler` + `AppError` |
| **Modul 3** | Database & Repository Pattern | `pgxpool` PostgreSQL, file SQL `migrations/`, interface repository terpisah dari implementasi |
| **Modul 4** | Clean Architecture | Pemisahan `model`, `repository`, `service` (`*_rules.go` & `*_service.go`), `config`, `database`, `helper`, `middleware`, `route` |
| **Modul 5** | Authentication & Security | Password `bcrypt`, `POST /auth/register`, `POST /auth/login` (rate limiter), `POST /auth/refresh` (token rotation), JWT Bearer |
| **Modul 6** | Authorization & RBAC | Tabel `roles`, `permissions`, `role_permissions`; 3 role (`admin`, `staff`, `customer`); `RequireAuth` & `RequirePermission` |
| **Modul 7** | Advanced API Design | Cursor pagination + filter pada `GET /products` & `GET /orders`, `slog` structured logger + `lumberjack`, `RequestID`, `CORS`, header `Location` |
| **Modul 9** | Transactions & Concurrency | `TxManager.WithTx`, penguncian baris `SELECT ... FOR UPDATE` pada `daily_production_loads` saat `POST /orders` & `cancel` + optimistic locking (`version`) |
| **Modul 10** | API dengan MongoDB | Dual-DB `GET /health` (PostgreSQL + MongoDB) dan ulasan kue di MongoDB (`POST` & `GET /products/:id/reviews`, compound unique index `{product_id, user_id}`) |
| **Modul 11** | Background Jobs & Async Processing | Transactional Outbox (`outbox` ditulis dalam transaksi `POST /orders` & `PATCH /orders/:id/status`) -> Relay -> Redis/Asynq Worker -> `notifications` (`GET /notifications/mine`) |
| **Modul 12** | Testing, Swagger, & Docker | Unit test (`order_rules_test.go`), integration test (`//go:build integration`), anotasi Swagger + `go tool swag init` (`/swagger/*`), `Dockerfile` multi-stage non-root + `docker-compose.yml` (5 service) |

---

## 3. Profil Bisnis, Katalog, & Aturan Toko Kue

### 3.1 Ringkasan Sistem
- **Nama Sistem:** **BakeOrder** - Sistem Pemesanan Toko Kue *Made-to-Order* Berbasis Antrean & *Smart Dynamic Lead Time*.
- **Konsep Utama:** Toko **tidak menyimpan stok kue jadi di lemari etalase** (`products` tidak memiliki kolom `stock`). Seluruh kue baru diproduksi berdasarkan **daftar pesanan** yang masuk baik secara **online** (pelanggan) maupun **di tempat / walk-in** (diinput kasir/staff).
- **Data Wajib Pemesan:** Setiap pesanan mencatat **nama pemesan** (`customer_name`), **nomor telepon** (`customer_phone`), **jadwal pengambilan tanggal & jam** (`pickup_at`), serta kustomisasi per kategori.
- **Pembayaran Manual & Bayar di Tempat:** Tidak memakai payment gateway. Pelanggan membayar di tempat atau transfer/QRIS manual, lalu kasir/staff mengonfirmasi pembayaran via `PATCH /orders/:id/payment` (`unpaid` -> `dp_paid` / `paid`). Opsi `dp_paid` melindungi toko dari pembatalan sepihak untuk kue ulang tahun kustom atau pesanan besar.

### 3.2 Lima Kategori Kue & Aturan Kustomisasi (Diisi via SQL Seeder)

| No | Kategori (`slug`) | Produk Contoh | Varian (`product_variants`) | Topping & Kustomisasi | Minimal Order | Lead Time Dasar |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | **Kue Ulang Tahun** (`birthday-cake`) | Kue Tart, Blackforest | Ukuran diameter (`16 cm`, `20 cm`, `24 cm`) | Request warna, tema desain, bentuk, teks ucapan (`custom_spec` JSONB) + **upload gambar referensi** (`reference_image_url`) | 1 kue | **4 jam** (dinamis) |
| 2 | **Kue Bolu** (`sponge-cake`) | Bolu Original, Pandan, Mocca, Marmer, Coklat, Pisang, Tape | 7 varian rasa (`original`, `pandan`, `mocca`, `marmer`, `coklat`, `pisang`, `tape`) + ukuran loyang | Opsional request tambahan topping di atas: **seres**, **keju** | 1 loyang | **4 jam** (dinamis) |
| 3 | **Kue Ulang Tahun Lainnya** (`other-birthday-cake`) | Bolu Petak, Brownies | Ukuran loyang / jumlah potong | Request topping, teks ucapan, desain sederhana, opsional upload referensi | 1 loyang | **4 jam** (dinamis) |
| 4 | **Kue Basah** (`traditional-cake`) | Risol, Lapis, Donat, Dadar Gulung, Bolu Kukus, Kue Lumpur, Onde-Onde | Satuan per buah (`piece`) | Wajib penuhi **`min_order_qty`** produk (misal 10 atau 20 pcs). Khusus **Donat** bisa pilih topping: **seres**, **coklat crumble**, **tepung gula** | **Wajib >= `min_order_qty`** | **1 hari sebelum (`24 jam` / H-1)** |
| 5 | **Kue Kering** (`cookies`) | Nastar, Kastengel, Semprit, Skippy, Putri Salju, Chocochips | Ukuran toples (`jar`): **1/2 kg (500 g)** dan **1 kg (1000 g)** | Harga mengikuti varian ukuran toples (`1/2 kg` vs `1 kg`) | 1 toples | **2 minggu sebelum (`336 jam` / H-14)** |

---

## 4. Spesifikasi Smart Dynamic Lead Time (`app/service/order_rules.go`)

Karena produksi bersifat *made-to-order*, waktu pengambilan minimal (`earliest_pickup_at`) dihitung otomatis dari waktu pesan (`orderTime`), lead time dasar kategori, dan kepadatan antrean aktif pada tanggal target (`active_order_count` pada `daily_production_loads`).

### 4.1 Rumus Murni (Pure Functions)
```go
type LeadTimeInput struct {
    BaseLeadTimeHours int
    BusyThreshold     int
    ExtraHoursPerStep int
    MaxExtraHours     int
    ActiveOrderCount  int
}

func CalculateCategoryLeadHours(in LeadTimeInput) int {
    extra := 0
    if in.BusyThreshold > 0 && in.ExtraHoursPerStep > 0 {
        steps := in.ActiveOrderCount / in.BusyThreshold
        extra = steps * in.ExtraHoursPerStep
        if in.MaxExtraHours > 0 && extra > in.MaxExtraHours {
            extra = in.MaxExtraHours
        }
    }
    return in.BaseLeadTimeHours + extra
}
```
- **Contoh Kue Ulang Tahun** (`base=4`, `busy_threshold=5`, `extra_per_step=1`, `max_extra=4`):
  - Antrean aktif hari itu `0..4` -> `steps = 0` -> minimal ambil **`orderTime + 4 jam`**.
  - Antrean aktif hari itu `5..9` -> `steps = 1` -> minimal ambil otomatis naik menjadi **`orderTime + 5 jam`**.
  - Antrean aktif hari itu `10..14` -> `steps = 2` -> minimal ambil otomatis naik menjadi **`orderTime + 6 jam`**.
- **Contoh Kue Basah** (`base=24`, `busy_threshold=10`, `extra_per_step=12`, `max_extra=24`): normal minimal **24 jam (H-1)**; jika antrean padat (`>= 10`), naik menjadi 36 jam.
- **Contoh Kue Kering** (`base=336`, `busy_threshold=15`, `extra_per_step=24`, `max_extra=72`): normal minimal **336 jam (2 minggu / H-14)**.
- Jika satu pesanan berisi beberapa item lintas kategori, sistem mengambil nilai maksimum `earliest_pickup_at` dari seluruh kategori item yang dipesan.
- Jika `pickup_at < earliest_pickup_at`, API menolak dengan `422` atau `409` kode `LEAD_TIME_NOT_MET` beserta waktu minimal yang diizinkan.

### 4.2 Keamanan Transaksi & Konkurensi (`POST /orders` & `POST /orders/:id/cancel`)
1. Mulai transaksi lewat `txManager.WithTx`.
2. Untuk setiap `category_id` unik dalam pesanan pada tanggal `pickup_date = pickup_at.Date`:
   - `INSERT INTO daily_production_loads (production_date, category_id, active_order_count) VALUES (...) ON CONFLICT DO NOTHING`
   - Kunci baris dengan `SELECT ... FROM daily_production_loads WHERE production_date = $1 AND category_id = $2 FOR UPDATE`.
   - Hitung ulang `earliest_pickup_at` memakai `active_order_count` terbaru yang terkunci (mencegah 2 request serentak pada pesanan ke-5 dan ke-6 sama-sama lolos aturan 4 jam!).
   - Cek batas `max_daily_orders` (bila diatur > 0).
   - Increment `active_order_count = active_order_count + 1` dan `version = version + 1`.
3. Validasi `quantity >= product.min_order_qty`, validasi varian & topping milik produk terkait, dan hitung `unit_price` + `subtotal` + `total_price` di server (abaikan harga dari client!).
4. Simpan `orders`, `order_items`, `order_item_toppings`, `order_status_logs`, serta tulis event `order.created` ke `outbox` **di dalam transaksi yang sama**.
5. Saat `POST /orders/:id/cancel` (hanya boleh saat status `pending`), transaksi mengunci `orders` & `daily_production_loads` dengan `FOR UPDATE`, mengurangi `active_order_count` (`GREATEST(active_order_count - 1, 0)`), mengubah status menjadi `cancelled`, mencatat `order_status_logs`, dan memasukkan event `order.cancelled` ke `outbox`.

---

## 5. Rancangan Database Lengkap (PostgreSQL + MongoDB)

### 5.1 Skema PostgreSQL (`migrations/`)

- **`001_users_rbac.sql`**
  - `users`: `id`, `name`, `username` (UK), `email` (UK), `password_hash`, `phone`, `role` (FK ke `roles`), `is_active`, `created_at`
  - `refresh_tokens`: `id`, `user_id` (FK), `token_hash` (UK), `expires_at`, `revoked_at`, `created_at`
  - `roles` (`admin`, `staff`, `customer`), `permissions`, `role_permissions`:
    - `admin`: seluruh permission
    - `staff`: `order:list`, `order:read:any`, `order:create:walkin`, `order:update:status`, `order:confirm:payment`
    - `customer`: `order:create`, `order:read:own`, `order:cancel:own`, `review:create`
- **`002_catalog_seed.sql`**
  - `categories`: `id`, `slug` (UK), `name`, `description`, `base_lead_time_hours`, `busy_threshold`, `extra_hours_per_step`, `max_extra_hours`, `allows_custom_design`, `allows_reference_image`, `created_at`
  - `products`: `id`, `category_id` (FK), `name`, `description`, `base_price` (BIGINT), `unit_type` (`cake`/`pan`/`piece`/`jar`), `min_order_qty`, `is_active`, `created_at`
  - `product_variants`: `id`, `product_id` (FK), `variant_type` (`size`/`flavor`/`weight`), `name`, `price_adjustment` (BIGINT), `is_active`
  - `toppings`: `id`, `name` (UK), `extra_price` (BIGINT), `is_active`
  - `product_toppings`: `product_id` (PK, FK), `topping_id` (PK, FK)
  - **Seeder bawaan:** langsung mengisi 5 kategori, seluruh daftar produk (Tart, Blackforest, Bolu 7 rasa, Bolu Petak, Brownies, Risol, Lapis, Donat, Dadar Gulung, Bolu Kukus, Kue Lumpur, Onde-Onde, Nastar, Kastengel, Semprit, Skippy, Putri Salju, Chocochips), varian ukuran/rasa/toples (`1/2 kg`, `1 kg`), serta mapping topping (Bolu -> seres/keju; Donat -> seres/coklat crumble/tepung gula).
- **`003_orders.sql`**
  - `daily_production_loads`: `id`, `production_date`, `category_id` (FK), `active_order_count`, `max_daily_orders`, `version`, `updated_at`, `UNIQUE (production_date, category_id)`, `CHECK (active_order_count >= 0)`
  - `orders`: `id`, `order_code` (UK), `user_id` (FK nullable untuk walk-in), `customer_name`, `customer_phone`, `order_channel` (`online`/`walk_in`), `pickup_at` (`TIMESTAMPTZ`), `earliest_pickup_snapshot` (`TIMESTAMPTZ`), `notes`, `total_price` (`BIGINT`), `status` (`pending`/`confirmed`/`in_production`/`ready`/`completed`/`cancelled`/`rejected`), `payment_status` (`unpaid`/`dp_paid`/`paid`), `payment_method`, `paid_amount`, `payment_confirmed_by` (FK), `paid_at`, `created_at`, `updated_at`
  - `order_items`: `id`, `order_id` (FK), `product_id` (FK), `variant_id` (FK nullable), `quantity`, `unit_price`, `subtotal`, `custom_spec` (`JSONB`), `reference_image_url` (`TEXT`)
  - `order_item_toppings`: `id`, `order_item_id` (FK), `topping_id` (FK), `topping_name`, `extra_price`
  - `order_status_logs`: `id`, `order_id` (FK), `actor_id` (FK), `from_status`, `to_status`, `note`, `created_at`
- **`004_outbox_notifications.sql`**
  - `outbox`: `id`, `event_type` (`order.created`, `order.status_changed`, `order.cancelled`), `payload` (`JSONB`), `created_at`, `published_at` (index parsial `WHERE published_at IS NULL`)
  - `notifications`: `id`, `outbox_id`, `user_id` (FK), `channel`, `subject`, `body`, `sent_at`, `UNIQUE (outbox_id, user_id)`

### 5.2 Skema MongoDB (`product_reviews`)
Collection `product_reviews` menyimpan ulasan pelanggan terhadap kue yang pernah dipesan:
- Fields dokumen BSON: `_id` (`ObjectID`), `product_id` (`int`), `user_id` (`int`), `username` (`string`), `rating` (`1..5`), `comment` (`string`), `tags` (`[]string`), `created_at`, `updated_at`.
- Indexes:
  1. Unique compound index `{product_id: 1, user_id: 1}` (`product_user_unique`) -> 1 pelanggan maksimal 1 ulasan per produk.
  2. Compound index `{product_id: 1, _id: -1}` -> cursor pagination ulasan terbaru lebih dulu.

---

## 6. Spesifikasi 15 Endpoint Inti (`/api/v1`)

| No | Method & Path | Auth & Role / Permission | Modul | Kontrak Singkat Request & Response |
| --- | --- | --- | --- | --- |
| 1 | `GET /health` | Publik | 1, 3, 10 | Ping PostgreSQL & MongoDB -> `200` atau `503` |
| 2 | `POST /auth/register` | Publik | 2, 5 | Body `{name, username, email, phone, password}` -> `201` + `Location` |
| 3 | `POST /auth/login` | Publik (rate limit) | 5 | Body `{username, password}` -> `200` `TokenPair` (`access_token`, `refresh_token`) |
| 4 | `POST /auth/refresh` | Publik | 5 | Body `{refresh_token}` -> rotasi token -> `200` `TokenPair` |
| 5 | `GET /categories` | Publik | 3, 4 | Daftar 5 kategori beserta `base_lead_time_hours`, `busy_threshold`, aturan desain |
| 6 | `GET /products` | Publik | 3, 7 | Query `?category_id=&search=&cursor=&limit=` -> daftar produk, varian, topping, `min_order_qty` (cursor meta) |
| 7 | `POST /orders/estimate` | Terautentikasi | 4, 9 | Body `{pickup_at, items: [{product_id, variant_id, quantity, topping_ids}]}` -> mengembalikan `total_price`, `earliest_pickup_at`, dan flag apakah antrean padat (`is_busy`) |
| 8 | `POST /orders` | Login (`order:create` / `order:create:walkin`) | 2, 9, 11 | Mendukung JSON atau `multipart/form-data` (untuk upload foto referensi desain kue ultah). Validasi `customer_name`, `customer_phone`, `pickup_at`, `min_order_qty`, kunci `daily_production_loads` (`FOR UPDATE`), simpan order + outbox -> `201` |
| 9 | `GET /orders/mine` | Customer (`order:read:own`) | 6 | Daftar riwayat pesanan milik user yang sedang login |
| 10 | `GET /orders` | Staff / Admin (`order:list`) | 6, 7 | Query `?status=&pickup_date=&cursor=&limit=` -> daftar antrean pesanan untuk dapur/kasir |
| 11 | `PATCH /orders/:id/status` | Staff / Admin (`order:update:status`) | 6, 9, 11 | Body `{status, note}` (`confirmed`/`in_production`/`ready`/`completed`/`rejected`) -> update status + `order_status_logs` + event `outbox` |
| 12 | `PATCH /orders/:id/payment` | Staff / Admin (`order:confirm:payment`) | 6, 9 | Body `{payment_status, payment_method, paid_amount}` (`dp_paid` atau `paid`) -> konfirmasi manual oleh kasir |
| 13 | `POST /orders/:id/cancel` | Customer (`order:cancel:own`) | 9, 11 | Hanya saat `status == pending` -> ubah ke `cancelled`, kurangi `active_order_count` atomik + outbox |
| 14 | `POST /products/:id/reviews` & `GET /products/:id/reviews` | Login (`review:create`) / Publik (`GET`) | 10 | Simpan & baca ulasan produk di **MongoDB** (pelanggan harus pernah punya order `completed` untuk produk tersebut) |
| 15 | `GET /notifications/mine` | Terautentikasi | 11 | Daftar notifikasi status pesanan milik pelanggan yang sudah diproses worker Redis/Asynq |

*(Total 15 endpoint inti di `/api/v1`, ditambah endpoint dokumentasi `/swagger/*` pada `route.go` saat `SWAGGER_ENABLED=true`.)*

---

## 7. Struktur Folder Clean Architecture

```text
UAS_KEL1_PBL/
├── app/
│   ├── model/
│   │   ├── common.go           # WebResponse, ErrorResponse, CursorMeta
│   │   ├── user.go             # User, AuthUser, RegisterRequest, LoginRequest, TokenPair
│   │   ├── catalog.go          # Category, Product, ProductVariant, Topping, ProductReview
│   │   ├── order.go            # Order, OrderItem, CustomSpec, CreateOrderRequest, EstimateResponse, OrderStatusLog
│   │   └── notification.go     # OutboxEvent, NotificationJob, Notification
│   ├── repository/
│   │   ├── dbtx.go             # Interface DBTX & TxManager (WithTx / WithTxOnce)
│   │   ├── errors.go           # ErrNotFound, ErrDuplicate, ErrInvalidID, pgErrorCode helper
│   │   ├── user_repository.go
│   │   ├── token_repository.go
│   │   ├── role_repository.go
│   │   ├── catalog_repository.go
│   │   ├── order_repository.go
│   │   ├── review_repository.go       # MongoDB collection product_reviews
│   │   ├── outbox_repository.go
│   │   └── notification_repository.go
│   └── service/
│       ├── auth_rules.go
│       ├── auth_service.go
│       ├── order_rules.go             # Smart Dynamic Lead Time + min_order_qty + kalkulasi harga
│       ├── order_rules_test.go        # Unit test table-driven & subtest
│       ├── catalog_service.go
│       ├── order_service.go
│       ├── review_service.go
│       ├── outbox_relay.go
│       ├── notification_worker.go
│       └── notification_service.go
├── config/
│   ├── env.go
│   ├── logger.go
│   ├── queue.go
│   └── app.go                  # fiber.New + ErrorHandler terpusat
├── database/
│   ├── postgres.go
│   └── mongo.go
├── docs/                       # Hasil go tool swag init
├── helper/
│   ├── context.go
│   ├── cursor.go
│   ├── errors.go               # AppError & konstruktor error HTTP
│   ├── jwt.go
│   ├── mailer.go
│   ├── request.go
│   ├── response.go
│   ├── security.go
│   ├── upload.go               # Validasi & penyimpanan foto referensi desain
│   └── validator.go
├── middleware/
│   ├── auth.go
│   ├── authz.go
│   └── middleware.go
├── migrations/
│   ├── 001_users_rbac.sql
│   ├── 002_catalog_seed.sql
│   ├── 003_orders.sql
│   └── 004_outbox_notifications.sql
├── route/
│   └── route.go
├── tests/
│   └── integration/
│       ├── main_test.go
│       └── order_concurrency_test.go
├── .dockerignore
├── .env.example
├── .gitignore
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── main.go
├── project.md
└── README.md
```

---

## 8. Pembagian Jobdesk Detail, Branch, & Checklist File per Anggota

Aturan mutlak: **1 branch = 1 tugas**, semua perubahan masuk `main` lewat **Pull Request** dan di-merge oleh **Git Master (Orang 1)** menggunakan **Create a merge commit**. Tidak ada push langsung ke `main` setelah fase inisialisasi proyek.

### Orang 1 — Glennovian Benhur United Lumbantoruann / `434241044` (Git Master + Fondasi, Auth/RBAC, & Modul 11 Outbox/Worker)
1. **`chore/scaffold-core`** (atau setup awal `main`):
   - File: `go.mod`, `.gitignore`, `.env.example`, `config/env.go`, `config/logger.go`, `config/app.go`, `database/postgres.go`, `database/mongo.go`, `route/route.go` (kerangka + `GET /health`), `main.go`.
2. **`feat/auth-rbac`** (Modul 5 & 6):
   - File: `migrations/001_users_rbac.sql`, `app/model/user.go`, `app/repository/user_repository.go`, `token_repository.go`, `role_repository.go`, `helper/jwt.go`, `helper/security.go`, `middleware/auth.go`, `middleware/authz.go`, `app/service/auth_rules.go`, `app/service/auth_service.go`.
3. **`feat/outbox-notifications`** (Modul 11):
   - File: `migrations/004_outbox_notifications.sql`, `app/model/notification.go`, `app/repository/outbox_repository.go`, `notification_repository.go`, `config/queue.go`, `helper/mailer.go`, `app/service/outbox_relay.go`, `notification_worker.go`, `notification_service.go`, update `worker` wiring di `main.go`.

### Orang 2 — Alisya Aninda Safwa / `434241032` (Helper/Error + Seeder & Read Katalog + Modul 10 MongoDB + Swagger)
1. **`feat/helper-errorhandler`** (Modul 2 & 4):
   - File: `app/model/common.go`, `helper/errors.go`, `helper/response.go`, `helper/request.go`, `helper/validator.go`, `helper/cursor.go`, `helper/context.go`, `middleware/middleware.go`.
2. **`feat/catalog-module`** (Modul 3, 4, 7 — **Fitur Utuh Pertama**):
   - File: `migrations/002_catalog_seed.sql`, `app/model/catalog.go`, `app/repository/catalog_repository.go`, `app/service/catalog_service.go`, dan registrasi `GET /categories`, `GET /products` di `route/route.go`.
3. **`feat/mongodb-reviews`** (Modul 10):
   - File: `app/repository/review_repository.go`, `app/service/review_service.go`, registrasi `POST` & `GET /products/:id/reviews`.
4. **`docs/swagger-setup`** (Modul 12):
   - Menambahkan komentar anotasi Swagger di `main.go` dan file `app/service/*_service.go`, route `/swagger/*`, serta generate folder `docs/`.

### Orang 3 — Mellinda Impiawati / `434231032` (TxManager + Docs/ERD + Pesanan & Smart Lead Time + Test & Docker)
1. **`feat/database-txmanager`** (Modul 3 & 9):
   - File: `app/repository/dbtx.go`, `app/repository/errors.go`.
2. **`docs/readme-erd`** (Dokumentasi Proyek):
   - Melengkapi/memperbarui diagram ERD (`Mermaid` / `draw.io` / `dbdiagram.io`) dan contoh payload di `README.md` bila diperlukan.
3. **`feat/order-smart-leadtime`** (Modul 4, 6, 7, 9):
   - File: `migrations/003_orders.sql`, `app/model/order.go`, `app/repository/order_repository.go`, `helper/upload.go`, `app/service/order_rules.go` (Smart Dynamic Lead Time 4→5 jam, H-1 + `min_order_qty`, H-14), `app/service/order_service.go` (`Estimate`, `Create`, `Mine`, `List`, `UpdateStatus`, `ConfirmPayment`, `Cancel`), dan registrasi route pesanan.
4. **`chore/tests-docker`** (Modul 12):
   - File: `app/service/order_rules_test.go`, `tests/integration/main_test.go`, `tests/integration/order_concurrency_test.go`, `Dockerfile`, `.dockerignore`, `docker-compose.yml`.
