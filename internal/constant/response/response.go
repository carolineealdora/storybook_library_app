package response

import (
	"net/http"

	"codebase/models"
)

// Error codes from PRD §8 / §12.4. The map key is the machine-readable `error.code`.
const (
	VALIDATION_ERROR        = "VALIDATION_ERROR"
	INVALID_FILE_FORMAT     = "INVALID_FILE_FORMAT"
	INVALID_AGE_RANGE       = "INVALID_AGE_RANGE"
	INVALID_CREDENTIALS     = "INVALID_CREDENTIALS"
	UNAUTHORIZED            = "UNAUTHORIZED"
	TOKEN_EXPIRED           = "TOKEN_EXPIRED"
	REFRESH_TOKEN_EXPIRED   = "REFRESH_TOKEN_EXPIRED"
	INVALID_PIN             = "INVALID_PIN"
	INVALID_PASSWORD        = "INVALID_PASSWORD"
	PAYMENT_FAILED          = "PAYMENT_FAILED"
	FORBIDDEN               = "FORBIDDEN"
	BOOK_LOCKED             = "BOOK_LOCKED"
	PARENT_MODE_REQUIRED    = "PARENT_MODE_REQUIRED"
	BOOK_NOT_FOUND          = "BOOK_NOT_FOUND"
	ORDER_NOT_FOUND         = "ORDER_NOT_FOUND"
	BOOK_ALREADY_OWNED      = "BOOK_ALREADY_OWNED"
	BOOK_UNAVAILABLE        = "BOOK_UNAVAILABLE"
	BOOK_NOT_PURCHASABLE    = "BOOK_NOT_PURCHASABLE"
	SUBSCRIPTION_ACTIVE     = "SUBSCRIPTION_ACTIVE"
	SUBSCRIPTION_NOT_ACTIVE = "SUBSCRIPTION_NOT_ACTIVE"
	DUPLICATE_REQUEST       = "DUPLICATE_REQUEST"
	PIN_NOT_SET             = "PIN_NOT_SET"
	TOO_MANY_ATTEMPTS       = "TOO_MANY_ATTEMPTS"
	GENERAL_ERROR           = "GENERAL_ERROR"
)

var Responses = map[string]models.ResponsePayload{
	VALIDATION_ERROR:    {Message: "Request validation failed.", HttpCode: http.StatusBadRequest},
	INVALID_FILE_FORMAT: {Message: "Book file must be .epub and cover must be .jpg, .jpeg or .png.", HttpCode: http.StatusBadRequest},
	INVALID_AGE_RANGE:   {Message: "Age must be whole numbers and min must be lower than max.", HttpCode: http.StatusBadRequest},

	INVALID_CREDENTIALS:   {Message: "Email or password is incorrect", HttpCode: http.StatusUnauthorized},
	UNAUTHORIZED:          {Message: "Please log in again.", HttpCode: http.StatusUnauthorized},
	TOKEN_EXPIRED:         {Message: "Access token has expired.", HttpCode: http.StatusUnauthorized},
	REFRESH_TOKEN_EXPIRED: {Message: "Session has expired. Please log in again.", HttpCode: http.StatusUnauthorized},
	INVALID_PIN:           {Message: "PIN is incorrect.", HttpCode: http.StatusUnauthorized},
	INVALID_PASSWORD:      {Message: "Password is incorrect.", HttpCode: http.StatusUnauthorized},

	PAYMENT_FAILED: {Message: "Payment failed. Please try again.", HttpCode: http.StatusPaymentRequired},

	FORBIDDEN:            {Message: "You do not have access to this resource.", HttpCode: http.StatusForbidden},
	BOOK_LOCKED:          {Message: "Subscribe or purchase this book to read it.", HttpCode: http.StatusForbidden},
	PARENT_MODE_REQUIRED: {Message: "Switch to Parent view to continue.", HttpCode: http.StatusForbidden},

	BOOK_NOT_FOUND:  {Message: "Book not found.", HttpCode: http.StatusNotFound},
	ORDER_NOT_FOUND: {Message: "Order not found.", HttpCode: http.StatusNotFound},

	BOOK_ALREADY_OWNED:      {Message: "You already own this book.", HttpCode: http.StatusConflict},
	BOOK_UNAVAILABLE:        {Message: "This book is no longer available.", HttpCode: http.StatusConflict},
	BOOK_NOT_PURCHASABLE:    {Message: "This book is free and cannot be purchased.", HttpCode: http.StatusConflict},
	SUBSCRIPTION_ACTIVE:     {Message: "You already have an active subscription.", HttpCode: http.StatusConflict},
	SUBSCRIPTION_NOT_ACTIVE: {Message: "There is no active subscription.", HttpCode: http.StatusConflict},
	DUPLICATE_REQUEST:       {Message: "This request was already processed.", HttpCode: http.StatusConflict},
	PIN_NOT_SET:             {Message: "Please set a PIN in Settings first.", HttpCode: http.StatusConflict},

	TOO_MANY_ATTEMPTS: {Message: "Too many wrong attempts. Please try again in 1 minute.", HttpCode: http.StatusTooManyRequests},

	GENERAL_ERROR: {Message: "Something went wrong.", HttpCode: http.StatusInternalServerError},
}
