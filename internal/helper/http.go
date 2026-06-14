package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"codebase/internal/constant/key"
	"codebase/pkg/logger"
	"codebase/pkg/monitor"

	"github.com/go-resty/resty/v2"
)

type (
	Restful interface {
		GET(ctx context.Context, path string, header map[string]string, param *map[string]interface{}) (*BaseResponse, error)
		POST(ctx context.Context, path string, header map[string]string, body interface{}) (*BaseResponse, error)
		PUT(ctx context.Context, path string, header map[string]string, body interface{}) (*BaseResponse, error)
		PATCH(ctx context.Context, path string, header map[string]string, body interface{}) (*BaseResponse, error)
		DELETE(ctx context.Context, path string, header map[string]string, body interface{}) (*BaseResponse, error)

		POSTForm(ctx context.Context, path string, header map[string]string, body map[string]string) (*BaseResponse, error)

		POSTProxy(ctx context.Context, path string, pathProxy string, header map[string]string, body interface{}) (*BaseResponse, error)

		POSTFormBasicAuth(ctx context.Context, path string, header map[string]string, body map[string]string, username string, pass string) (*BaseResponse, error)

		SetBasicAuth(username, password string) Restful

		Monitor(ctx context.Context, monitor monitor.Monitor, requestId string, captureError bool) Restful
	}

	BaseResponse struct {
		Status int
		Header http.Header
		Body   []byte
	}
)

type (
	Client struct {
		logger       logger.Logger
		client       *resty.Client
		isMonitor    bool
		monitor      monitor.Monitor
		context      context.Context
		requestId    string
		captureError bool
	}

	RequestLog struct {
		URL          string       `json:"url,omitempty"`
		Method       string       `json:"method,omitempty"`
		Token        string       `json:"token,omitempty"`
		AuthScheme   string       `json:"auth_scheme,omitempty"`
		QueryParam   *url.Values  `json:"query_param,omitempty"`
		FormData     *url.Values  `json:"form_data,omitempty"`
		Header       *http.Header `json:"header,omitempty"`
		Time         time.Time    `json:"time,omitempty"`
		Body         interface{}  `json:"body,omitempty"`
		ResponseBody string       `json:"response_body,omitempty"`
	}
)

func NewHttpRestyClient(ctx context.Context, logger logger.Logger) Restful {
	client := resty.New()

	return &Client{
		logger: logger,
		client: client.
			SetLogger(logger).
			OnBeforeRequest(preRequest(ctx, logger)).
			OnAfterResponse(postResponse(ctx, logger)).
			OnError(onError(ctx, logger)),
	}
}

func preRequest(ctx context.Context, logger logger.Logger) resty.RequestMiddleware {
	return func(restyClient *resty.Client, request *resty.Request) error {
		var (
			requestID  = request.Context().Value(key.RequestId)
			requestLog = RequestLog{
				URL:        request.URL,
				Method:     request.Method,
				Token:      request.Token,
				AuthScheme: request.AuthScheme,
				Time:       request.Time,
			}
		)
		if len(request.QueryParam) > 0 {
			requestLog.QueryParam = &request.QueryParam
		}
		if len(request.FormData) > 0 {
			requestLog.FormData = &request.FormData
		}
		if len(request.Header) > 0 {
			requestLog.Header = &request.Header
		}
		if request.Body != nil {
			requestLog.Body = request.Body
		}

		logger.WithFields(map[string]any{
			"request_id": requestID,
			"request":    requestLog,
		}).Info("preRequest")
		return nil
	}
}

type JsonBody map[string]any

func postResponse(ctx context.Context, logger logger.Logger) resty.ResponseMiddleware {
	return func(restyClient *resty.Client, r *resty.Response) error {
		var (
			requestID  = r.Request.Context().Value(key.RequestId)
			request    = r.Request
			requestLog = RequestLog{
				URL:        request.URL,
				Method:     request.Method,
				Token:      request.Token,
				AuthScheme: request.AuthScheme,
				Time:       request.Time,
			}
		)
		if len(request.QueryParam) > 0 {
			requestLog.QueryParam = &request.QueryParam
		}
		if len(request.FormData) > 0 {
			requestLog.FormData = &request.FormData
		}
		if len(request.Header) > 0 {
			requestLog.Header = &request.Header
		}
		if request.Body != nil {
			requestLog.Body = request.Body
		}

		responseLog := JsonBody{}
		json.Unmarshal(r.Body(), &responseLog)
		logger.WithFields(map[string]any{
			"request_id": requestID,
			"request":    requestLog,
			"response":   responseLog,
		}).Info("postResponse")
		return nil
	}
}

func onError(ctx context.Context, logger logger.Logger) resty.ErrorHook {
	return func(request *resty.Request, err error) {
		var (
			requestID  = request.Context().Value(key.RequestId)
			requestLog = RequestLog{
				URL:        request.URL,
				Method:     request.Method,
				Token:      request.Token,
				AuthScheme: request.AuthScheme,
				Time:       request.Time,
			}
		)
		if len(request.QueryParam) > 0 {
			requestLog.QueryParam = &request.QueryParam
		}
		if len(request.FormData) > 0 {
			requestLog.FormData = &request.FormData
		}
		if len(request.Header) > 0 {
			requestLog.Header = &request.Header
		}
		if request.Body != nil {
			requestLog.Body = request.Body
		}

		logger.WithFields(map[string]any{
			"request_id": requestID,
			"request":    requestLog,
			"error":      err.Error(),
		}).Error("onError")
	}
}

func (c *Client) GET(ctx context.Context, path string, header map[string]string, param *map[string]interface{}) (*BaseResponse, error) {
	tr := c.startMonitor("GET", path)

	if param != nil && len(*param) > 0 {
		template := "%s=%s"
		start := true
		for key, value := range *param {
			if start {
				path += "?"
				start = false
			} else {
				path += "&"
			}
			path += fmt.Sprintf(template, key, value)
		}
	}

	request := c.client.R().SetHeaders(header).SetContext(ctx)
	response, err := request.Get(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) POST(ctx context.Context, path string, header map[string]string, body interface{}) (*BaseResponse, error) {
	tr := c.startMonitor("POST", path)

	request := c.client.R().SetHeaders(header).SetBody(body).SetContext(ctx)
	response, err := request.Post(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) POSTProxy(ctx context.Context, path string, pathPRoxy string, header map[string]string, body interface{}) (*BaseResponse, error) {
	tr := c.startMonitor("POST", path)

	request := c.client.R().SetHeaders(header).SetBody(body).SetContext(ctx)
	response, err := request.Post(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) POSTForm(ctx context.Context, path string, header map[string]string, body map[string]string) (*BaseResponse, error) {
	tr := c.startMonitor("POST", path)

	request := c.client.R().SetHeaders(header).SetFormData(body).SetContext(ctx)
	response, err := request.Post(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) POSTFormBasicAuth(ctx context.Context, path string, header map[string]string, body map[string]string, username string, pass string) (*BaseResponse, error) {
	tr := c.startMonitor("POST", path)

	request := c.client.R().SetBasicAuth(username, pass).SetHeaders(header).SetFormData(body).SetContext(ctx)
	response, err := request.Post(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) PUT(ctx context.Context, path string, header map[string]string, body interface{}) (*BaseResponse, error) {
	tr := c.startMonitor("PUT", path)

	request := c.client.R().SetHeaders(header).SetBody(body).SetContext(ctx)
	response, err := request.Put(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) PATCH(ctx context.Context, path string, header map[string]string, body interface{}) (*BaseResponse, error) {
	tr := c.startMonitor("PATCH", path)

	request := c.client.R().SetHeaders(header).SetBody(body).SetContext(ctx)
	response, err := request.Patch(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) DELETE(ctx context.Context, path string, header map[string]string, body interface{}) (*BaseResponse, error) {
	tr := c.startMonitor("DELETE", path)
	request := c.client.R().SetHeaders(header).SetBody(body).SetContext(ctx)

	response, err := request.Delete(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) SetBasicAuth(username, password string) Restful {
	c.client.R().SetBasicAuth(username, password)
	return &Client{
		logger:       c.logger,
		client:       c.client,
		isMonitor:    true,
		monitor:      c.monitor,
		context:      c.context,
		requestId:    c.requestId,
		captureError: c.captureError,
	}
}

func (c *Client) Monitor(ctx context.Context, mntr monitor.Monitor, requestId string, captureError bool) Restful {
	return &Client{
		logger:       c.logger,
		client:       c.client,
		isMonitor:    true,
		monitor:      mntr,
		context:      ctx,
		requestId:    requestId,
		captureError: captureError,
	}
}

func (c *Client) startMonitor(method, url string) monitor.Transaction {
	if c.isMonitor {
		name := fmt.Sprintf("%s %s", method, url)
		return c.monitor.NewTransactionFromContext(c.context, monitor.Tick{
			Operation:       "http",
			TransactionName: name,
			Tags: []monitor.Tag{
				{"requestId", c.requestId},
				{"action", name},
				{"method", method},
			},
		})
	}
	return nil
}

func (c *Client) finishMonitor(transaction monitor.Transaction, response *resty.Response, err error) {
	var (
		statusCode = response.StatusCode()
		status     = response.Status()
	)
	if c.isMonitor {
		if statusCode == 0 {
			statusCode = 500
			status = "InternalError"
		}

		transaction.FinishWithTags([]monitor.Tag{
			{"code", fmt.Sprintf("%d", statusCode)},
			{"status", status},
		})
		if c.captureError {
			c.monitor.Capture(err)
		}
	}
}

func convertToBaseResponse(response *resty.Response) *BaseResponse {
	if response == nil {
		return nil
	}

	return &BaseResponse{
		Status: response.StatusCode(),
		Header: response.Header(),
		Body:   response.Body(),
	}
}
