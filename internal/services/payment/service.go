package payment

import (
	"time"

	helpers "codebase/internal/helper"
	"codebase/internal/repositories/payment"
	"codebase/models"
	"codebase/pkg/shared"
)

type (
	Service interface {
		GetByRefId(paymentRefId, partnerId string) (*models.Payment, error)
	}

	repositoryHolder struct {
		PaymentRepository payment.Repository
	}

	ServiceImpl struct {
		repositoryHolder repositoryHolder
		sharedHolder     *models.SharedHolder
		now              shared.Now
		errors           helpers.Error
	}
)

func NewViewService(sharedHolder *models.SharedHolder) (Service, error) {
	paymentRepository, err := payment.NewRepository(sharedHolder)
	errors := helpers.NewError(sharedHolder)

	if err != nil {
		panic(err)
	}
	rHolder := repositoryHolder{
		PaymentRepository: paymentRepository,
	}

	return &ServiceImpl{rHolder, sharedHolder, time.Now, errors}, nil
}
