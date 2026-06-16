# Data Models
**Last updated:** [date]

This document defines every entity in the platform: fields, types, constraints, indexes, and relationships. It is the authoritative reference before writing any migration, domain struct, or repository query.

---

## Conventions — read before adding any table

| Convention | Rule |
|---|---|
| Primary keys | `UUID` — `DEFAULT gen_random_uuid()` on all tables |
| Money | `INTEGER` in smallest currency unit (sen). `150000 = Rp 1.500`. **Never FLOAT.** |
| Timestamps | `TIMESTAMPTZ NOT NULL DEFAULT NOW()` — always timezone-aware |
| Audit fields | Every table has `created_at` and `updated_at`. Add `deleted_at` if soft-deletable. |
| Soft deletes | Use `deleted_at TIMESTAMPTZ` or a `status` enum — never hard-delete records that have financial or audit significance |
| Enums | Stored as `VARCHAR(N)` with a `CHECK` constraint — not PostgreSQL native enums (easier to migrate) |
| Naming | `snake_case` for all tables and columns |
| Foreign keys | Always explicit FK constraint — do not rely on application-level enforcement |
| Indexes | Add index on every FK column and every column used in WHERE or ORDER BY |

---

## Entity Relationships (overview)

```
organizations ──< locations ──< courts ──< bookings >── members
                                                 │
                                           payment_intents
                                                 │
                                          xendit_webhooks

members ──< membership_tiers
        ──< credits_wallet_transactions
        ──< leave_requests  [Phase 1 example module]

staff ──< attendance_records
      ──< shifts

locations ──< lounge_sessions ──< lounge_orders >── menu_items
          ──< class_sessions  ──< enrollments   >── class_templates
          ──< events          ──< event_registrations
```

---

## Core tables

### organizations
Top-level entity. A single Viva Sports Club entity, or a franchise group.

```sql
CREATE TABLE organizations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(200) NOT NULL,
    slug        VARCHAR(100) NOT NULL UNIQUE,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### locations
A physical club location. New locations are added via admin config — no code change.

```sql
CREATE TABLE locations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    name            VARCHAR(200) NOT NULL,          -- "Viva Sports Club — Kemang"
    address         TEXT NOT NULL,
    city            VARCHAR(100) NOT NULL,
    timezone        VARCHAR(50) NOT NULL DEFAULT 'Asia/Jakarta',
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_locations_organization_id ON locations(organization_id);
```

---

### members
Registered club members. Linked to auth (users table via member_id in JWT).

```sql
CREATE TABLE members (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id  UUID NOT NULL REFERENCES organizations(id),
    full_name        VARCHAR(200) NOT NULL,
    email            VARCHAR(255) NOT NULL UNIQUE,
    phone            VARCHAR(30),
    password_hash    VARCHAR(255) NOT NULL,
    membership_tier  VARCHAR(20) NOT NULL DEFAULT 'standard'
                     CHECK (membership_tier IN ('standard','silver','gold','platinum')),
    credits_balance  INTEGER NOT NULL DEFAULT 0,    -- in sen
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    joined_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_members_email            ON members(email);
CREATE INDEX idx_members_organization_id  ON members(organization_id);
```

---

### refresh_tokens
Server-side refresh token store. Enables true logout (revoke the token).

```sql
CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id   UUID NOT NULL REFERENCES members(id),
    token_hash  VARCHAR(255) NOT NULL UNIQUE,   -- SHA-256 of the opaque token
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_member_id   ON refresh_tokens(member_id);
CREATE INDEX idx_refresh_tokens_token_hash  ON refresh_tokens(token_hash);
```

---

### courts
A bookable court at a location.

```sql
CREATE TABLE courts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_id  UUID NOT NULL REFERENCES locations(id),
    name         VARCHAR(100) NOT NULL,        -- "Court A", "Court B"
    court_type   VARCHAR(20) NOT NULL
                 CHECK (court_type IN ('padel','tennis','futsal','badminton')),
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_courts_location_id ON courts(location_id);
```

---

### bookings
A confirmed or pending court reservation. Financial record — never hard-delete.

```sql
CREATE TABLE bookings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id           UUID NOT NULL REFERENCES members(id),
    court_id            UUID NOT NULL REFERENCES courts(id),
    location_id         UUID NOT NULL REFERENCES locations(id),
    booking_ref         VARCHAR(30) NOT NULL UNIQUE,    -- "VIV-20250701-0042"
    date                DATE NOT NULL,
    start_time          TIME NOT NULL,
    end_time            TIME NOT NULL,
    duration_minutes    INTEGER NOT NULL,
    status              VARCHAR(25) NOT NULL DEFAULT 'pending_payment'
                        CHECK (status IN (
                            'pending_payment','confirmed','cancelled','no_show','refunded'
                        )),
    total_amount_idr    INTEGER NOT NULL,               -- in sen, never float
    xendit_payment_id   VARCHAR(255) UNIQUE,            -- idempotency key
    qr_code_token       VARCHAR(255),                   -- used for check-in QR
    notes               TEXT,
    confirmed_at        TIMESTAMPTZ,
    cancelled_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bookings_member_id         ON bookings(member_id);
CREATE INDEX idx_bookings_court_id          ON bookings(court_id);
CREATE INDEX idx_bookings_location_id       ON bookings(location_id);
CREATE INDEX idx_bookings_date              ON bookings(date);
CREATE INDEX idx_bookings_status            ON bookings(status);
CREATE INDEX idx_bookings_xendit_payment_id ON bookings(xendit_payment_id);
```

---

### booking_slots
Normalized slot records for a booking. Supports multi-slot bookings (e.g. 2-hour block = 2 × 1hr slots).

```sql
CREATE TABLE booking_slots (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id  UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    slot_start  TIMESTAMPTZ NOT NULL,
    slot_end    TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_booking_slots_booking_id ON booking_slots(booking_id);
CREATE INDEX idx_booking_slots_dates      ON booking_slots(slot_start, slot_end);
```

---

### staff
Club staff members (coaches, front office, F&B, managers).

```sql
CREATE TABLE staff (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_id     UUID NOT NULL REFERENCES locations(id),
    full_name       VARCHAR(200) NOT NULL,
    email           VARCHAR(255) NOT NULL UNIQUE,
    phone           VARCHAR(30),
    password_hash   VARCHAR(255) NOT NULL,
    role            VARCHAR(30) NOT NULL
                    CHECK (role IN ('admin','manager','front_office','coach','fnb','maintenance')),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_staff_location_id ON staff(location_id);
CREATE INDEX idx_staff_email       ON staff(email);
```

---

### attendance_records
QR-based clock-in/clock-out for staff.

```sql
CREATE TABLE attendance_records (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    staff_id     UUID NOT NULL REFERENCES staff(id),
    location_id  UUID NOT NULL REFERENCES locations(id),
    clock_in     TIMESTAMPTZ NOT NULL,
    clock_out    TIMESTAMPTZ,
    shift_date   DATE NOT NULL,
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_attendance_staff_id    ON attendance_records(staff_id);
CREATE INDEX idx_attendance_shift_date  ON attendance_records(shift_date);
```

---

### menu_items
F&B items available at Viva Lounge.

```sql
CREATE TABLE menu_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_id  UUID NOT NULL REFERENCES locations(id),
    name         VARCHAR(200) NOT NULL,
    description  TEXT,
    category     VARCHAR(50) NOT NULL
                 CHECK (category IN ('food','beverage','snack','package')),
    price_idr    INTEGER NOT NULL,      -- in sen
    is_available BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_menu_items_location_id ON menu_items(location_id);
```

---

### lounge_orders
F&B orders placed at Viva Lounge.

```sql
CREATE TABLE lounge_orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_id     UUID NOT NULL REFERENCES locations(id),
    member_id       UUID REFERENCES members(id),    -- nullable: walk-in customers
    table_number    VARCHAR(20),
    status          VARCHAR(20) NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open','paid','cancelled','voided')),
    total_amount_idr INTEGER NOT NULL DEFAULT 0,    -- in sen
    payment_method  VARCHAR(30),
    notes           TEXT,
    created_by      UUID NOT NULL REFERENCES staff(id),
    closed_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lounge_orders_location_id ON lounge_orders(location_id);
CREATE INDEX idx_lounge_orders_member_id   ON lounge_orders(member_id);
CREATE INDEX idx_lounge_orders_status      ON lounge_orders(status);
```

---

### lounge_order_items
Line items for a lounge order.

```sql
CREATE TABLE lounge_order_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID NOT NULL REFERENCES lounge_orders(id) ON DELETE CASCADE,
    menu_item_id    UUID NOT NULL REFERENCES menu_items(id),
    quantity        INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_idr  INTEGER NOT NULL,   -- price at time of order (snapshot)
    subtotal_idr    INTEGER NOT NULL,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lounge_order_items_order_id ON lounge_order_items(order_id);
```

---

### notification_log
Audit trail for every notification sent (WhatsApp, email).

```sql
CREATE TABLE notification_log (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id    UUID REFERENCES members(id),
    channel      VARCHAR(20) NOT NULL CHECK (channel IN ('whatsapp','email','sms')),
    template_key VARCHAR(100) NOT NULL,     -- e.g. "booking_confirmed"
    recipient    VARCHAR(255) NOT NULL,     -- phone or email
    status       VARCHAR(20) NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending','sent','failed','bounced')),
    sent_at      TIMESTAMPTZ,
    error_msg    TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notification_log_member_id    ON notification_log(member_id);
CREATE INDEX idx_notification_log_status       ON notification_log(status);
CREATE INDEX idx_notification_log_created_at   ON notification_log(created_at);
```

---

## Template — adding a new table

Copy this block. Fill every field. Do not skip indexes or constraints.

```sql
CREATE TABLE [table_name] (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- FK to owner entity:
    [parent_id]     UUID NOT NULL REFERENCES [parent_table](id),
    -- Required fields:
    [field_name]    [TYPE] NOT NULL,
    -- Optional fields:
    [field_name]    [TYPE],
    -- Status / type fields use VARCHAR + CHECK:
    [status]        VARCHAR(20) NOT NULL DEFAULT '[default]'
                    CHECK ([status] IN ('[val1]','[val2]')),
    -- Money fields use INTEGER (sen):
    [amount]        INTEGER NOT NULL DEFAULT 0,
    -- Audit fields (always include):
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Soft delete (add if needed):
    deleted_at      TIMESTAMPTZ
);

-- Index every FK and every column used in WHERE / ORDER BY:
CREATE INDEX idx_[table]_[parent_id]  ON [table_name]([parent_id]);
CREATE INDEX idx_[table]_[field_name] ON [table_name]([field_name]);
```

**Go domain struct conventions:**
```go
type [Entity] struct {
    ID         uuid.UUID   `db:"id"`
    [ParentID] uuid.UUID   `db:"[parent_id]"`
    [Field]    string      `db:"[field_name]"`
    CreatedAt  time.Time   `db:"created_at"`
    UpdatedAt  time.Time   `db:"updated_at"`
    DeletedAt  *time.Time  `db:"deleted_at"` // pointer — nullable
}
```