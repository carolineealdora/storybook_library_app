package payment

import (
	"errors"
	"reflect"
	"testing"

	"codebase/models"
	sql_mock "codebase/test/mock/sql"

	sql "codebase/pkg/orm"
	"codebase/pkg/shared"
)

func Test_repo_GetByTrxId(t *testing.T) {
	type fields struct {
		sharedHolder *models.SharedHolder
		now          shared.Now
	}
	var (
		sqlMock    = new(sql_mock.ORMWrapperMock)
		trxId      = "trxId"
		fieldsMock = fields{
			sharedHolder: &models.SharedHolder{
				Sql: sqlMock,
			},
		}
		queries = []sql.Query{
			{
				Field:    "trx_id",
				Operator: "=",
				Value:    trxId,
			},
		}
		payment *models.Payment
	)
	type args struct {
		trxId string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      *models.Payment
		wantErr   bool
		mockSetup func()
	}{
		{
			name:   "Test Case Negative - Error FindByQuery on DB GetByTrxId Payment",
			fields: fieldsMock,
			args: args{
				trxId: trxId,
			},
			want:    nil,
			wantErr: true,
			mockSetup: func() {
				sqlMock.On("FindByQuery", models.TablePayment, []string{}, queries, &payment).Return(sqlMock).Times(1)
				sqlMock.On("Error").Return(errors.New("unexpected error")).Times(1)
			},
		},
		{
			name:   "Test Case Negative - Record not Found FindByQuery on DB GetByTrxId Payment",
			fields: fieldsMock,
			args: args{
				trxId: trxId,
			},
			want:    nil,
			wantErr: true,
			mockSetup: func() {
				sqlMock.On("FindByQuery", models.TablePayment, []string{}, queries, &payment).Return(sqlMock).Times(1)
				sqlMock.On("Error").Return(nil).Times(1)
				sqlMock.On("RowsAffected").Return(int64(0)).Times(1)
			},
		},
		{
			name:   "Test Case Positive - Success FindByQuery on DB GetByTrxId Payment",
			fields: fieldsMock,
			args: args{
				trxId: trxId,
			},
			want:    nil,
			wantErr: false,
			mockSetup: func() {
				sqlMock.On("FindByQuery", models.TablePayment, []string{}, queries, &payment).Return(sqlMock).Times(1)
				sqlMock.On("Error").Return(nil).Times(1)
				sqlMock.On("RowsAffected").Return(int64(1)).Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			r := Repo{
				sharedHolder: tt.fields.sharedHolder,
				now:          tt.fields.now,
			}
			got, err := r.GetByTrxId(tt.args.trxId)
			if (err != nil) != tt.wantErr {
				t.Errorf("repo.GetByTrxId() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("repo.GetByTrxId() = %v, want %v", got, tt.want)
			}
		})
	}
}
