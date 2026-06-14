package payment

import (
	"time"

	"codebase/models"
	"codebase/pkg/shared"
)

type (
	Repository interface {
		GetByRefId(refId string) (*models.Payment, error)
	}

	Repo struct {
		sharedHolder *models.SharedHolder
		now          shared.Now
	}
)

func NewRepository(sharedHolder *models.SharedHolder) (Repository, error) {
	return &Repo{sharedHolder, time.Now}, nil
}
