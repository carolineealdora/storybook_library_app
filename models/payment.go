package models

const TablePayment = "payment"

type Payment struct {
	CommonDatabaseModel
	RefId       string `gorm:"column:ref_id" json:"ref_id"`
	TotalAmount int    `gorm:"column:total_amount" json:"total_amount"`
	PartnerId   int    `gorm:"column:partner_id" json:"partner_id"`
}

func (c *Payment) TableName() string {
	return TablePayment
}
