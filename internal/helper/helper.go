package helpers

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/labstack/echo/v4"
)

type CustomHTTPError struct {
	// *echo.HTTPError
	Code      int         `json:"http_status_code"`
	Message   interface{} `json:"desc"`
	Internal  error       `json:"-"`
	RequestID string      `json:"service_request_id"`
	Data      interface{} `json:"data"`
	Body      interface{} `json:"body"`
}

func init() {
	_ = validate.RegisterValidation("matches", func(fl validator.FieldLevel) bool {
		regexPattern := fl.Param()
		regex := regexp.MustCompile(regexPattern)
		return regex.MatchString(fl.Field().String())
	})
}

func CustomHTTPErrorHandler(err error, c echo.Context) {
	report, ok := err.(*echo.HTTPError)
	if !ok {
		if c.Echo().Debug {
			report = echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		} else {
			report = echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error")
		}
	}

	UUID := c.Get("RequestID").(string)
	c.Logger().Error(report, UUID)
	c.JSON(report.Code, &CustomHTTPError{
		Code:      report.Code,
		Message:   report.Message,
		Internal:  report.Internal,
		RequestID: UUID,
	})
}

func GenerateRefID(n int) string {
	// Generate n random bytes
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}

	// Convert to a hexadecimal string and trim to 15 characters
	refID := fmt.Sprintf("%x", b)[:15]
	return refID
}

func SlicesContains(arrayData []string, str string) bool {
	for _, data := range arrayData {
		if data == str {
			return true
		}
	}
	return false
}

func initStruct(data []reflect.StructField) any {
	typ := reflect.StructOf(data)
	val := reflect.New(typ).Elem()
	obj := val.Addr().Interface()
	return obj
}

func WhitelistPartnerHeader(urlPath string) bool {
	whitelistPaths := []string{
		"/healthcheck",
	}
	for _, path := range whitelistPaths {
		if strings.Contains(urlPath, path) {
			return true
		}
	}
	return false
}

func handleBindError(err error) error {
	var unmarshalTypeError *json.UnmarshalTypeError
	if errors.As(err, &unmarshalTypeError) {
		msg := fmt.Sprintf("Field '%s' must be %s, but got %s",
			unmarshalTypeError.Field,
			unmarshalTypeError.Type.String(),
			unmarshalTypeError.Value,
		)
		return errors.New(msg)
	}
	return err
}
