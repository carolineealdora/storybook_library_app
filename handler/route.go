package handler

import (
	"errors"
	"net/http"
	"strings"

	"codebase/handler/controllers/account"
	"codebase/handler/controllers/auth"
	"codebase/handler/controllers/book"
	"codebase/handler/controllers/healthcheck"
	"codebase/handler/controllers/order"
	"codebase/handler/controllers/subscription"
	respConst "codebase/internal/constant/response"
	helpers "codebase/internal/helper"
	"codebase/models"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.elastic.co/apm/module/apmechov4/v2"
)

// InitRoute registers middleware and every endpoint of PRD §12.3 under /api/v1.
func InitRoute(sharedHolder *models.SharedHolder) {
	e := sharedHolder.Echo
	e.HTTPErrorHandler = errorHandler(sharedHolder)

	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(apmechov4.Middleware())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "[${time_rfc3339}] | ${status} | ${method} ${path} | ${latency_human} | ip=${remote_ip} | request_id=${id}\n",
		Output: e.Logger.Output(),
	}))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: sharedHolder.Config.CORSAllowOrigins,
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType, echo.HeaderXRequestID, auth.HeaderParentToken},
	}))

	//init controller here
	healthcheckCtrl := healthcheck.NewController(sharedHolder)
	authCtrl := auth.NewController(sharedHolder)
	accountCtrl := account.NewController(sharedHolder)
	bookCtrl := book.NewController(sharedHolder)
	orderCtrl := order.NewController(sharedHolder)
	subscriptionCtrl := subscription.NewController(sharedHolder)

	parent := authCtrl.Require(models.AccountParent)
	parentView := authCtrl.RequireParentMode // money and settings: not reachable from Child view
	admin := authCtrl.Require(models.AccountAdmin)
	uploadLimit := middleware.BodyLimit("60M") // 50 MB EPUB + 5 MB cover + form fields

	//insert route here
	v1 := e.Group("/api/v1")

	// PUBLIC
	v1.GET("/healthcheck", healthcheckCtrl.GetHealthcheck)
	v1.POST("/auth/login", authCtrl.Login)
	v1.POST("/auth/refresh", authCtrl.Refresh)
	v1.POST("/admin/auth/login", authCtrl.AdminLogin)

	// Digital Storybook Library app (parent)
	v1.POST("/auth/logout", authCtrl.Logout, parent)
	v1.GET("/me", accountCtrl.Me, parent)
	v1.PATCH("/me", accountCtrl.UpdateMe, parent, parentView)
	v1.PUT("/me/pin", accountCtrl.SetPin, parent, parentView)
	v1.POST("/me/pin/verify", accountCtrl.VerifyPin, parent)
	v1.POST("/me/pin/forgot", accountCtrl.ForgotPin, parent)
	v1.GET("/me/books", bookCtrl.MyBooks, parent)
	v1.GET("/books", bookCtrl.List, parent)
	v1.GET("/books/:id", bookCtrl.Detail, parent)
	v1.GET("/books/:id/cover", bookCtrl.Cover, parent)
	v1.GET("/books/:id/content", bookCtrl.Content, parent)
	v1.POST("/book-orders", orderCtrl.Create, parent, parentView)
	v1.POST("/book-orders/:id/pay", orderCtrl.Pay, parent, parentView)
	v1.GET("/book-orders", orderCtrl.History, parent)
	v1.GET("/subscription/plan", subscriptionCtrl.Plan, parent)
	v1.POST("/subscription-orders", subscriptionCtrl.CreateOrder, parent, parentView)
	v1.POST("/subscription-orders/:id/pay", subscriptionCtrl.PayOrder, parent, parentView)
	v1.POST("/subscriptions/current/cancel", subscriptionCtrl.Cancel, parent, parentView)
	v1.GET("/subscriptions", subscriptionCtrl.History, parent)

	// Book Management app (admin)
	v1.POST("/admin/auth/logout", authCtrl.Logout, admin)
	v1.GET("/admin/books", bookCtrl.AdminList, admin)
	v1.GET("/admin/books/:id", bookCtrl.AdminGet, admin)
	v1.GET("/admin/books/:id/cover", bookCtrl.AdminCover, admin)
	v1.POST("/admin/books", bookCtrl.AdminCreate, admin, uploadLimit)
	v1.PUT("/admin/books/:id", bookCtrl.AdminUpdate, admin, uploadLimit)
	v1.DELETE("/admin/books/:id", bookCtrl.AdminDelete, admin)
	v1.GET("/admin/subscription-plan", subscriptionCtrl.Plan, admin)
	v1.PUT("/admin/subscription-plan", subscriptionCtrl.UpdatePlan, admin)
}

// errorHandler gives router/framework errors (404, 405, 413, panics) the same
// {"error": {"code", "message"}} shape as the app's own errors.
func errorHandler(sharedHolder *models.SharedHolder) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		status := http.StatusInternalServerError
		body := helpers.ErrorBody{Code: respConst.GENERAL_ERROR, Message: respConst.Responses[respConst.GENERAL_ERROR].Message}
		var he *echo.HTTPError
		if errors.As(err, &he) && he.Code < 500 {
			status = he.Code
			text := http.StatusText(he.Code)
			body = helpers.ErrorBody{Code: strings.ToUpper(strings.ReplaceAll(text, " ", "_")), Message: text}
		} else {
			sharedHolder.Log.Error("unhandled error: ", err)
		}
		if err := c.JSON(status, map[string]any{"error": body}); err != nil {
			sharedHolder.Log.Error("write error response: ", err)
		}
	}
}
