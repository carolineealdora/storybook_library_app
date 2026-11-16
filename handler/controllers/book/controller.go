package book

import (
	"errors"
	"io"
	"mime"
	"net/http"

	"codebase/internal/constant/key"
	respConst "codebase/internal/constant/response"
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

func (c *Controller) AdminList(ec echo.Context) error {
	var q dto.AdminBookListQuery
	if err := c.Bind(ec, &q); err != nil {
		return c.Respond(ec, nil, err)
	}
	p := dto.PageQuery{Page: q.Page, PageSize: q.PageSize}.Normalize()
	books, total, err := c.Service.AdminList(q.Search, q.Status, p.PageSize, p.Offset())
	return c.RespondPage(ec, books, total, p, err)
}

func (c *Controller) AdminGet(ec echo.Context) error {
	return c.withID(ec, func(id string) error {
		b, err := c.Service.AdminGet(id)
		return c.Respond(ec, b, err)
	})
}

func (c *Controller) AdminCover(ec echo.Context) error {
	return c.withID(ec, func(id string) error {
		f, err := c.Service.AdminCover(id)
		return c.file(ec, f, err, "private, max-age=60")
	})
}

// AdminCreate takes multipart/form-data: title, author, description, age_type, age_min,
// age_max, price, status + files cover and epub.
func (c *Controller) AdminCreate(ec echo.Context) error {
	in, err := c.form(ec)
	if err != nil {
		return c.Respond(ec, nil, err)
	}
	b, err := c.Service.Create(in)
	return c.Respond(ec, b, err)
}

// AdminUpdate is the same form as create; cover and epub are optional (omitted = keep current).
func (c *Controller) AdminUpdate(ec echo.Context) error {
	return c.withID(ec, func(id string) error {
		in, err := c.form(ec)
		if err != nil {
			return c.Respond(ec, nil, err)
		}
		b, err := c.Service.Update(id, in)
		return c.Respond(ec, b, err)
	})
}

func (c *Controller) AdminDelete(ec echo.Context) error {
	return c.withID(ec, func(id string) error {
		return c.Respond(ec, map[string]bool{"deleted": true}, c.Service.Delete(id))
	})
}

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

func (c *Controller) form(ec echo.Context) (book.Input, error) {
	in := book.Input{
		Title:       ec.FormValue("title"),
		Author:      ec.FormValue("author"),
		Description: ec.FormValue("description"),
		AgeType:     ec.FormValue("age_type"),
		AgeMin:      ec.FormValue("age_min"),
		AgeMax:      ec.FormValue("age_max"),
		Price:       ec.FormValue("price"),
		Status:      ec.FormValue("status"),
	}
	var err error
	if in.Cover, err = upload(ec, "cover"); err != nil {
		return in, c.Errors.Wrap(respConst.VALIDATION_ERROR, "cover: "+err.Error())
	}
	if in.Epub, err = upload(ec, "epub"); err != nil {
		return in, c.Errors.Wrap(respConst.VALIDATION_ERROR, "epub: "+err.Error())
	}
	return in, nil
}

// upload reads one multipart file; nil when the field is absent.
func upload(ec echo.Context, field string) (*book.Upload, error) {
	fh, err := ec.FormFile(field)
	if errors.Is(err, http.ErrMissingFile) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	content, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return &book.Upload{FileName: fh.Filename, Content: content}, nil
}
