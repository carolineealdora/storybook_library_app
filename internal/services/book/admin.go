package book

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	respConst "codebase/internal/constant/response"
	"codebase/internal/repositories/library"
	"codebase/models"
)

const (
	MaxCoverBytes = 5 << 20
	MaxEpubBytes  = 50 << 20
)

type (
	Upload struct {
		FileName string
		Content  []byte
	}

	// Input is the raw multipart form. Numbers stay strings so "1.5" or "abc" can be
	// rejected as INVALID_AGE_RANGE instead of failing to bind.
	Input struct {
		Title, Author, Description, AgeType, AgeMin, AgeMax, Price, Status string
		Cover, Epub                                                        *Upload // nil on update = keep current file
	}

	inputError struct{ code, msg string }
)

func (s *Service) AdminList(search, status string, limit, offset int) ([]models.Book, int64, error) {
	books, total, err := s.repo.ListBooks(strings.TrimSpace(search), status, limit, offset)
	if err != nil {
		return nil, 0, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	for i := range books {
		books[i].AgeLabel = AgeLabel(books[i].AgeType, books[i].AgeMin, books[i].AgeMax)
	}
	return books, total, nil
}

func (s *Service) AdminGet(bookID string) (*models.Book, error) {
	b, err := s.repo.GetBook(bookID)
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	if b == nil {
		return nil, s.errors.Wrap(respConst.BOOK_NOT_FOUND, "admin: book "+bookID+" not found")
	}
	b.AgeLabel = AgeLabel(b.AgeType, b.AgeMin, b.AgeMax)
	return b, nil
}

func (s *Service) AdminCover(bookID string) (*models.BookFile, error) {
	if _, err := s.AdminGet(bookID); err != nil {
		return nil, err
	}
	return s.cover(bookID)
}

func (s *Service) Create(in Input) (*models.Book, error) {
	if in.Cover == nil || in.Epub == nil {
		return nil, s.errors.Wrap(respConst.VALIDATION_ERROR, "cover and epub files are required.")
	}
	w, ie := ValidateInput(in)
	if ie != nil {
		return nil, s.errors.Wrap(ie.code, ie.msg)
	}
	var id string
	err := s.repo.InTx(func(tx *library.Repo) (err error) {
		id, err = tx.CreateBook(w)
		return err
	})
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	return s.AdminGet(id)
}

// Update replaces the metadata (PUT). Price changes only affect new orders: orders and
// purchases keep their own price snapshot (PRD US-16, E7).
func (s *Service) Update(bookID string, in Input) (*models.Book, error) {
	w, ie := ValidateInput(in)
	if ie != nil {
		return nil, s.errors.Wrap(ie.code, ie.msg)
	}
	var found bool
	err := s.repo.InTx(func(tx *library.Repo) (err error) {
		found, err = tx.UpdateBook(bookID, w)
		return err
	})
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	if !found {
		return nil, s.errors.Wrap(respConst.BOOK_NOT_FOUND, "admin update: book "+bookID+" not found")
	}
	return s.AdminGet(bookID)
}

func (s *Service) Delete(bookID string) error {
	found, err := s.repo.SoftDeleteBook(bookID)
	if err != nil {
		return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	if !found {
		return s.errors.Wrap(respConst.BOOK_NOT_FOUND, "admin delete: book "+bookID+" not found")
	}
	return nil
}

// ValidateInput turns the raw form into a DB write or the PRD error code for the first problem.
func ValidateInput(in Input) (library.BookWrite, *inputError) {
	w := library.BookWrite{
		Title:       strings.TrimSpace(in.Title),
		Author:      strings.TrimSpace(in.Author),
		Description: strings.TrimSpace(in.Description),
		AgeType:     in.AgeType,
		Status:      in.Status,
	}
	invalid := func(msg string) (library.BookWrite, *inputError) {
		return library.BookWrite{}, &inputError{respConst.VALIDATION_ERROR, msg}
	}
	badAge := func(msg string) (library.BookWrite, *inputError) {
		return library.BookWrite{}, &inputError{respConst.INVALID_AGE_RANGE, msg}
	}

	switch {
	case w.Title == "" || len(w.Title) > 255:
		return invalid("title is required (max 255 characters).")
	case w.Author == "" || len(w.Author) > 255:
		return invalid("author is required (max 255 characters).")
	case w.Description == "":
		return invalid("description is required.")
	case w.Status != models.BookDraft && w.Status != models.BookPublished:
		return invalid("status must be draft or published.")
	}

	price, err := strconv.Atoi(strings.TrimSpace(in.Price))
	if err != nil || price < 0 {
		return invalid("price must be a whole number of 0 or more.")
	}
	w.Price = price

	minAge, err := strconv.Atoi(strings.TrimSpace(in.AgeMin))
	if err != nil || minAge < 0 {
		return badAge("age_min must be a whole number of 0 or more.")
	}
	w.AgeMin = minAge
	switch in.AgeType {
	case models.AgeRange:
		maxAge, err := strconv.Atoi(strings.TrimSpace(in.AgeMax))
		if err != nil {
			return badAge("age_max must be a whole number.")
		}
		if minAge >= maxAge {
			return badAge("age_min must be lower than age_max.")
		}
		w.AgeMax = &maxAge
	case models.AgePlus:
		if strings.TrimSpace(in.AgeMax) != "" {
			return badAge("age_max is only allowed when age_type is range.")
		}
	default:
		return badAge("age_type must be range or plus.")
	}

	if in.Cover != nil {
		mime, ie := checkCover(in.Cover)
		if ie != nil {
			return library.BookWrite{}, ie
		}
		w.Cover = &models.BookFile{MimeType: mime, FileName: filepath.Base(in.Cover.FileName), Content: in.Cover.Content}
	}
	if in.Epub != nil {
		if ie := checkEpub(in.Epub); ie != nil {
			return library.BookWrite{}, ie
		}
		name := filepath.Base(in.Epub.FileName)
		w.EpubFileName, w.Epub = &name, in.Epub.Content
	}
	return w, nil
}

// checkCover accepts .jpg/.jpeg/.png whose bytes really are JPEG/PNG.
func checkCover(u *Upload) (string, *inputError) {
	ext := strings.ToLower(filepath.Ext(u.FileName))
	mime := http.DetectContentType(u.Content)
	if (ext != ".jpg" && ext != ".jpeg" && ext != ".png") || (mime != "image/jpeg" && mime != "image/png") {
		return "", &inputError{respConst.INVALID_FILE_FORMAT, "cover must be a .jpg, .jpeg or .png image."}
	}
	if len(u.Content) > MaxCoverBytes {
		return "", &inputError{respConst.VALIDATION_ERROR, fmt.Sprintf("cover must be at most %d MB.", MaxCoverBytes>>20)}
	}
	return mime, nil
}

// checkEpub accepts .epub files that are ZIP containers (every EPUB is).
func checkEpub(u *Upload) *inputError {
	if strings.ToLower(filepath.Ext(u.FileName)) != ".epub" || !bytes.HasPrefix(u.Content, []byte("PK\x03\x04")) {
		return &inputError{respConst.INVALID_FILE_FORMAT, "book file must be an .epub file."}
	}
	if len(u.Content) > MaxEpubBytes {
		return &inputError{respConst.VALIDATION_ERROR, fmt.Sprintf("book file must be at most %d MB.", MaxEpubBytes>>20)}
	}
	return nil
}

// AgeLabel renders "1–3" for a range and "3+" for plus (PRD US-16).
func AgeLabel(ageType string, minAge int, maxAge *int) string {
	if ageType == models.AgeRange && maxAge != nil {
		return fmt.Sprintf("%d–%d", minAge, *maxAge)
	}
	return fmt.Sprintf("%d+", minAge)
}
