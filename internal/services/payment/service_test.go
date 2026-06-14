package payment

import (
	"testing"

	"codebase/models"
	repository_mock "codebase/test/mock/repository"
)

func TestNewViewService(t *testing.T) {
	type args struct {
		sharedHolder *models.SharedHolder
	}

	var (
		paymentRepositoryMock = new(repository_mock.MockPaymentRepository)

		viewServiceImpl = ServiceImpl{
			repositoryHolder: repositoryHolder{
				PaymentRepository: paymentRepositoryMock,
			},
			sharedHolder: &models.SharedHolder{},
		}
	)

	tests := []struct {
		name    string
		args    args
		want    Service
		wantErr bool
	}{
		{
			name: "Test Case Positive - Success Return Service Payment",
			args: args{
				sharedHolder: viewServiceImpl.sharedHolder,
			},
			want:    viewServiceImpl,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewViewService(tt.args.sharedHolder)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewViewService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
