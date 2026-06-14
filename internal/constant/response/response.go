package response

import (
	"net/http"

	"codebase/models"
)

var Responses = map[string]models.ResponsePayload{
	// 200 - Success
	SUCCESS: {Code: "0001", Desc: "Success", HttpCode: http.StatusOK},

	// 400 - Bad Request
	ACCOUNT_WO:             {Code: "3039", Desc: "Account write off, not allowed to generated bill", HttpCode: http.StatusBadRequest},
	INVALID_DATE_FORMAT:    {Code: "4000", Desc: "Invalid date format", HttpCode: http.StatusBadRequest},
	OUTBOUND_FAILED:        {Code: "4003", Desc: "Error occurred in the outbound service", HttpCode: http.StatusBadRequest},
	MANDATORY_FIELD:        {Code: "9001", Desc: "Mandatory field", HttpCode: http.StatusBadRequest},
	INVALID_REQUEST_HEADER: {Code: "9002", Desc: "Invalid request header", HttpCode: http.StatusBadRequest},
	AMOUNT_LESS_THAN_ZERO:  {Code: "9003", Desc: "Amount must be greater than 0", HttpCode: http.StatusBadRequest},
	INVALID_TRX_ID:         {Code: "9003", Desc: "TrxId must be valid alphanumeric", HttpCode: http.StatusBadRequest},
	INVALID_REQUEST_METHOD: {Code: "9006", Desc: "GET request should not have a body", HttpCode: http.StatusBadRequest},

	// 404 - Not Found
	PARTNER_NOT_FOUND:        {Code: "1002", Desc: "Partner not found", HttpCode: http.StatusNotFound},
	CUSTOMER_NOT_FOUND:       {Code: "2002", Desc: "Customer not found", HttpCode: http.StatusNotFound},
	LOAN_APP_NOT_FOUND:       {Code: "3001", Desc: "Loan application not found", HttpCode: http.StatusNotFound},
	CHARGE_NOT_FOUND:         {Code: "4001", Desc: "Charge not found", HttpCode: http.StatusNotFound},
	TRANSACTION_ID_NOT_FOUND: {Code: "4002", Desc: "Transaction ID not found", HttpCode: http.StatusNotFound},
	BILL_NOT_FOUND:           {Code: "5002", Desc: "Bill not found", HttpCode: http.StatusNotFound},
	PATH_NOT_FOUND:           {Code: "9007", Desc: "Requested URL path not found", HttpCode: http.StatusNotFound},

	// 409 - Conflict
	INVALID_LOAN_APP_STATUS: {Code: "3003", Desc: "Invalid loan application status", HttpCode: http.StatusConflict},
	DUPLICATE_TRX_ID:        {Code: "4007", Desc: "Duplicate TrxId", HttpCode: http.StatusConflict},

	// 500 - Internal Server Error
	EARLY_TERMINATION_FAILED: {Code: "5004", Desc: "Early termination process failed", HttpCode: http.StatusInternalServerError},
	PAYMENT_FAILED:           {Code: "5005", Desc: "Payment failed", HttpCode: http.StatusInternalServerError},
	GENERAL_ERROR:            {Code: "9999", Desc: "General Error", HttpCode: http.StatusInternalServerError},

	// 429 - To Many Request
	CAN_NOT_ACCESS_API: {Code: "9004", Desc: "Same request is still on process", HttpCode: http.StatusTooManyRequests},
}

const (
	SUCCESS                  = "SUCCESS"
	ACCOUNT_WO               = "ACCOUNT_WO"
	INVALID_DATE_FORMAT      = "INVALID_DATE_FORMAT"
	OUTBOUND_FAILED          = "OUTBOUND_FAILED"
	MANDATORY_FIELD          = "MANDATORY_FIELD"
	INVALID_REQUEST_HEADER   = "INVALID_REQUEST_HEADER"
	INVALID_PARTNER_HEADER   = "INVALID_PARTNER_HEADER"
	AMOUNT_LESS_THAN_ZERO    = "AMOUNT_LESS_THAN_ZERO"
	INVALID_TRX_ID           = "INVALID_TRX_ID"
	INVALID_REQUEST_METHOD   = "INVALID_REQUEST_METHOD"
	PARTNER_NOT_FOUND        = "PARTNER_NOT_FOUND"
	CUSTOMER_NOT_FOUND       = "CUSTOMER_NOT_FOUND"
	LOAN_APP_NOT_FOUND       = "LOAN_APP_NOT_FOUND"
	CHARGE_NOT_FOUND         = "CHARGE_NOT_FOUND"
	TRANSACTION_ID_NOT_FOUND = "TRANSACTION_ID_NOT_FOUND"
	BILL_NOT_FOUND           = "BILL_NOT_FOUND"
	PATH_NOT_FOUND           = "PATH_NOT_FOUND"
	INVALID_LOAN_APP_STATUS  = "INVALID_LOAN_APP_STATUS"
	DUPLICATE_TRX_ID         = "DUPLICATE_TRX_ID"
	EARLY_TERMINATION_FAILED = "EARLY_TERMINATION_FAILED"
	PAYMENT_FAILED           = "PAYMENT_FAILED"
	GENERAL_ERROR            = "GENERAL_ERROR"
	CAN_NOT_ACCESS_API       = "CAN_NOT_ACCESS_API"
)
