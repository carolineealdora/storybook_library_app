package models

import "time"

const (
	AccountParent = "parent"
	AccountAdmin  = "admin"

	BookDraft     = "draft"
	BookPublished = "published"

	AgeRange = "range"
	AgePlus  = "plus"

	// Book access for the current parent (PRD §12.3 GET /books).
	AccessFree       = "free"
	AccessOwned      = "owned"
	AccessSubscribed = "subscribed"
	AccessLocked     = "locked"

	OrderPending = "pending"
	OrderPaid    = "paid"
	OrderFailed  = "failed"

	SubscriptionActive    = "active"
	SubscriptionCancelled = "cancelled"
	SubscriptionExpired   = "expired"

	OrderTypeNew     = "new"
	OrderTypeRenewal = "renewal"
)

type (
	Credential struct {
		ID           string
		Email        string
		PasswordHash string
		AccountType  string
		UserID       *string
		AdminID      *string
	}

	RefreshToken struct {
		ID           string
		CredentialID string
		ExpiresAt    time.Time
		RevokedAt    *time.Time
		AccountType  string
		UserID       *string
		AdminID      *string
	}

	User struct {
		ID                  string
		Name                string
		Email               string
		PasswordHash        string
		PinHash             *string
		PinLockedUntil      *time.Time
		PasswordLockedUntil *time.Time
	}

	Book struct {
		ID            string     `json:"id"`
		Title         string     `json:"title"`
		Author        string     `json:"author"`
		Description   string     `json:"description"`
		AgeType       string     `json:"age_type"`
		AgeMin        int        `json:"age_min"`
		AgeMax        *int       `json:"age_max"`
		AgeLabel      string     `json:"age_label" gorm:"-"`
		Price         int        `json:"price"`
		Status        string     `json:"status"`
		Access        string     `json:"access,omitempty" gorm:"-"`
		EpubFileName  string     `json:"epub_file_name"`
		EpubSizeBytes int64      `json:"epub_size_bytes"`
		Owned         bool       `json:"-"`
		PurchasedAt   *time.Time `json:"purchased_at,omitempty"`
		CreatedAt     time.Time  `json:"created_at"`
		UpdatedAt     time.Time  `json:"updated_at"`
		DeletedAt     *time.Time `json:"-"`
	}

	// BookFile is a cover image or an EPUB, streamed straight from the DB.
	BookFile struct {
		MimeType string
		FileName string
		Content  []byte
	}

	BookOrder struct {
		ID        string     `json:"id"`
		BookID    string     `json:"book_id"`
		BookTitle string     `json:"book_title"`
		Price     int        `json:"price"`
		Status    string     `json:"status"`
		CreatedAt time.Time  `json:"created_at"`
		PaidAt    *time.Time `json:"paid_at"`
	}

	SubscriptionPlan struct {
		ID        string    `json:"id"`
		Price     int       `json:"price"`
		Benefits  string    `json:"benefits"`
		Duration  string    `json:"duration"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	SubscriptionOrder struct {
		ID             string     `json:"id"`
		PlanID         string     `json:"plan_id"`
		SubscriptionID *string    `json:"subscription_id"`
		OrderType      string     `json:"order_type"`
		Price          int        `json:"price"`
		Status         string     `json:"status"`
		CreatedAt      time.Time  `json:"created_at"`
		PaidAt         *time.Time `json:"paid_at"`
	}

	Subscription struct {
		ID            string     `json:"id"`
		UserID        string     `json:"-"`
		PlanID        string     `json:"plan_id"`
		RenewedFromID *string    `json:"renewed_from_id"`
		PricePaid     int        `json:"price_paid"`
		Status        string     `json:"status"`
		StartedAt     time.Time  `json:"started_at"`
		ExpiresAt     time.Time  `json:"expires_at"`
		CancelledAt   *time.Time `json:"cancelled_at"`
		OrderType     *string    `json:"order_type,omitempty"`
	}
)
