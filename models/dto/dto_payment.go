package dto

import (
	"time"

	"codebase/models"
)

type (
	GetByRefIdParam struct {
		RefId string `param:"refId" validate:"required"`
	}

	PaymentResponse struct {
		RefId        string    `json:"paymentRefId"`
		LoanAppRefId string    `json:"loanApplicationRefId"`
		TrxId        string    `json:"trxId"`
		Status       string    `json:"status"`
		CreatedAt    time.Time `json:"createdAt"`
	}
)

func ToPaymentResponse(payment models.Payment) *PaymentResponse {
	return &PaymentResponse{
		RefId:     payment.RefId,
		CreatedAt: payment.CreatedAt,
	}
}

func (u GetByRefIdParam) Validate() interface{} {
	return u
}
