package models

import "github.com/labstack/echo/v4"

type BaseResponses struct {
	ResponseCode string      `json:"responseCode"`
	ResponseDesc string      `json:"responseDesc"`
	ResponseData interface{} `json:"responseData"`
}

type ResponsePayload struct {
	Code     string `json:"response_code"`
	Desc     string `json:"response_desc"`
	HttpCode int
}

type ResponseArgs struct {
	Ctx       echo.Context
	RespData  interface{}
	ErrorData error
}
