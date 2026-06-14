package payment

import (
	"fmt"
	"strconv"

	respConst "codebase/internal/constant/response"
	"codebase/models"

	"gorm.io/gorm"
)

func (s ServiceImpl) GetByRefId(refId, partnerId string) (*models.Payment, error) {
	result, err := s.repositoryHolder.PaymentRepository.GetByRefId(refId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, s.errors.Wrap(respConst.BILL_NOT_FOUND, err.Error())
		}
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}

	partnerIdInt, _ := strconv.Atoi(partnerId)
	if result.PartnerId != partnerIdInt {
		return nil, s.errors.Wrap(respConst.PARTNER_NOT_FOUND,
			fmt.Sprintf("GetByRefId failed: expected partnerId=%d, got=%d", result.PartnerId, partnerIdInt))
	}

	return result, nil
}
