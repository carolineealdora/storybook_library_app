package library

import (
	"codebase/models"
)

// Never select epub_content here: list/detail queries must stay small (PRD §15).
const bookCols = `b.id, b.title, b.author, b.description, b.age_type, b.age_min, b.age_max, b.price, b.status,
	b.epub_file_name, b.epub_size_bytes, b.created_at, b.updated_at, b.deleted_at`

type BookWrite struct {
	Title, Author, Description, AgeType, Status string
	AgeMin                                      int
	AgeMax                                      *int
	Price                                       int
	EpubFileName                                *string // nil on update = keep current EPUB
	Epub                                        []byte
	Cover                                       *models.BookFile // nil on update = keep current cover
}

// ListPublishedBooks returns the showcase with ownership for userID, newest first.
func (r *Repo) ListPublishedBooks(userID string, limit, offset int) ([]models.Book, int64, error) {
	return page[models.Book](r.db, limit, offset, `SELECT `+bookCols+`, (p.id IS NOT NULL) AS owned
		FROM books b LEFT JOIN purchases p ON p.book_id = b.id AND p.user_id = ?
		WHERE b.status = 'published' AND b.deleted_at IS NULL
		ORDER BY b.created_at DESC, b.id`, userID)
}

// GetBookForUser returns the book (even if draft/deleted) with ownership for userID.
func (r *Repo) GetBookForUser(bookID, userID string) (*models.Book, error) {
	var b models.Book
	err := r.db.RawSql(`SELECT `+bookCols+`, (p.id IS NOT NULL) AS owned
		FROM books b LEFT JOIN purchases p ON p.book_id = b.id AND p.user_id = ?
		WHERE b.id = ?`, &b, userID, bookID)
	return one(&b, b.ID, err)
}

func (r *Repo) ListPurchasedBooks(userID string, limit, offset int) ([]models.Book, int64, error) {
	return page[models.Book](r.db, limit, offset, `SELECT `+bookCols+`, true AS owned, p.purchased_at
		FROM purchases p JOIN books b ON b.id = p.book_id
		WHERE p.user_id = ?
		ORDER BY p.purchased_at DESC, b.id`, userID)
}

func (r *Repo) GetCover(bookID string) (*models.BookFile, error) {
	var f models.BookFile
	err := r.db.RawSql(`SELECT mime_type, file_name, content FROM book_files WHERE book_id = ?`, &f, bookID)
	return one(&f, f.MimeType, err)
}

func (r *Repo) GetEpub(bookID string) (*models.BookFile, error) {
	var f models.BookFile
	err := r.db.RawSql(`SELECT 'application/epub+zip' AS mime_type, epub_file_name AS file_name, epub_content AS content
		FROM books WHERE id = ?`, &f, bookID)
	return one(&f, f.MimeType, err)
}

func (r *Repo) upsertCover(bookID string, cover *models.BookFile) error {
	if cover == nil {
		return nil
	}
	return r.db.Exec(`INSERT INTO book_files (book_id, mime_type, file_name, size_bytes, content) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (book_id) DO UPDATE SET mime_type = EXCLUDED.mime_type, file_name = EXCLUDED.file_name,
			size_bytes = EXCLUDED.size_bytes, content = EXCLUDED.content, updated_at = now()`,
		bookID, cover.MimeType, cover.FileName, len(cover.Content), cover.Content)
}
