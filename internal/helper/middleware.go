package helpers

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"time"

	"codebase/models"
	"codebase/pkg/shared"

	"github.com/labstack/echo/v4"
)

type (
	Middleware struct {
		sharedHolder models.SharedHolder
		mongoQuery   Mongo
		now          shared.Now
	}

	ParamRecursive struct {
		field string
		value []byte
		data  map[string]interface{}
	}
)

func NewMiddleware(sharedHolder *models.SharedHolder) Middleware {
	mongo := NewMongo(sharedHolder)
	return Middleware{
		sharedHolder: *sharedHolder,
		mongoQuery:   mongo,
		now:          time.Now,
	}
}

func (m Middleware) BodyDumpHandler(c echo.Context, reqBody, resBody []byte) error {
	var (
		param                         = make(map[string]string)
		manageReqData, manageRespData = make(chan interface{}), make(chan interface{})
	)

	// validate method
	methods := strings.Split(m.sharedHolder.Config.LoggingAllowedMethod, ",")
	if !SlicesContains(methods, c.Request().Method) {
		return errors.New("method not allowed")
	}

	// restructure path variable
	paramNames := c.ParamNames()
	for _, name := range paramNames {
		paramValue := c.Param(name)
		param[name] = paramValue
	}

	// restructure value on image and doc field
	wg := &sync.WaitGroup{}
	wg.Add(2)
	// on request
	go func() {
		defer wg.Done()
		manageReqData <- RestructureRecursive(ParamRecursive{
			value: reqBody,
		})
	}()
	// on response
	go func() {
		defer wg.Done()
		manageRespData <- RestructureRecursive(ParamRecursive{
			value: resBody,
		})
	}()
	go func() {
		wg.Wait()
		close(manageReqData)
		close(manageRespData)
	}()

	// define data
	data := map[string]interface{}{
		"status": c.Response().Status,
		"method": c.Request().Method,
		"uri":    string(c.Path()),
		"request": map[string]interface{}{
			"header":        c.Request().Header,
			"body":          <-manageReqData,
			"query_param":   c.QueryParams(),
			"path_variable": param,
		},
		"response":  <-manageRespData,
		"timestamp": m.now(),
	}

	// insert to database
	if _, err := m.mongoQuery.InsertOne(data, context.TODO()); err != nil {
		return err
	}
	return nil
}

func RestructureRecursive(params ParamRecursive) interface{} {
	var (
		respData                   = make(map[string]interface{})
		imgDocFields               = []string{}
		rawArrObj, processedArrObj []interface{}
	)
	json.Unmarshal(params.value, &respData)
	if len(respData) > 0 {
		for respField, respValue := range respData {
			if strings.Contains(strings.ToLower(respField), "image") || strings.Contains(strings.ToLower(respField), "doc") {
				imgDocFields = append(imgDocFields, respField)
			}
			// check nested data recursively
			value, _ := json.Marshal(respValue)
			RestructureRecursive(ParamRecursive{
				field: respField,
				value: value,
				data:  respData,
			})
		}
		if len(imgDocFields) > 0 {
			for _, key := range imgDocFields {
				respData[key] = "image or doc file"
			}
		}
		if params.field != "" {
			params.data[params.field] = respData
		} else {
			params.data = respData
		}
		// assign array back to null
		imgDocFields = nil
	}

	// check array object recursively
	json.Unmarshal(params.value, &rawArrObj)
	if len(rawArrObj) > 0 {
		// provide sample data for assets with only object data types
		if reflect.DeepEqual(reflect.TypeOf(rawArrObj[0]), reflect.TypeOf(respData)) {
			for _, obj := range rawArrObj {
				value, _ := json.Marshal(obj)
				recursiveData := RestructureRecursive(ParamRecursive{
					value: value,
				})
				processedArrObj = append(processedArrObj, recursiveData)
			}
			if params.field != "" {
				params.data[params.field] = processedArrObj
			} else {
				// returning only array object
				return processedArrObj
			}
		}
	}

	return params.data
}
