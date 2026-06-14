package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"

	helpers "codebase/internal/helper"
	"codebase/internal/services/healthcheck"
	"codebase/models"
	helper_mocks "codebase/test/mock/helper"
	services_mocks "codebase/test/mock/service"

	"github.com/labstack/echo/v4"
)

func Test_getStatus(t *testing.T) {
	type args struct {
		status bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test Case Positive - Return UP",
			args: args{
				status: true,
			},
			want: "UP",
		},
		{
			name: "Test Case Positive - Return DOWN",
			args: args{
				status: false,
			},
			want: "DOWN",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStatus(tt.args.status); got != tt.want {
				t.Errorf("getStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_GetHealthcheck(t *testing.T) {
	type fields struct {
		sharedHolder       *models.SharedHolder
		Response           helpers.Response
		HealthcheckService healthcheck.ViewService
	}

	var (
		serviceMock  = new(services_mocks.ViewHealthCheckService)
		responseMock = new(helper_mocks.Response)
		e            = echo.New()
		rec          = httptest.NewRecorder()
		requestId    = ""

		fieldsMock = fields{
			sharedHolder:       &models.SharedHolder{},
			Response:           responseMock,
			HealthcheckService: serviceMock,
		}
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthcheck", nil)

	req.Header.Set("Content-Type", "application/json")
	contextReq := e.NewContext(req, rec)
	contextReq.Set("RequestID", requestId)

	ctx := contextReq.Request().Context()

	type args struct {
		ec echo.Context
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		mockSetup func()
	}{
		{
			name:   "Test Case Positive - Return Response Healthcheck",
			fields: fieldsMock,
			args: args{
				ec: contextReq,
			},
			wantErr: false,
			mockSetup: func() {
				serviceMock.On("GetDatabaseStatus", ctx, requestId).Return(true).Times(1)
				ResponseArgs := models.ResponseArgs{
					Ctx: contextReq,
					RespData: map[string]interface{}{
						"database": getStatus(true),
					},
				}
				responseMock.On("CommonResponse", ResponseArgs).Return(nil).Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			c := &Controller{
				sharedHolder:       tt.fields.sharedHolder,
				Response:           tt.fields.Response,
				HealthcheckService: tt.fields.HealthcheckService,
			}
			if err := c.GetHealthcheck(tt.args.ec); (err != nil) != tt.wantErr {
				t.Errorf("Controller.GetHealthcheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
