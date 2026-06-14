package payment

import (
	"codebase/models"

	" persistent/sql/sql"

	"gorm.io/gorm"
)

func (r Repo) GetByRefId(refId string) (*models.Payment, error) {
	var (
		queries = []sql.Query{
			{
				Field:    "ref_id",
				Operator: "=",
				Value:    refId,
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
