package account

import (
	"codebase/internal/constant/key"
	helpers "codebase/internal/helper"
	"codebase/internal/services/account"
	"codebase/internal/services/auth"
	"codebase/models"
	"codebase/models/dto"

	"github.com/labstack/echo/v4"
)

// verified is the Child -> Parent view response: the parent token goes in X-Parent-Token.
type verified struct {
	Verified bool `json:"verified"`
	*auth.ParentToken
}

type Controller struct {
	helpers.Base
	Service *account.Service
}

func NewController(holder *models.SharedHolder) Controller {
	return Controller{Base: helpers.NewBase(holder), Service: account.NewService(holder)}
}

func (c *Controller) Me(ec echo.Context) error {
	me, err := c.Service.Me(ec.Get(key.AccountID).(string))
	return c.Respond(ec, me, err)
}

func (c *Controller) UpdateMe(ec echo.Context) error {
	var req dto.UpdateMeRequest
	if err := c.Bind(ec, &req); err != nil {
		return c.Respond(ec, nil, err)
	}
	me, err := c.Service.UpdateName(ec.Get(key.AccountID).(string), req.Name)
	return c.Respond(ec, me, err)
}

func (c *Controller) SetPin(ec echo.Context) error {
	var req dto.SetPinRequest
	if err := c.Bind(ec, &req); err != nil {
		return c.Respond(ec, nil, err)
	}
	err := c.Service.SetPin(ec.Get(key.AccountID).(string), req.Pin, req.Password)
	return c.Respond(ec, map[string]bool{"pin_set": true}, err)
}

func (c *Controller) VerifyPin(ec echo.Context) error {
	var req dto.VerifyPinRequest
	if err := c.Bind(ec, &req); err != nil {
		return c.Respond(ec, nil, err)
	}
	pt, err := c.Service.VerifyPin(ec.Get(key.AccountID).(string), req.Pin)
	return c.Respond(ec, verified{true, pt}, err)
}

func (c *Controller) ForgotPin(ec echo.Context) error {
	var req dto.PasswordRequest
	if err := c.Bind(ec, &req); err != nil {
		return c.Respond(ec, nil, err)
	}
	pt, err := c.Service.VerifyPassword(ec.Get(key.AccountID).(string), req.Password)
	return c.Respond(ec, verified{true, pt}, err)
}
