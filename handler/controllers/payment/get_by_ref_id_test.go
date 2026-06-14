package payment

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	respConst "codebase/internal/constant/response"
	helpers "codebase/internal/helper"
	"codebase/internal/services/payment"
	"codebase/models"
	"codebase/models/dto"
	helper_mocks "codebase/test/mock/helper"
	services_mocks "codebase/test/mock/service"

	"github.com/labstack/echo/v4"
)

func TestController_GetByRefId(t *testing.T) {
	type fields struct {
		sharedHolder   *models.SharedHolder
		Response       helpers.Response
		Binding        helpers.Binding
		PaymentService payment.Service
		Errors         helpers.Error
	}
	var (
		paymentMock  = new(services_mocks.ViewPaymentService)
		responseMock = new(helper_mocks.Response)
		bindingMock  = new(helper_mocks.Binding)
		errorMock    = new(helper_mocks.Error)
		e            = echo.New()
		rec          = httptest.NewRecorder()
		requestId    = "requestId"
		partnerId    = "103"
		fieldsMock   = fields{
			sharedHolder:   &models.SharedHolder{},
			Response:       responseMock,
			Binding:        bindingMock,
			PaymentService: paymentMock,
			Errors:         errorMock,
		}

		param dto.GetByRefIdParam
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment/ref/:refId", nil)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-partner-id", partnerId)

	contextReq := e.NewContext(req, rec)
	contextReq.Set("RequestID", requestId)

	// ctx := contextReq.Request().Context()

	CustomBinder := helpers.CustomBinder{
		Request: &param,
		Context: contextReq,
	}

	Payment := models.Payment{
		CommonDatabaseModel: models.CommonDatabaseModel{
			ID: 0,
		},
		RefId:       "",
		TotalAmount: 0,
		PartnerId:   0,
	}
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
			name:   "Test Case Negative - Response Error Binding Request Get By Ref ID Payment Controller",
			fields: fieldsMock,
			args: args{
				ec: contextReq,
			},
			wantErr: false,
			mockSetup: func() {
				err := errors.New("unexpected error")
				bindingMock.On("Bind", CustomBinder).Return(err).Times(1)
				errorMock.On("Wrap", respConst.MANDATORY_FIELD, err.Error()).Return(err).Times(1)
				ResponseArgs := models.ResponseArgs{
					Ctx:       contextReq,
					ErrorData: err,
				}
				responseMock.On("CommonResponse", ResponseArgs).Return(nil).Times(1)
			},
		},
		{
			name:   "Test Case Negative - Response Error Get By Ref ID Payment Controller",
			fields: fieldsMock,
			args: args{
				ec: contextReq,
			},
			wantErr: false,
			mockSetup: func() {
				err := errors.New("unexpected error")
				bindingMock.On("Bind", CustomBinder).Return(nil).Times(1)
				paymentMock.On("GetByRefId", param.RefId, partnerId).Return(nil, err).Times(1)

				ResponseArgs := models.ResponseArgs{
					Ctx:       contextReq,
					ErrorData: err,
				}
				responseMock.On("CommonResponse", ResponseArgs).Return(nil).Times(1)
			},
		},
		{
			name:   "Test Case Positive - Response Success Get By Ref ID Payment Controller",
			fields: fieldsMock,
			args: args{
				ec: contextReq,
			},
			wantErr: false,
			mockSetup: func() {
				bindingMock.On("Bind", CustomBinder).Return(nil).Times(1)
				paymentMock.On("GetByRefId", param.RefId, partnerId).Return(&Payment, nil).Times(1)

				ResponseArgs := models.ResponseArgs{
					Ctx:      contextReq,
					RespData: dto.ToPaymentResponse(Payment),
				}

				responseMock.On("CommonResponse", ResponseArgs).Return(nil).Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				sharedHolder:   tt.fields.sharedHolder,
				Response:       tt.fields.Response,
				Binding:        tt.fields.Binding,
				PaymentService: tt.fields.PaymentService,
				Errors:         tt.fields.Errors,
			}
			tt.mockSetup()
			if err := c.GetByRefId(tt.args.ec); (err != nil) != tt.wantErr {
				t.Errorf("Controller.GetByRefId() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
