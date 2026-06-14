package payment

import (
	respConst "codebase/internal/constant/response"
	helpers "codebase/internal/helper"
	"codebase/models"
	"codebase/models/dto"

	"github.com/labstack/echo/v4"
)

func (c *Controller) GetByRefId(ec echo.Context) error {
	var (
		param     dto.GetByRefIdParam
		partnerId = ec.Request().Header.Get("x-partner-id")
	)

	if err := c.Binding.Bind(helpers.CustomBinder{
		Request: &param,
		Context: ec,
	}); err != nil {
		return c.Response.CommonResponse(models.ResponseArgs{
			Ctx:       ec,
			ErrorData: c.Errors.Wrap(respConst.MANDATORY_FIELD, err.Error()),
		})
	}

	payment, err := c.PaymentService.GetByRefId(param.RefId, partnerId)
	if err != nil {
		return c.Response.CommonResponse(models.ResponseArgs{
			Ctx:       ec,
			ErrorData: err,
		})
	}

	return c.Response.CommonResponse(models.ResponseArgs{
		Ctx:      ec,
		RespData: dto.ToPaymentResponse(*payment),
	})
}
