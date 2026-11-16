package helpers

import (
	respConst "codebase/internal/constant/response"
	"codebase/models"

	"github.com/labstack/echo/v4"
)

type (
	ResponseImpl struct {
		sharedHolder *models.SharedHolder
	}

	Response interface {
		CommonResponse(args models.ResponseArgs) error
	}

	ErrorBody struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
)

func NewResponse(sharedHolder *models.SharedHolder) *ResponseImpl {
	return &ResponseImpl{
		sharedHolder: sharedHolder,
	}
}

// CommonResponse writes `{"data": ...}` on success and the PRD §12.4 shape
// `{"error": {"code", "message"}}` on failure. Internal details are only logged.
func (r *ResponseImpl) CommonResponse(args models.ResponseArgs) error {
	if args.ErrorData == nil {
		body := map[string]any{"data": args.RespData}
		if args.Meta != nil {
			body["meta"] = args.Meta
		}
		return args.Ctx.JSON(200, body)
	}

	code, message := respConst.GENERAL_ERROR, ""
	fields := map[string]any{
		"request_id": args.Ctx.Response().Header().Get(echo.HeaderXRequestID),
		"error":      args.ErrorData.Error(),
	}
	if e, ok := args.ErrorData.(*LogEntry); ok {
		code = e.ErrConst
		if e.ErrWrap {
			message = e.ErrInfo
		}
		fields["source"] = map[string]any{"component": e.Component, "src_file": e.SrcFile, "src_func": e.SrcFunc}
	}

	payload, ok := respConst.Responses[code]
	if !ok {
		code, payload = respConst.GENERAL_ERROR, respConst.Responses[respConst.GENERAL_ERROR]
	}
	if message == "" {
		message = payload.Message
	}
	if payload.HttpCode >= 500 {
		r.sharedHolder.Log.WithFields(fields).Error("onError")
	} else {
		r.sharedHolder.Log.WithFields(fields).Info("onClientError")
	}

	return args.Ctx.JSON(payload.HttpCode, map[string]any{"error": ErrorBody{Code: code, Message: message}})
}
