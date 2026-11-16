package helpers

import (
	respConst "codebase/internal/constant/response"
	"codebase/models"
	"codebase/models/dto"

	"github.com/labstack/echo/v4"
)

// Base is embedded by controllers: request binding + the common response writer.
type Base struct {
	Response Response
	Binding  Binding
	Errors   Error
}

func NewBase(holder *models.SharedHolder) Base {
	return Base{Response: NewResponse(holder), Binding: NewBinding(), Errors: NewError(holder)}
}

// Bind fills req from path/query/body and validates it; failures are VALIDATION_ERROR.
func (b Base) Bind(ec echo.Context, req interface{ Validate() interface{} }) error {
	if err := b.Binding.Bind(CustomBinder{Request: req, Context: ec}); err != nil {
		return b.Errors.Wrap(respConst.VALIDATION_ERROR, err.Error())
	}
	return nil
}

// Page binds ?page=&page_size= (defaults applied).
func (b Base) Page(ec echo.Context) (dto.PageQuery, error) {
	var p dto.PageQuery
	err := b.Bind(ec, &p)
	return p.Normalize(), err
}

// RespondPage writes {"data": rows, "meta": {total, page, page_size}}.
func (b Base) RespondPage(ec echo.Context, rows any, total int64, p dto.PageQuery, err error) error {
	meta := dto.PageMeta{Total: total, Page: p.Page, PageSize: p.PageSize}
	return b.Response.CommonResponse(models.ResponseArgs{Ctx: ec, RespData: rows, Meta: meta, ErrorData: err})
}

// Respond writes {"data": data}, or the error body when err != nil.
func (b Base) Respond(ec echo.Context, data any, err error) error {
	return b.Response.CommonResponse(models.ResponseArgs{Ctx: ec, RespData: data, ErrorData: err})
}
