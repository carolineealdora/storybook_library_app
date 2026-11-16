// Package library is the single Postgres repository for the storybook apps.
// ponytail: one repo instead of one-per-table, because payments need transactions that span
// orders, purchases and subscriptions. Split it if a domain ever moves to its own store.
package library

import (
	"errors"

	"codebase/models"
	sql "codebase/pkg/orm"

	"github.com/jackc/pgconn"
)

type Repo struct {
	db sql.ORM
}

func NewRepository(holder *models.SharedHolder) *Repo {
	return &Repo{db: holder.Sql}
}

// InTx runs fn in one DB transaction. fn gets a Repo bound to the transaction;
// returning an error rolls back.
func (r *Repo) InTx(fn func(tx *Repo) error) error {
	tx := r.db.Begin()
	if err := tx.Error(); err != nil {
		return err
	}
	if err := fn(&Repo{db: tx}); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Join(err, rbErr)
		}
		return err
	}
	return tx.Commit()
}

// IsUniqueViolation reports a Postgres unique-constraint error (SQLSTATE 23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// page runs query (which must end in ORDER BY) for one page and counts all of its rows.
// ponytail: two queries per list, fine at catalogue size. Keyset paging if lists get big.
func page[T any](db sql.ORM, limit, offset int, query string, args ...any) ([]T, int64, error) {
	var count struct{ N int64 }
	if err := db.RawSql(`SELECT count(*) AS n FROM (`+query+`) q`, &count, args...); err != nil {
		return nil, 0, err
	}
	rows := []T{}
	err := db.RawSql(query+` LIMIT ? OFFSET ?`, &rows, append(args, limit, offset)...)
	return rows, count.N, err
}

// one returns v when the scanned row exists (id set), nil when no row matched.
func one[T any](v *T, id string, err error) (*T, error) {
	if err != nil || id == "" {
		return nil, err
	}
	return v, nil
}
