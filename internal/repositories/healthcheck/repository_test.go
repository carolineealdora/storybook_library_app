package healthcheck

import (
	"context"
	"testing"

	"codebase/models"
	sql_mock "codebase/test/mock/sql"
)

func TestNewHealthcheckRepository(t *testing.T) {
	type args struct {
		ctx          context.Context
		sharedHolder *models.SharedHolder
	}
	var (
		sqlMock         = new(sql_mock.ORMWrapperMock)
		ctx             = context.Background()
		viewServiceImpl = Repo{
			sharedHolder: &models.SharedHolder{
				Sql: sqlMock,
			},
			now: nil,
		}
	)
	tests := []struct {
		name    string
		args    args
		want    Repository
		wantErr bool
	}{
		{
			name: "Test Case Positive - Success Return HealthCheck Repository",
			args: args{
				ctx:          ctx,
				sharedHolder: viewServiceImpl.sharedHolder,
			},
			want:    &viewServiceImpl,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewHealthcheckRepository(tt.args.ctx, tt.args.sharedHolder)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHealthcheckRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
