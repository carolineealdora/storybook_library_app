package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"

	"github.com/go-playground/validator/v10"
)

func init() {
	_ = validate.RegisterValidation("matches", func(fl validator.FieldLevel) bool {
		regexPattern := fl.Param()
		regex := regexp.MustCompile(regexPattern)
		return regex.MatchString(fl.Field().String())
	})
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
