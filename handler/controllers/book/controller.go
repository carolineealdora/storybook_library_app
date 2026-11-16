package book

import (
	"mime"
	"net/http"

	"codebase/internal/constant/key"
	helpers "codebase/internal/helper"
	"codebase/internal/services/book"
	"codebase/models"
	"codebase/models/dto"

	"github.com/labstack/echo/v4"
)

type Controller struct {
	helpers.Base
	Service *book.Service
}

func NewController(holder *models.SharedHolder) Controller {
	return Controller{Base: helpers.NewBase(holder), Service: book.NewService(holder)}
}

// ---- Library app (parent) ----

func (c *Controller) List(ec echo.Context) error {
	p, err := c.Page(ec)
	if err != nil {
		return c.Respond(ec, nil, err)
	}
	rows, total, err := c.Service.List(ec.Get(key.AccountID).(string), p.PageSize, p.Offset())
	return c.RespondPage(ec, rows, total, p, err)
}

func (c *Controller) MyBooks(ec echo.Context) error {
	p, err := c.Page(ec)
	if err != nil {
		return c.Respond(ec, nil, err)
	}
	rows, total, err := c.Service.MyBooks(ec.Get(key.AccountID).(string), p.PageSize, p.Offset())
	return c.RespondPage(ec, rows, total, p, err)
}

func (c *Controller) Detail(ec echo.Context) error {
	return c.withID(ec, func(id string) error {
		b, err := c.Service.Detail(ec.Get(key.AccountID).(string), id)
		return c.Respond(ec, b, err)
	})
}

func (c *Controller) Cover(ec echo.Context) error {
	return c.withID(ec, func(id string) error {
		f, err := c.Service.Cover(ec.Get(key.AccountID).(string), id)
		return c.file(ec, f, err, "private, max-age=300")
	})
}

// Content streams the EPUB. Locked books get 403 BOOK_LOCKED and no bytes.
func (c *Controller) Content(ec echo.Context) error {
	return c.withID(ec, func(id string) error {
		f, err := c.Service.Content(ec.Get(key.AccountID).(string), id)
		return c.file(ec, f, err, "no-store")
	})
}

// ---- Book Management app (admin) ----

func (c *Controller) withID(ec echo.Context, fn func(id string) error) error {
	var p dto.IDParam
	if err := c.Bind(ec, &p); err != nil {
		return c.Respond(ec, nil, err)
	}
	return fn(p.ID)
}

func (c *Controller) file(ec echo.Context, f *models.BookFile, err error, cacheControl string) error {
	if err != nil {
		return c.Respond(ec, nil, err)
	}
	ec.Response().Header().Set(echo.HeaderCacheControl, cacheControl)
	ec.Response().Header().Set(echo.HeaderContentDisposition, mime.FormatMediaType("inline", map[string]string{"filename": f.FileName}))
	return ec.Blob(http.StatusOK, f.MimeType, f.Content)
}
