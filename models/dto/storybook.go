package dto

// Request bodies. Validate() returns the struct itself — that is the contract of helpers.Binding.
type (
	LoginRequest struct {
		Email    string `json:"email" validate:"required,email,max=255"`
		Password string `json:"password" validate:"required,max=72"`
	}

	RefreshTokenRequest struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	UpdateMeRequest struct {
		Name string `json:"name" validate:"required,max=100"`
	}

	SetPinRequest struct {
		Pin             string `json:"pin" validate:"required,matches=^[0-9]{4}$"`
		PinConfirmation string `json:"pin_confirmation" validate:"required,eqfield=Pin"`
		Password        string `json:"password" validate:"max=72"`
	}

	VerifyPinRequest struct {
		Pin string `json:"pin" validate:"required,max=4"`
	}

	PasswordRequest struct {
		Password string `json:"password" validate:"required,max=72"`
	}

	IDParam struct {
		ID string `param:"id" validate:"required,uuid"`
	}

	CreateBookOrderRequest struct {
		BookID         string `json:"book_id" validate:"required,uuid"`
		IdempotencyKey string `json:"idempotency_key" validate:"required,max=100"`
	}

	// ExpectedPrice is the price the paywall showed; a mismatch is 409 PRICE_CHANGED (CR-001 E-CR1).
	CreateSubscriptionOrderRequest struct {
		IdempotencyKey string `json:"idempotency_key" validate:"required,max=100"`
		ExpectedPrice  int    `json:"expected_price" validate:"required"`
	}

	// PayRequest is the mocked payment: the client picks the outcome (PRD: success / failed only).
	PayRequest struct {
		ID     string `param:"id" validate:"required,uuid"`
		Result string `json:"result" validate:"required,oneof=success failed"`
	}

	// PageQuery is ?page=&page_size= on list endpoints (TD-003). 0 = default.
	PageQuery struct {
		Page     int `query:"page" validate:"gte=0"`
		PageSize int `query:"page_size" validate:"gte=0,lte=100"`
	}

	// PageMeta is the "meta" next to "data" on list responses.
	PageMeta struct {
		Total    int64 `json:"total"`
		Page     int   `json:"page"`
		PageSize int   `json:"page_size"`
	}

	// AdminBookListQuery repeats the page fields: the binder only reads tagged top-level fields.
	AdminBookListQuery struct {
		Search   string `query:"search" validate:"max=255"`
		Status   string `query:"status" validate:"omitempty,oneof=draft published"`
		Page     int    `query:"page" validate:"gte=0"`
		PageSize int    `query:"page_size" validate:"gte=0,lte=100"`
	}

	// Price is a pointer so "missing" (VALIDATION_ERROR) differs from "<= 0" (INVALID_PRICE, CR-001 E-CR2).
	UpdatePlanRequest struct {
		Price    *int   `json:"price" validate:"required"`
		Benefits string `json:"benefits" validate:"required,max=5000"`
	}
)

func (r LoginRequest) Validate() interface{}                   { return r }
func (r RefreshTokenRequest) Validate() interface{}            { return r }
func (r UpdateMeRequest) Validate() interface{}                { return r }
func (r SetPinRequest) Validate() interface{}                  { return r }
func (r VerifyPinRequest) Validate() interface{}               { return r }
func (r PasswordRequest) Validate() interface{}                { return r }
func (r IDParam) Validate() interface{}                        { return r }
func (r CreateBookOrderRequest) Validate() interface{}         { return r }
func (r CreateSubscriptionOrderRequest) Validate() interface{} { return r }
func (r PayRequest) Validate() interface{}                     { return r }
func (r AdminBookListQuery) Validate() interface{}             { return r }
func (r PageQuery) Validate() interface{}                      { return r }

const DefaultPageSize = 20

// Normalize fills the defaults: page 1, 20 per page.
func (p PageQuery) Normalize() PageQuery {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PageSize == 0 {
		p.PageSize = DefaultPageSize
	}
	return p
}

func (p PageQuery) Offset() int                   { return (p.Page - 1) * p.PageSize }
func (r UpdatePlanRequest) Validate() interface{} { return r }
