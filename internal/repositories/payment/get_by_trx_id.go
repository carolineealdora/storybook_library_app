package payment

import (
	"codebase/models"
	sql "codebase/pkg/orm"

	"gorm.io/gorm"
)

func (r Repo) GetByTrxId(trxId string) (*models.Payment, error) {
	var (
		queries = []sql.Query{
			{
				Field:    "trx_id",
				Operator: "=",
				Value:    trxId,
			},
		}
		findResult sql.ORM
		payment    *models.Payment
	)

	findResult = r.sharedHolder.Sql.FindByQuery(models.TablePayment, []string{}, queries, &payment)
	if err := findResult.Error(); err != nil {
		return nil, err
	}

	if findResult.RowsAffected() == int64(0) {
		return nil, gorm.ErrRecordNotFound
	}

	return payment, nil
}
