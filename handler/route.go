package handler

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"codebase/handler/controllers/healthcheck"
	"codebase/handler/controllers/payment"
	"codebase/internal/constant/key"
	respConst "codebase/internal/constant/response"
	helpers "codebase/internal/helper"
	"codebase/models"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"go.elastic.co/apm/module/apmechov4/v2"
)

func InitRoute(sharedHolder *models.SharedHolder) {
	sharedHolder.Echo.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestId := c.Request().Header.Get("x-request-id")
			url := c.Request().URL.String()
			method := c.Request().Method

			// set request id on ctx
			ctx := c.Request().Context()
			ctx = context.WithValue(ctx, key.RequestId, requestId)
			c.SetRequest(c.Request().WithContext(ctx))

			// init handlers
			response := helpers.NewResponse(sharedHolder)
			validation := helpers.NewValidation(c, sharedHolder)
			errors := helpers.NewError(sharedHolder)

			// validate path
			if err := validation.ValidateRequestedURL(); err != nil {
				return response.CommonResponse(models.ResponseArgs{
					Ctx:       c,
					ErrorData: err,
				})
			}

			// validate method
			if err := validation.ValidateGetBody(); err != nil {
				return response.CommonResponse(models.ResponseArgs{
					Ctx:       c,
					ErrorData: err,
				})
			}

			// validate headers
			if err := validation.ValidateHeader(); err != nil {
				return response.CommonResponse(models.ResponseArgs{
					Ctx:       c,
					ErrorData: errors.Wrap(respConst.INVALID_REQUEST_HEADER, err.Error()),
				})
			}

			// restricting requests based on x-request-id data
			key := fmt.Sprintf("%s-%s-%s", requestId, url, method)
			data := sharedHolder.Redis.Exists(context.Background(), key)
			if data.Val() > 0 {
				return response.CommonResponse(models.ResponseArgs{
					Ctx: c,
					ErrorData: errors.Wrap(respConst.CAN_NOT_ACCESS_API,
						"Redis failed: redis key already exists"),
				})
			}
			sharedHolder.Redis.Set(context.Background(), key, "", -1)

			// process the request (call the next handler)
			err := next(c)

			_ = sharedHolder.Redis.Del(context.Background(), key)

			return err
		}
	})

	//init controller here
	healthcheckCtrl := healthcheck.NewController(sharedHolder)
	paymentCtrl := payment.NewController(sharedHolder)

	sharedHolder.Echo.Use(apmechov4.Middleware())
	sharedHolder.Echo.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "[${time_rfc3339}] | ${status} | ${method} ${path} | ${latency_human} | ip=${remote_ip} | request_id=${custom}\n",
		Output: sharedHolder.Echo.Logger.Output(),
		CustomTagFunc: func(c echo.Context, buf *bytes.Buffer) (int, error) {
			reqId := "-"
			if reqIdData := c.Request().Context().Value(key.RequestId); reqIdData != nil {
				reqId = reqIdData.(string)
			}

			return buf.Write([]byte(reqId))
		},
	}))

	// loging middleware
	sharedHolder.Echo.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
		respBodyDump := make(chan error)
		customMiddleware := helpers.NewMiddleware(sharedHolder)

		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			respBodyDump <- customMiddleware.BodyDumpHandler(c, reqBody, resBody)
		}()
		go func() {
			wg.Wait()
			close(respBodyDump)
		}()
	}))

	//insert route here
	v1 := sharedHolder.Echo.Group("/api/v1")

	healthcheckGroup := v1.Group("/healthcheck")
	healthcheckGroup.GET("", healthcheckCtrl.GetHealthcheck)

	paymentGroup := v1.Group("/payment")
	paymentGroup.GET("/ref/:refId", paymentCtrl.GetByRefId)

	err := sharedHolder.Echo.Start("0.0.0.0:" + sharedHolder.Config.Port)
	if err != nil {
		sharedHolder.Log.Fatal(errors.Wrap(err, "failed to start echo"))
		return
	}
}
