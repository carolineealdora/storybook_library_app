package auth

import (
	"strings"

	"codebase/internal/constant/key"
	respConst "codebase/internal/constant/response"
	helpers "codebase/internal/helper"
	"codebase/internal/services/auth"
	"codebase/models"
	"codebase/models/dto"

	"github.com/labstack/echo/v4"
)

type Controller struct {
	helpers.Base
	Service *auth.Service
}

func NewController(holder *models.SharedHolder) Controller {
	return Controller{Base: helpers.NewBase(holder), Service: auth.NewService(holder)}
}

// Require rejects requests without a valid access token for the given account type
// (401 UNAUTHORIZED / TOKEN_EXPIRED, 403 FORBIDDEN — PRD E17, E19, E20).
// SECURITY: manual review required.
func (c *Controller) Require(accountType string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ec echo.Context) error {
			token, ok := strings.CutPrefix(ec.Request().Header.Get(echo.HeaderAuthorization), "Bearer ")
			if !ok || token == "" {
				return c.Respond(ec, nil, c.Errors.Wrap(respConst.UNAUTHORIZED, "missing bearer token"))
			}
			claims, err := c.Service.ParseAccessToken(token)
			if err != nil {
				return c.Respond(ec, nil, err)
			}
			if claims.Role != accountType {
				return c.Respond(ec, nil, c.Errors.Wrap(respConst.FORBIDDEN, claims.Role+" token on "+accountType+" route"))
			}
			ec.Set(key.AccountID, claims.Subject)
			ec.Set(key.CredentialID, claims.CredentialID)
			return next(ec)
		}
	}
}

// HeaderParentToken carries the parent token (from login, PIN or Forgot PIN?).
const HeaderParentToken = "X-Parent-Token"

// RequireParentMode guards payment and settings routes: Child view has no parent token,
// so a child holding the parent's session still gets 403 PARENT_MODE_REQUIRED (TD-002).
// Must run after Require(models.AccountParent).
// SECURITY: manual review required.
func (c *Controller) RequireParentMode(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ec echo.Context) error {
		if err := c.Service.CheckParentToken(ec.Request().Header.Get(HeaderParentToken), ec.Get(key.AccountID).(string)); err != nil {
			return c.Respond(ec, nil, err)
		}
		return next(ec)
	}
}

// PUBLIC
func (c *Controller) Login(ec echo.Context) error { return c.login(ec, models.AccountParent) }

// PUBLIC
func (c *Controller) AdminLogin(ec echo.Context) error { return c.login(ec, models.AccountAdmin) }

// PUBLIC — authenticated by the refresh token itself.
func (c *Controller) Refresh(ec echo.Context) error {
	var req dto.RefreshTokenRequest
	if err := c.Bind(ec, &req); err != nil {
		return c.Respond(ec, nil, err)
	}
	tokens, err := c.Service.Refresh(req.RefreshToken)
	return c.Respond(ec, tokens, err)
}

func (c *Controller) Logout(ec echo.Context) error {
	var req dto.RefreshTokenRequest
	if err := c.Bind(ec, &req); err != nil {
		return c.Respond(ec, nil, err)
	}
	err := c.Service.Logout(req.RefreshToken, ec.Get(key.CredentialID).(string))
	return c.Respond(ec, map[string]bool{"logged_out": true}, err)
}

func (c *Controller) login(ec echo.Context, accountType string) error {
	var req dto.LoginRequest
	if err := c.Bind(ec, &req); err != nil {
		return c.Respond(ec, nil, err)
	}
	tokens, err := c.Service.Login(req.Email, req.Password, accountType)
	return c.Respond(ec, tokens, err)
}
