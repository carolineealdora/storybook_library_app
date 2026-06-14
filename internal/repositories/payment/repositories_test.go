package payment

import (
	"testing"

	"codebase/models"
	sql_mock "codebase/test/mock/sql"
)

func TestNewRepository(t *testing.T) {
	type args struct {
		sharedHolder *models.SharedHolder
	}
	var (
		sqlMock         = new(sql_mock.ORMWrapperMock)
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
			name: "Test Case Positive - Return Success Payment Repository",
			args: args{
				sharedHolder: viewServiceImpl.sharedHolder,
			},
			want:    viewServiceImpl,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRepository(tt.args.sharedHolder)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
