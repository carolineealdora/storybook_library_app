package subscription

import (
	"codebase/internal/constant/key"
	helpers "codebase/internal/helper"
	"codebase/internal/services/subscription"
	"codebase/models"
	"codebase/models/dto"

	"github.com/labstack/echo/v4"
)

type Controller struct {
	helpers.Base
	Service *subscription.Service
}

func NewController(holder *models.SharedHolder) Controller {
	return Controller{Base: helpers.NewBase(holder), Service: subscription.NewService(holder)}
}

// Plan serves both GET /subscription/plan (parent) and GET /admin/subscription-plan.
func (c *Controller) Plan(ec echo.Context) error {
	p, err := c.Service.Plan()
	return c.Respond(ec, p, err)
}

func (c *Controller) UpdatePlan(ec echo.Context) error {
	var req dto.UpdatePlanRequest
	if err := c.Bind(ec, &req); err != nil {
		return c.Respond(ec, nil, err)
	}
	p, err := c.Service.UpdatePlan(req.Price, req.Benefits)
	return c.Respond(ec, p, err)
}

func (c *Controller) CreateOrder(ec echo.Context) error {
	var req dto.CreateSubscriptionOrderRequest
	if err := c.Bind(ec, &req); err != nil {
		return c.Respond(ec, nil, err)
	}
	o, err := c.Service.CreateOrder(ec.Get(key.AccountID).(string), req.IdempotencyKey)
	return c.Respond(ec, o, err)
}

func (c *Controller) PayOrder(ec echo.Context) error {
	var req dto.PayRequest
	if err := c.Bind(ec, &req); err != nil {
		return c.Respond(ec, nil, err)
	}
	s, err := c.Service.PayOrder(ec.Get(key.AccountID).(string), req.ID, req.Result == "success")
	return c.Respond(ec, s, err)
}

func (c *Controller) Cancel(ec echo.Context) error {
	s, err := c.Service.Cancel(ec.Get(key.AccountID).(string))
	return c.Respond(ec, s, err)
}

func (c *Controller) History(ec echo.Context) error {
	p, err := c.Page(ec)
	if err != nil {
		return c.Respond(ec, nil, err)
	}
	rows, total, err := c.Service.History(ec.Get(key.AccountID).(string), p.PageSize, p.Offset())
	return c.RespondPage(ec, rows, total, p, err)
}
