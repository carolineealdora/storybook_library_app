package payment

import (
	"errors"
	"reflect"
	"testing"

	respConst "codebase/internal/constant/response"
	helpers "codebase/internal/helper"
	"codebase/models"
	helper_mocks "codebase/test/mock/helper"
	repository_mock "codebase/test/mock/repository"

	"codebase/pkg/shared"

	"gorm.io/gorm"
)

func TestServiceImpl_GetByRefId(t *testing.T) {
	type fields struct {
		repositoryHolder repositoryHolder
		sharedHolder     *models.SharedHolder
		now              shared.Now
		errors           helpers.Error
	}
	var (
		repositoriesMock = new(repository_mock.MockPaymentRepository)
		errorMock        = new(helper_mocks.Error)
		refId            = "refId"
		partnerId        = "102"
		fieldsMock       = fields{
			repositoryHolder: repositoryHolder{
				PaymentRepository: repositoriesMock,
			},
			sharedHolder: &models.SharedHolder{},
			now:          nil,
			errors:       errorMock,
		}
		payment = models.Payment{
			CommonDatabaseModel: models.CommonDatabaseModel{
				ID: 1,
			},
			RefId:       refId,
			TotalAmount: 100000,
			PartnerId:   103,
		}
	)
	type args struct {
		refId     string
		partnerId string
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
			name:   "Test Case Negative - Failed GetByRefId Repository Record Not Found on Service Payment",
			fields: fieldsMock,
			args: args{
				refId:     refId,
				partnerId: partnerId,
			},
			want:    nil,
			wantErr: true,
			mockSetup: func() {
				repositoriesMock.On("GetByRefId", refId).Return(&models.Payment{}, gorm.ErrRecordNotFound).Times(1)
				err := gorm.ErrRecordNotFound
				errorMock.On("Wrap", respConst.BILL_NOT_FOUND, err.Error()).Return(err).Times(1)
			},
		},
		{
			name:   "Test Case Negative - Failed GetByRefId Repository GENERAL ERROR on Service Payment",
			fields: fieldsMock,
			args: args{
				refId:     refId,
				partnerId: partnerId,
			},
			want:    nil,
			wantErr: true,
			mockSetup: func() {
				repositoriesMock.On("GetByRefId", refId).Return(&models.Payment{}, errors.New("unexpected error")).Times(1)
				err := errors.New("unexpected error")
				errorMock.On("Wrap", respConst.GENERAL_ERROR, err.Error()).Return(err).Times(1)
			},
		},
		{
			name:   "Test Case Negative - Failed PartnerId not found on Service Payment",
			fields: fieldsMock,
			args: args{
				refId:     refId,
				partnerId: partnerId,
			},
			want:    nil,
			wantErr: true,
			mockSetup: func() {
				repositoriesMock.On("GetByRefId", refId).Return(&payment, nil).Times(1)
				err := errors.New("GetByRefId failed: expected partnerId=103, got=102")
				errorMock.On("Wrap", respConst.PARTNER_NOT_FOUND, err.Error()).Return(err).Times(1)
			},
		},
		{
			name:   "Test Case Positive - Success Get By Ref Id on Service Payment",
			fields: fieldsMock,
			args: args{
				refId:     refId,
				partnerId: "103",
			},
			want:    &payment,
			wantErr: false,
			mockSetup: func() {
				repositoriesMock.On("GetByRefId", refId).Return(&payment, nil).Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			s := ServiceImpl{
				repositoryHolder: tt.fields.repositoryHolder,
				sharedHolder:     tt.fields.sharedHolder,
				now:              tt.fields.now,
				errors:           tt.fields.errors,
			}
			got, err := s.GetByRefId(tt.args.refId, tt.args.partnerId)
			if (err != nil) != tt.wantErr {
				t.Errorf("ServiceImpl.GetByRefId() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ServiceImpl.GetByRefId() = %v, want %v", got, tt.want)
			}
		})
	}
}
