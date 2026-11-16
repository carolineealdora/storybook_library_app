// Package order is single-book checkout with mocked payment (PRD US-09, US-12).
// SECURITY: manual review required — payment state transitions live here.
package order

import (
	"time"

	respConst "codebase/internal/constant/response"
	helpers "codebase/internal/helper"
	"codebase/internal/repositories/library"
	"codebase/models"
	"codebase/pkg/shared"
)

type Service struct {
	repo   *library.Repo
	errors helpers.Error
	now    shared.Now
}

func NewService(holder *models.SharedHolder) *Service {
	return &Service{repo: library.NewRepository(holder), errors: helpers.NewError(holder), now: time.Now}
}

// Create opens a pending order for one book (Buy Now -> Checkout). The price is snapshotted
// now and never re-read from the book (PRD E7).
func (s *Service) Create(userID, bookID, idempotencyKey string) (*models.BookOrder, error) {
	b, err := s.purchasable(s.repo, userID, bookID, respConst.BOOK_NOT_FOUND)
	if err != nil {
		return nil, err
	}
	o, err := s.repo.CreateBookOrder(userID, bookID, b.Price, idempotencyKey)
	if library.IsUniqueViolation(err) {
		return nil, s.errors.Wrap(respConst.DUPLICATE_REQUEST, "book order: idempotency key reused")
	}
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	return o, nil
}

// Pay runs the mocked payment. The order row is locked so a double-tap pays once (E8).
func (s *Service) Pay(userID, orderID string, success bool) (*models.BookOrder, error) {
	var order *models.BookOrder
	err := s.repo.InTx(func(tx *library.Repo) error {
		o, err := tx.LockBookOrder(orderID, userID)
		if err != nil {
			return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
		}
		if o == nil {
			return s.errors.Wrap(respConst.ORDER_NOT_FOUND, "pay: order "+orderID+" not found")
		}
		if o.Status != models.OrderPending {
			return s.errors.Wrap(respConst.DUPLICATE_REQUEST, "pay: order "+orderID+" already "+o.Status)
		}
		if _, err := s.purchasable(tx, userID, o.BookID, respConst.BOOK_UNAVAILABLE); err != nil {
			return err
		}

		// ponytail: mocked gateway — the client picks the outcome. A real gateway call goes here.
		now := s.now()
		o.Status, o.PaidAt = models.OrderFailed, nil
		if success {
			o.Status, o.PaidAt = models.OrderPaid, &now
		}
		if err := tx.SetBookOrderStatus(o.ID, o.Status, o.PaidAt); err != nil {
			return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
		}
		if success {
			err := tx.CreatePurchase(userID, o.BookID, o.ID, o.Price)
			if library.IsUniqueViolation(err) {
				return s.errors.Wrap(respConst.BOOK_ALREADY_OWNED, "pay: book bought by a parallel order")
			}
			if err != nil {
				return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
			}
		}
		order = o
		return nil
	})
	if err != nil {
		return nil, helpers.Coded(s.errors, err)
	}
	if order.Status == models.OrderFailed {
		return nil, s.errors.Wrap(respConst.PAYMENT_FAILED, "pay: mocked payment failed for "+orderID)
	}
	return order, nil
}

func (s *Service) History(userID string, limit, offset int) ([]models.BookOrder, int64, error) {
	orders, total, err := s.repo.ListBookOrders(userID, limit, offset)
	if err != nil {
		return nil, 0, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	return orders, total, nil
}

// purchasable enforces PRD E1, E2, E6, E10, E22. unavailableCode differs between
// opening checkout (404 BOOK_NOT_FOUND) and paying (409 BOOK_UNAVAILABLE).
func (s *Service) purchasable(repo *library.Repo, userID, bookID, unavailableCode string) (*models.Book, error) {
	b, err := repo.GetBookForUser(bookID, userID)
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	switch {
	case b == nil:
		return nil, s.errors.Wrap(respConst.BOOK_NOT_FOUND, "book "+bookID+" not found")
	case b.Owned:
		return nil, s.errors.Wrap(respConst.BOOK_ALREADY_OWNED, "book "+bookID+" already owned")
	case b.Status != models.BookPublished || b.DeletedAt != nil:
		return nil, s.errors.Wrap(unavailableCode, "book "+bookID+" not on the shelf")
	case b.Price == 0:
		return nil, s.errors.Wrap(respConst.BOOK_NOT_PURCHASABLE, "book "+bookID+" is free")
	}
	subscribed, err := repo.HasActiveSubscription(userID)
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	if subscribed {
		return nil, s.errors.Wrap(respConst.SUBSCRIPTION_ACTIVE, "user "+userID+" is subscribed")
	}
	return b, nil
}
