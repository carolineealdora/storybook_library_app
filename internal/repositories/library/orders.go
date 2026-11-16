package library

import (
	"time"

	"codebase/models"
)

const bookOrderCols = `o.id, o.book_id, b.title AS book_title, o.price, o.status, o.created_at, o.paid_at`

func (r *Repo) CreateBookOrder(userID, bookID string, price int, idempotencyKey string) (*models.BookOrder, error) {
	var o models.BookOrder
	err := r.db.RawSql(`WITH o AS (
			INSERT INTO book_orders (user_id, book_id, price, status, idempotency_key)
			VALUES (?, ?, ?, 'pending', ?) RETURNING *)
		SELECT `+bookOrderCols+` FROM o JOIN books b ON b.id = o.book_id`, &o,
		userID, bookID, price, idempotencyKey)
	return one(&o, o.ID, err)
}

// LockBookOrder reads the order and row-locks it for the rest of the transaction (double-tap on Pay).
func (r *Repo) LockBookOrder(orderID, userID string) (*models.BookOrder, error) {
	var o models.BookOrder
	err := r.db.RawSql(`SELECT `+bookOrderCols+` FROM book_orders o JOIN books b ON b.id = o.book_id
		WHERE o.id = ? AND o.user_id = ? FOR UPDATE OF o`, &o, orderID, userID)
	return one(&o, o.ID, err)
}

func (r *Repo) SetBookOrderStatus(orderID, status string, paidAt *time.Time) error {
	return r.db.Exec(`UPDATE book_orders SET status = ?, paid_at = ? WHERE id = ?`, status, paidAt, orderID)
}

func (r *Repo) CreatePurchase(userID, bookID, orderID string, pricePaid int) error {
	return r.db.Exec(`INSERT INTO purchases (user_id, book_id, book_order_id, price_paid) VALUES (?, ?, ?, ?)`,
		userID, bookID, orderID, pricePaid)
}

// ListBookOrders is Order History: finished book orders only, newest first.
func (r *Repo) ListBookOrders(userID string, limit, offset int) ([]models.BookOrder, int64, error) {
	return page[models.BookOrder](r.db, limit, offset, `SELECT `+bookOrderCols+` FROM book_orders o JOIN books b ON b.id = o.book_id
		WHERE o.user_id = ? AND o.status IN ('paid', 'failed')
		ORDER BY o.created_at DESC, o.id`, userID)
}
