package order

import (
	"codebase/internal/constant/key"
	helpers "codebase/internal/helper"
	"codebase/internal/services/order"
	"codebase/models"
	"codebase/models/dto"

	"github.com/labstack/echo/v4"
)

type Controller struct {
	helpers.Base
	Service *order.Service
}

func NewController(holder *models.SharedHolder) Controller {
	return Controller{Base: helpers.NewBase(holder), Service: order.NewService(holder)}
}

func (c *Controller) Create(ec echo.Context) error {
	var req dto.CreateBookOrderRequest
	if err := c.Bind(ec, &req); err != nil {
		return c.Respond(ec, nil, err)
	}
	o, err := c.Service.Create(ec.Get(key.AccountID).(string), req.BookID, req.IdempotencyKey)
	return c.Respond(ec, o, err)
}

func (c *Controller) Pay(ec echo.Context) error {
	var req dto.PayRequest
	if err := c.Bind(ec, &req); err != nil {
		return c.Respond(ec, nil, err)
	}
	o, err := c.Service.Pay(ec.Get(key.AccountID).(string), req.ID, req.Result == "success")
	return c.Respond(ec, o, err)
}

func (c *Controller) History(ec echo.Context) error {
	p, err := c.Page(ec)
	if err != nil {
		return c.Respond(ec, nil, err)
	}
	rows, total, err := c.Service.History(ec.Get(key.AccountID).(string), p.PageSize, p.Offset())
	return c.RespondPage(ec, rows, total, p, err)
}
