package healthcheck

import (
	"context"
	"errors"
	"testing"

	"codebase/models"
	repository_mock "codebase/test/mock/repository"

	"codebase/pkg/shared"
)

func Test_service_GetDatabaseStatus(t *testing.T) {
	type fields struct {
		sharedHolder     *models.SharedHolder
		repositoryHolder repositoryHolder
		now              shared.Now
	}

	var (
		repositoriesMock = new(repository_mock.MockHealthCheckRepository)
		ctx              = context.Background()
		requestID        = "request-id"

		mockField = fields{
			repositoryHolder: repositoryHolder{
				HealthcheckRepository: repositoriesMock,
			},
		}
	)

	type args struct {
		ctx       context.Context
		requestID string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		mockSetup func()
	}{
		{
			name:   "Test Case Negative - Error get ping from DB repository on Service Healthcheck",
			fields: mockField,
			args: args{
				ctx:       ctx,
				requestID: requestID,
			},
			want: false,
			mockSetup: func() {
				repositoriesMock.On("Ping", ctx, requestID).Return(errors.New("error unexpected")).Times(1)

			},
		},
		{
			name:   "Test Case Positive - Success get ping from DB repository on Service Healthcheck",
			fields: mockField,
			args: args{
				ctx:       ctx,
				requestID: requestID,
			},
			want: true,
			mockSetup: func() {
				repositoriesMock.On("Ping", ctx, requestID).Return(nil).Times(1)

			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			s := &service{
				sharedHolder:     tt.fields.sharedHolder,
				repositoryHolder: tt.fields.repositoryHolder,
				now:              tt.fields.now,
			}
			if got := s.GetDatabaseStatus(tt.args.ctx, tt.args.requestID); got != tt.want {
				t.Errorf("service.GetDatabaseStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
