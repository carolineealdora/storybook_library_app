package library

import (
	"strings"

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

// ListBooks is the admin list: every non-deleted book, optional title search and status filter.
func (r *Repo) ListBooks(search, status string, limit, offset int) ([]models.Book, int64, error) {
	like := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search)
	return page[models.Book](r.db, limit, offset, `SELECT `+bookCols+` FROM books b
		WHERE b.deleted_at IS NULL
		  AND (?::text = '' OR b.title ILIKE '%' || ?::text || '%')
		  AND (?::text = '' OR b.status = ?::text)
		ORDER BY b.created_at DESC, b.id`, like, like, status, status)
}

func (r *Repo) GetBook(bookID string) (*models.Book, error) {
	var b models.Book
	err := r.db.RawSql(`SELECT `+bookCols+` FROM books b WHERE b.id = ? AND b.deleted_at IS NULL`, &b, bookID)
	return one(&b, b.ID, err)
}

func (r *Repo) CreateBook(w BookWrite) (string, error) {
	var id string
	err := r.db.RawSql(`INSERT INTO books (title, author, description, age_type, age_min, age_max, price, status,
			epub_file_name, epub_size_bytes, epub_content)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`, &id,
		w.Title, w.Author, w.Description, w.AgeType, w.AgeMin, w.AgeMax, w.Price, w.Status,
		*w.EpubFileName, len(w.Epub), w.Epub)
	if err != nil {
		return "", err
	}
	return id, r.upsertCover(id, w.Cover)
}

// UpdateBook replaces metadata; files are replaced only when provided. Returns false if no such book.
func (r *Repo) UpdateBook(bookID string, w BookWrite) (bool, error) {
	var id string
	err := r.db.RawSql(`UPDATE books SET title = ?, author = ?, description = ?, age_type = ?, age_min = ?, age_max = ?,
			price = ?, status = ?, updated_at = now(),
			epub_file_name = COALESCE(?, epub_file_name),
			epub_size_bytes = CASE WHEN ?::bytea IS NULL THEN epub_size_bytes ELSE ? END,
			epub_content = COALESCE(?::bytea, epub_content)
		WHERE id = ? AND deleted_at IS NULL RETURNING id`, &id,
		w.Title, w.Author, w.Description, w.AgeType, w.AgeMin, w.AgeMax, w.Price, w.Status,
		w.EpubFileName, w.Epub, len(w.Epub), w.Epub, bookID)
	if err != nil || id == "" {
		return false, err
	}
	return true, r.upsertCover(id, w.Cover)
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

// SoftDeleteBook hides the book from the showcase; owners keep it (PRD US-16, E5).
func (r *Repo) SoftDeleteBook(bookID string) (bool, error) {
	var id string
	err := r.db.RawSql(`UPDATE books SET deleted_at = now(), updated_at = now()
		WHERE id = ? AND deleted_at IS NULL RETURNING id`, &id, bookID)
	return id != "", err
}
