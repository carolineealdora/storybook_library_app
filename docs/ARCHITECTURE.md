# Architecture Overview — Viva Sports Club Platform
**Version:** 1.0 | **Last updated:** 2025-06-16
**Keep this under 300 lines. Detail lives in docs/. This is the map, not the manual.**

---

## System Purpose

A unified digital platform for Viva Sports Club covering court booking, member management, staff attendance, F&B (Viva Lounge), events, classes, loyalty, and business intelligence. Architected for multi-location from day one — new courts, lounges, or club locations are added via admin configuration, no code changes.

---

## Tech Stack

| Layer | Technology | Why |
|---|---|---|
| Backend API | Go 1.22 (Gin) | High concurrency for real-time slot booking; low memory overhead at scale |
| Frontend | React / Next.js 14 | SSR for SEO (marketing site); CSR for member portal |
| Database | PostgreSQL 16 | ACID compliance for bookings and financial transactions |
| Cache | Redis 7 | Real-time court slot availability; session storage |
| Payments | Xendit + QRIS | Primary gateway; QRIS for QR payment touchpoints |
| Notifications | WhatsApp Business API + SMTP | 8 automated triggers (Phase 1); ticket system (Phase 2) |
| Social | Instagram Graph API | Auto-sync gallery on marketing site |
| Auth | JWT (RS256) + refresh tokens | Stateless; refresh tokens in Redis |
| Hosting | AWS (backend) + Vercel (frontend) | Backend needs persistent compute; frontend benefits from edge CDN |
| Search | Google Search Console | SEO indexing for marketing site |

---

## System Boundaries

```
┌─────────────────────────────────────────────────────────┐
│  Public Web (Next.js / Vercel)                          │
│  Marketing site · Member portal · Booking flow          │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTPS / REST
┌──────────────────────▼──────────────────────────────────┐
│  Go API (AWS)                                           │
│  Auth · Booking · Members · Staff · F&B · Reporting     │
├──────────────┬────────────────┬───────────────────────  │
│  PostgreSQL  │  Redis         │  Background workers      │
│  (primary)   │  (slots/cache) │  (notifications, jobs)   │
└──────────────┴────────────────┴───────────────────────  │
                       │
┌──────────────────────▼──────────────────────────────────┐
│  External Services                                      │
│  Xendit · QRIS · WhatsApp API · Instagram API · SMTP    │
└─────────────────────────────────────────────────────────┘
```

---

## Key Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Multi-location model | Location ID on every resource (not separate DBs) | Simpler cross-location reporting; single migration path |
| Slot availability | Redis bitmask per court per day | Sub-10ms reads under concurrent booking load |
| Payment flow | Server-side charge initiation + webhook confirmation | Never trust client-reported payment status |
| Auth model | Short-lived JWT (15min) + Redis-stored refresh token | Revocable sessions; logout actually works |
| Soft deletes | `deleted_at` timestamp on all major tables | Audit trail; cancellations preserve history |
| Money storage | INTEGER (smallest currency unit — sen/cents) | Never store floats for money |
| Background jobs | Go goroutines + Redis queue (Phase 1); migrate to proper queue if needed | Start simple; right-size when volume justifies it |

---

## Module Map

### Phase 1 (MVP)
| Module | Key entities | Owner package |
|---|---|---|
| Auth | users, sessions, refresh_tokens | `internal/auth` |
| Court Booking | courts, bookings, slots, locations | `internal/booking` |
| Member Portal | members, membership_tiers, credits_wallet | `internal/member` |
| Staff Attendance | staff, attendance_records, shifts | `internal/attendance` |
| F&B / Viva Lounge | menu_items, orders, pos_sessions | `internal/lounge` |
| Classes | class_templates, class_sessions, enrollments | `internal/classes` |
| Events | events, event_registrations, tickets | `internal/events` |
| Pro Shop | products, inventory, purchase_orders | `internal/shop` |
| Payments | payment_intents, transactions, webhooks | `internal/payment` |
| Notifications | notification_log, templates, delivery_queue | `internal/notify` |
| Admin | roles, permissions, audit_log | `internal/admin` |
| Org Management | organizations, locations, staff_roles | `internal/org` |

### Phase 2
| Module | Key entities | Owner package |
|---|---|---|
| Matchmaking | open_matches, match_results, elo_ratings | `internal/matchmaking` |
| Loyalty | loyalty_points, leaderboard_snapshots, rewards | `internal/loyalty` |
| Player Stats | player_profiles, stats_snapshots | `internal/playerstats` |
| Leagues | leagues, league_registrations, match_schedule | `internal/league` |
| Coach Management | coaches, coach_sessions, coach_bookings | `internal/coach` |
| Dynamic Pricing | pricing_rules, time_slots_pricing | `internal/pricing` |
| BI Dashboard | analytics_events, report_snapshots | `internal/analytics` |

---

## Repository Structure

```
viva-api/                          # Go backend
├── cmd/
│   └── main.go                    # Entry point
├── internal/
│   ├── auth/
│   ├── booking/
│   │   ├── handler/               # HTTP layer
│   │   ├── service/               # Business logic
│   │   └── repository/            # DB layer
│   ├── [other modules same pattern]
│   ├── config/                    # Config loading (env vars)
│   ├── db/                        # DB connection pool
│   ├── middleware/                # Auth, logging, CORS
│   ├── response/                  # Shared JSON response helpers
│   └── validator/                 # Input validation helpers
├── db/
│   └── migrations/                # SQL migrations (numbered)
├── CLAUDE.md
├── PLAN.md
├── tech-debt.md
└── docs/
    ├── ARCHITECTURE.md            # ← this file
    ├── API-contracts.md
    ├── DATA-MODELS.md
    └── PRD-[feature].md

viva-web/                          # Next.js frontend
├── app/                           # App router
│   ├── (marketing)/               # Public marketing pages
│   ├── (member)/                  # Authenticated member portal
│   └── (admin)/                   # Admin dashboard
├── components/
├── lib/
│   └── api/                       # Typed API client
└── CLAUDE.md
```

---

## Data Flow: Court Booking (critical path)

```
Member selects slot
    → POST /api/v1/bookings/hold   (Redis lock, 5-min TTL)
    → POST /api/v1/payments/initiate  (Xendit charge)
    → [Xendit webhook] POST /webhooks/xendit
    → Service confirms payment + commits booking to PostgreSQL
    → Redis slot marked as booked
    → WhatsApp confirmation sent via notification queue
```

**Rule:** Slot is never committed to PostgreSQL until payment webhook confirms. The Redis hold prevents double-booking during payment processing.

---

## External Integration Touchpoints

| Service | Auth method | Error handling |
|---|---|---|
| Xendit | API key (server-side only) | Retry with exponential backoff; webhook idempotency key |
| QRIS | Merchant ID + secret | Same as Xendit |
| WhatsApp Business API | Bearer token | Queue-backed; failures logged, retried 3× |
| Instagram Graph API | OAuth access token | Best-effort; gallery sync failure non-fatal |
| SMTP | Credentials in env | Queue-backed; same as WhatsApp |

---

## What to read before making structural changes

- Payment flow: `internal/payment/` + `docs/API-contracts.md`
- Slot availability logic: `internal/booking/repository/slot.go`
- Auth token lifecycle: `internal/auth/` + this doc (Auth model decision above)
- Adding a new location: admin docs (config-only, no code change)
- Data models: `docs/DATA-MODELS.md`