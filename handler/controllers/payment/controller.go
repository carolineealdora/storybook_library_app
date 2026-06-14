package payment

import (
	helpers "codebase/internal/helper"
	"codebase/internal/services/payment"
	"codebase/models"
)

type (
	Controller struct {
		sharedHolder *models.SharedHolder
		Response     helpers.Response
		Binding      helpers.Binding
		Errors       helpers.Error

		//input services here
		PaymentService payment.Service
	}
)

func NewController(holder *models.SharedHolder) Controller {
	response := helpers.NewResponse(holder)
	binding := helpers.NewBinding()
	errors := helpers.NewError(holder)

	//init services here
	paymentService, err := payment.NewViewService(holder)
	if err != nil {
		panic(err)
	}

	return Controller{
		sharedHolder:   holder,
		PaymentService: paymentService,
		Response:       response,
		Binding:        binding,
		Errors:         errors,
	}
}
