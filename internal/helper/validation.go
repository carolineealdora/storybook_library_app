package helpers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	respConst "codebase/internal/constant/response"
	"codebase/models"

	"github.com/labstack/echo/v4"
)

type (
	ValidationImpl struct {
		ctx    echo.Context
		errors Error
	}

	Validation interface {
		ValidateHeader() error
		ValidateGetBody() error
		ValidateRequestedURL() error
	}
)

func NewValidation(c echo.Context, shareHolder *models.SharedHolder) *ValidationImpl {
	errors := NewError(shareHolder)
	return &ValidationImpl{
		ctx:    c,
		errors: errors,
	}
}

func (v *ValidationImpl) ValidateHeader() error {
	var requiredHeaders []string

	requiredHeaders = []string{"x-partner-id", "x-request-id"}
	urlPath := v.ctx.Path()
	if WhitelistPartnerHeader(urlPath) {
		requiredHeaders = append(requiredHeaders[:0], requiredHeaders[1:]...)
	}

	// get and validate header value
	for _, header := range requiredHeaders {
		headerValue := v.ctx.Request().Header.Get(header)
		if len(strings.TrimSpace(headerValue)) == 0 {
			return errors.New("Header " + "'" + header + "'" + " is required")
		}

		if header == "x-partner-id" {
			if _, err := strconv.Atoi(headerValue); err != nil {
				return errors.New("Header " + "'" + header + "'" + " is invalid")
			}
		}
	}

	return nil
}

func (v *ValidationImpl) ValidateGetBody() error {
	method := v.ctx.Request().Method
	body := v.ctx.Request().Body

	// validate get method (not should have request body)
	if method == "GET" && fmt.Sprintf("%v", body) != "{}" {
		return v.errors.Wrap(respConst.INVALID_REQUEST_METHOD,
			"ValidateGetBody failed: invalid request method")
	}

	return nil
}

func (v *ValidationImpl) ValidateRequestedURL() error {
	path := v.ctx.Path()
	if path == "" {
		return v.errors.Wrap(respConst.PATH_NOT_FOUND,
			"ValidateRequestedURL failed: path not found")
	}

	return nil
}
