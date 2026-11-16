// Package book covers the showcase, book detail, viewer content, My Book (PRD US-04..US-08)
// and the back-office Master Book CRUD (US-16).
package book

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

// Access applies the PRD §12.1 order: free -> purchased -> active subscription -> locked.
func Access(price int, owned, subscribed bool) string {
	switch {
	case price == 0:
		return models.AccessFree
	case owned:
		return models.AccessOwned
	case subscribed:
		return models.AccessSubscribed
	default:
		return models.AccessLocked
	}
}

// Visible: on the shelf, or owned (owners keep soft-deleted/unpublished books, PRD E5).
func Visible(b *models.Book) bool {
	return b != nil && (b.Owned || (b.Status == models.BookPublished && b.DeletedAt == nil))
}

func (s *Service) List(userID string, limit, offset int) ([]models.Book, int64, error) {
	books, total, err := s.repo.ListPublishedBooks(userID, limit, offset)
	if err != nil {
		return nil, 0, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	books, err = s.decorate(userID, books)
	return books, total, err
}

func (s *Service) MyBooks(userID string, limit, offset int) ([]models.Book, int64, error) {
	books, total, err := s.repo.ListPurchasedBooks(userID, limit, offset)
	if err != nil {
		return nil, 0, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	books, err = s.decorate(userID, books)
	return books, total, err
}

func (s *Service) Detail(userID, bookID string) (*models.Book, error) {
	b, err := s.visibleBook(userID, bookID)
	if err != nil {
		return nil, err
	}
	list, err := s.decorate(userID, []models.Book{*b})
	if err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (s *Service) Cover(userID, bookID string) (*models.BookFile, error) {
	if _, err := s.visibleBook(userID, bookID); err != nil {
		return nil, err
	}
	return s.cover(bookID)
}

// Content streams the EPUB only after the access check passes (PRD US-07, E4, E18).
func (s *Service) Content(userID, bookID string) (*models.BookFile, error) {
	b, err := s.Detail(userID, bookID)
	if err != nil {
		return nil, err
	}
	if b.Access == models.AccessLocked {
		return nil, s.errors.Wrap(respConst.BOOK_LOCKED, "content: book "+bookID+" locked")
	}
	f, err := s.repo.GetEpub(bookID)
	if err != nil || f == nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, "content: epub missing for "+bookID)
	}
	return f, nil
}

func (s *Service) visibleBook(userID, bookID string) (*models.Book, error) {
	b, err := s.repo.GetBookForUser(bookID, userID)
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	if !Visible(b) {
		return nil, s.errors.Wrap(respConst.BOOK_NOT_FOUND, "book "+bookID+" not visible")
	}
	return b, nil
}

func (s *Service) decorate(userID string, books []models.Book) ([]models.Book, error) {
	subscribed, err := s.repo.HasActiveSubscription(userID)
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	for i := range books {
		b := &books[i]
		// Subscription unlocks the shelf only; a delisted book stays readable just for its owners.
		onShelf := b.Status == models.BookPublished && b.DeletedAt == nil
		b.Access = Access(b.Price, b.Owned, subscribed && onShelf)
		b.AgeLabel = AgeLabel(b.AgeType, b.AgeMin, b.AgeMax)
	}
	return books, nil
}

func (s *Service) cover(bookID string) (*models.BookFile, error) {
	f, err := s.repo.GetCover(bookID)
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	if f == nil {
		return nil, s.errors.Wrap(respConst.BOOK_NOT_FOUND, "cover missing for "+bookID)
	}
	return f, nil
}
