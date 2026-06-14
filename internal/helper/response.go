package helpers

import (
	"codebase/internal/constant/key"
	respConst "codebase/internal/constant/response"
	"codebase/models"
)

type (
	ResponseImpl struct {
		sharedHolder *models.SharedHolder
	}

	Response interface {
		CommonResponse(args models.ResponseArgs) error
	}
)

func NewResponse(sharedHolder *models.SharedHolder) *ResponseImpl {
	return &ResponseImpl{
		sharedHolder: sharedHolder,
	}
}

func (r *ResponseImpl) CommonResponse(args models.ResponseArgs) error {
	var respInfo = respConst.Responses[respConst.SUCCESS]

	if args.ErrorData != nil {
		if e, ok := args.ErrorData.(*LogEntry); ok {
			errConst := e.ErrConst
			respInfo = respConst.Responses[errConst]
			if e.ErrWrap {
				respInfo.Desc = e.ErrInfo
			}

			r.sharedHolder.Log.WithFields(map[string]any{
				"request_id": args.Ctx.Request().Context().Value(key.RequestId),
				"error":      e.ErrInfo,
				"source": map[string]any{
					"component": e.Component,
					"src_file":  e.SrcFile,
					"src_func":  e.SrcFunc,
				},
			}).Error("onError")
		} else {
			respInfo = respConst.Responses[respConst.GENERAL_ERROR]
			r.sharedHolder.Log.WithFields(map[string]any{
				"request_id": args.Ctx.Request().Context().Value(key.RequestId),
				"error":      args.ErrorData.Error(),
			}).Error("onError")
		}
	}

	resp := models.BaseResponses{
		ResponseCode: respInfo.Code,
		ResponseDesc: respInfo.Desc,
		ResponseData: args.RespData,
	}

	return args.Ctx.JSON(respInfo.HttpCode, resp)
}
