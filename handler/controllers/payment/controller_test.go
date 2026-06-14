package payment

import (
	"testing"

	"codebase/internal/services/payment"
	"codebase/models"
	helper_mocks "codebase/test/mock/helper"
	services_mocks "codebase/test/mock/service"
)

func TestNewController(t *testing.T) {
	type args struct {
		holder *models.SharedHolder
	}
	var (
		paymentMock    = new(services_mocks.ViewPaymentService)
		response       = new(helper_mocks.Response)
		binding        = new(helper_mocks.Binding)
		controllerMock = Controller{
			sharedHolder:   &models.SharedHolder{},
			Response:       response,
			Binding:        binding,
			PaymentService: paymentMock,
		}
	)
	tests := []struct {
		name string
		args args
		want Controller
	}{
		{
			name: "Test Case Positive -  Success Return Payment Controller",
			args: args{
				holder: &models.SharedHolder{},
			},
			want: controllerMock,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewController(tt.args.holder)
			if _, ok := got.PaymentService.(payment.Service); !ok {
				t.Errorf("PaymentService has wrong type, want *payment.Service")
			}
		})
	}
}
