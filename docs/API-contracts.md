# API Contracts
**Base URL:** `https://api.vivasports.id/api/v1`
**Last updated:** [date]

This document defines the contract for every API endpoint: method, path, auth requirement, request shape, success response, and error codes. It is the authoritative reference for both backend implementation and frontend integration.

**Rules:**
- Update this document when adding, changing, or removing any endpoint
- Endpoint changes are breaking if they alter the request shape, response shape, or error codes
- Breaking changes require a new API version prefix (`/api/v2/...`)

---

## How to read this document

Each endpoint follows this structure:

```
### [METHOD] /path
Auth:     Required | Public | Admin only
Purpose:  One sentence.

Request body (if applicable):
Response (success):
Error codes:
```

**Standard error response shape** (all errors):
```json
{
  "error": {
    "code": "MACHINE_READABLE_CODE",
    "message": "Human-readable description.",
    "fields": {}
  }
}
```

**Standard collection response shape:**
```json
{
  "data": [],
  "meta": { "total": 0, "page": 1, "page_size": 20 }
}
```

**Standard single-resource response shape:**
```json
{
  "data": {}
}
```

---

## Auth

### POST /auth/login
Auth: Public

Request:
```json
{ "email": "string", "password": "string" }
```

Response `201`:
```json
{
  "data": {
    "access_token":  "string (JWT, 15-min TTL)",
    "refresh_token": "string (opaque, 7-day TTL)",
    "expires_in":    900
  }
}
```

Error codes:
| Code | HTTP | When |
|---|---|---|
| `INVALID_CREDENTIALS` | 401 | Email not found or password incorrect |
| `VALIDATION_ERROR` | 422 | Missing required fields |

---

### POST /auth/refresh
Auth: Public

Request:
```json
{ "refresh_token": "string" }
```

Response `200`:
```json
{
  "data": {
    "access_token": "string (new JWT)",
    "expires_in":   900
  }
}
```

Error codes:
| Code | HTTP | When |
|---|---|---|
| `INVALID_REFRESH_TOKEN` | 401 | Token not found, expired, or already rotated |

---

### POST /auth/logout
Auth: Required

Request:
```json
{ "refresh_token": "string" }
```

Response `204`: No body.

---

## Members

### GET /members/me
Auth: Required
Purpose: Get the authenticated member's own profile.

Response `200`:
```json
{
  "data": {
    "id":              "uuid",
    "full_name":       "string",
    "email":           "string",
    "phone":           "string",
    "membership_tier": "standard | silver | gold | platinum",
    "credits_balance": 0,
    "joined_at":       "ISO8601"
  }
}
```

---

### PATCH /members/me
Auth: Required
Purpose: Update the authenticated member's own profile.

Request (all fields optional — only provided fields are updated):
```json
{
  "full_name": "string",
  "phone":     "string"
}
```

Response `200`: Updated member object (same shape as GET /members/me).

Error codes:
| Code | HTTP | When |
|---|---|---|
| `VALIDATION_ERROR` | 422 | Field fails validation (e.g. invalid phone format) |

---

## Courts

### GET /courts
Auth: Public
Purpose: List all active courts, optionally filtered by location.

Query params:
| Param | Type | Required | Description |
|---|---|---|---|
| `location_id` | uuid | No | Filter by location |
| `court_type` | string | No | `padel`, `tennis`, `futsal` |

Response `200`:
```json
{
  "data": [
    {
      "id":          "uuid",
      "name":        "Court A",
      "court_type":  "padel",
      "location_id": "uuid",
      "location_name": "Viva Sports Club — Kemang"
    }
  ],
  "meta": { "total": 4, "page": 1, "page_size": 20 }
}
```

---

### GET /courts/:id/availability
Auth: Public
Purpose: Return available and unavailable slots for a given court on a given date.

Query params:
| Param | Type | Required | Description |
|---|---|---|---|
| `date` | string | Yes | Format: `YYYY-MM-DD` |

Response `200`:
```json
{
  "data": {
    "court_id": "uuid",
    "date":     "2025-07-01",
    "slots": [
      {
        "start":    "07:00",
        "end":      "08:00",
        "status":   "available | booked | held",
        "price_idr": 15000000
      }
    ]
  }
}
```

Error codes:
| Code | HTTP | When |
|---|---|---|
| `INVALID_DATE` | 422 | Date is in the past or invalid format |
| `COURT_NOT_FOUND` | 404 | Court ID does not exist |

---

## Bookings

### POST /bookings/hold
Auth: Required
Purpose: Temporarily hold a slot for 5 minutes while the member proceeds to payment.

Request:
```json
{
  "court_id":   "uuid",
  "date":       "YYYY-MM-DD",
  "start_time": "HH:MM",
  "end_time":   "HH:MM"
}
```

Response `201`:
```json
{
  "data": {
    "hold_id":    "uuid",
    "expires_at": "ISO8601",
    "ttl_seconds": 300
  }
}
```

Error codes:
| Code | HTTP | When |
|---|---|---|
| `SLOT_UNAVAILABLE` | 409 | Slot is already booked or held |
| `INVALID_DATE_RANGE` | 422 | start_time in past, or end before start |
| `VALIDATION_ERROR` | 422 | Missing or malformed fields |

---

### POST /bookings
Auth: Required
Purpose: Create a booking and initiate payment. Must have a valid hold from POST /bookings/hold.

Request:
```json
{
  "hold_id":       "uuid",
  "payment_method": "xendit_va | xendit_qris | xendit_cc"
}
```

Response `201`:
```json
{
  "data": {
    "booking_id":      "uuid",
    "booking_ref":     "VIV-20250701-0042",
    "status":          "pending_payment",
    "total_amount_idr": 15000000,
    "payment": {
      "method":     "xendit_va",
      "expires_at": "ISO8601",
      "va_number":  "string (if VA method)",
      "qr_string":  "string (if QRIS method)"
    }
  }
}
```

Error codes:
| Code | HTTP | When |
|---|---|---|
| `HOLD_EXPIRED` | 409 | Hold TTL elapsed before booking was created |
| `HOLD_NOT_FOUND` | 404 | hold_id not found or does not belong to this member |
| `PAYMENT_INITIATION_FAILED` | 502 | Xendit returned an error |

---

### GET /bookings
Auth: Required
Purpose: List the authenticated member's own bookings.

Query params:
| Param | Type | Required | Default |
|---|---|---|---|
| `status` | string | No | all |
| `page` | int | No | 1 |
| `page_size` | int | No | 20 (max 100) |

Response `200`: Collection of booking summaries.
```json
{
  "data": [
    {
      "id":           "uuid",
      "booking_ref":  "VIV-20250701-0042",
      "court_name":   "Court A",
      "date":         "2025-07-01",
      "start_time":   "07:00",
      "end_time":     "08:00",
      "status":       "confirmed | pending_payment | cancelled | no_show",
      "total_amount_idr": 15000000
    }
  ],
  "meta": { "total": 12, "page": 1, "page_size": 20 }
}
```

---

### GET /bookings/:id
Auth: Required
Purpose: Get full detail of a single booking.

Response `200`: Full booking object.

Error codes:
| Code | HTTP | When |
|---|---|---|
| `BOOKING_NOT_FOUND` | 404 | Booking not found, or belongs to a different member |

> Returns 404 (not 403) on cross-member access — do not leak that the booking exists.

---

### DELETE /bookings/:id
Auth: Required
Purpose: Cancel a pending booking. Sets status to `cancelled`.

Response `204`: No body.

Error codes:
| Code | HTTP | When |
|---|---|---|
| `BOOKING_NOT_FOUND` | 404 | Booking not found or belongs to another member |
| `CANNOT_CANCEL_CONFIRMED` | 409 | Booking is already confirmed (approved) |
| `CANCELLATION_WINDOW_CLOSED` | 409 | Inside the no-cancellation window |

---

## Webhooks

### POST /webhooks/xendit
Auth: Public — HMAC-verified (X-CALLBACK-TOKEN header)
Purpose: Receive payment status updates from Xendit.

> This endpoint is called by Xendit, not by the frontend.
> Verify the `X-CALLBACK-TOKEN` header before processing. Reject unsigned requests with 401.
> This endpoint must be idempotent — duplicate webhooks for the same `xendit_payment_id` must not create duplicate bookings.

Request body: Xendit webhook payload (see Xendit docs for full schema).

Relevant fields:
```json
{
  "external_id": "string (our booking_id)",
  "status":      "PAID | EXPIRED | FAILED",
  "id":          "string (xendit_payment_id — idempotency key)"
}
```

Response `200`: Always respond 200 quickly. Xendit retries on non-2xx or timeout > 30s.

---

## Admin endpoints

> All `/admin/` endpoints require `admin` role in JWT claims.

### GET /admin/bookings
Auth: Admin only
Purpose: List all bookings across all members, filterable.

Query params: `status`, `location_id`, `date_from`, `date_to`, `page`, `page_size`

---

### PATCH /admin/bookings/:id/status
Auth: Admin only
Purpose: Manually override a booking status (e.g. mark as `no_show`).

Request:
```json
{ "status": "no_show | cancelled | confirmed" }
```

Response `200`: Updated booking object.

---

## Adding a new endpoint

Copy this block and fill in all fields:

```markdown
### [METHOD] /[path]
Auth: Required | Public | Admin only
Purpose: [one sentence]

Request:
```json
{}
```

Response `[status]`:
```json
{}
```

Error codes:
| Code | HTTP | When |
|---|---|---|
| `` |  |  |
```