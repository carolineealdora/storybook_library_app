package healthcheck

import (
	"context"
	"testing"

	"codebase/models"
	sql_mock "codebase/test/mock/sql"

	"shared/shared"
)

func Test_repo_Ping(t *testing.T) {
	type fields struct {
		sharedHolder *models.SharedHolder
		now          shared.Now
	}
	var (
		sqlMock    = new(sql_mock.ORMWrapperMock)
		ctx        = context.Background()
		fieldsMock = fields{
			sharedHolder: &models.SharedHolder{
				Sql: sqlMock,
			},
		}
	)
	type args struct {
		ctx       context.Context
		requestId string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		mockSetup func()
	}{
		{
			name:   "Test Case Positive - Return Success Ping on HealthCheck Repository",
			fields: fieldsMock,
			args: args{
				ctx:       ctx,
				requestId: "requestId",
			},
			wantErr: false,
			mockSetup: func() {
				sqlMock.On("Ping").Return(nil).Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			r := &Repo{
				sharedHolder: tt.fields.sharedHolder,
				now:          tt.fields.now,
			}
			if err := r.Ping(tt.args.ctx, tt.args.requestId); (err != nil) != tt.wantErr {
				t.Errorf("repo.Ping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
