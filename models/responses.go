package models

import "github.com/labstack/echo/v4"

type ResponsePayload struct {
	Message  string
	HttpCode int
}

type ResponseArgs struct {
	Ctx       echo.Context
	RespData  interface{}
	Meta      interface{} // list endpoints: page info, omitted when nil
	ErrorData error
}
