package wrap_test

import (
	"context"

	"codebase/pkg/monitor"
	sql "codebase/pkg/orm"
)

type ORMWrapperMock interface {
	Where(condition interface{}, args ...interface{}) sql.ORM
	OrWhere(query interface{}, args ...interface{}) sql.ORM
	First(result interface{}) error
	All(result interface{}) (int64, error)
	Order(order interface{}) sql.ORM
	Limit(limit int) sql.ORM
	Offset(offset int) sql.ORM
	Create(data interface{}) error
	Update(data interface{}) error
	Patch(data interface{}, fields ...string) error
	Delete(data interface{}, args ...interface{}) error
	Upsert(chunkSize int, data interface{}, onConflict sql.OnConflict) error
	Exec(query string, args ...interface{}) error
	RawSql(query string, result interface{}, args ...interface{}) error
	FindByQuery(query string, selectFields []string, where []sql.Query, result interface{}) sql.ORM
	FillQuery(tableName string, query []sql.Query) sql.ORM
	Count() int64
	Select(fields []string) sql.ORM
	Sort(sorts sql.Sorts) sql.ORM
	Join(joins sql.Joins) sql.ORM
	Group(fields []string) sql.ORM
	HasTable(tableName string) bool
	CreateTable(data interface{}) error
	CreateTableWithName(name string, data interface{}) error
	DropTable(data interface{}) error
	DropTableWithName(name string, data interface{}) error
	HasColumn(data interface{}, column string) bool
	AddColumn(data interface{}, column string) error
	DropColumn(data interface{}, column string) error
	AlterColumn(data interface{}, column string) error
	Table(tableName string) sql.ORM
	Begin() sql.ORM
	Commit() error
	Rollback() error
	Ping() error
	Close() error
	Error() error
	RowsAffected() int64
	Monitor(ctx context.Context, mntr monitor.Monitor, requestId string, captureError bool) sql.ORM
}

// ORMWrapper adalah wrapper untuk interface ORM
type ORMWrapper struct {
	orm sql.ORM
}

// NewORMWrapper membuat instansi ORMWrapper
func NewORMWrapper(orm sql.ORM) *ORMWrapper {
	return &ORMWrapper{orm: orm}
}

// Implementasikan semua method dari ORM dengan meneruskan ke objek ORM yang diberikan
func (w *ORMWrapper) Where(condition interface{}, args ...interface{}) sql.ORM {
	return w.orm.Where(condition, args...)
}

func (w *ORMWrapper) OrWhere(query interface{}, args ...interface{}) sql.ORM {
	return w.orm.OrWhere(query, args...)
}

func (w *ORMWrapper) First(result interface{}) error {
	return w.orm.First(result)
}

func (w *ORMWrapper) All(result interface{}) (int64, error) {
	return w.orm.All(result)
}

func (w *ORMWrapper) Order(order interface{}) sql.ORM {
	return w.orm.Order(order)
}

func (w *ORMWrapper) Limit(limit int) sql.ORM {
	return w.orm.Limit(limit)
}

func (w *ORMWrapper) Offset(offset int) sql.ORM {
	return w.orm.Offset(offset)
}

func (w *ORMWrapper) Create(data interface{}) error {
	return w.orm.Create(data)
}

func (w *ORMWrapper) Update(data interface{}) error {
	return w.orm.Update(data)
}

func (w *ORMWrapper) Patch(data interface{}, fields ...string) error {
	return w.orm.Patch(data, fields...)
}

func (w *ORMWrapper) Delete(data interface{}, args ...interface{}) error {
	return w.orm.Delete(data, args...)
}

func (w *ORMWrapper) Upsert(chunkSize int, data interface{}, onConflict sql.OnConflict) error {
	return w.orm.Upsert(chunkSize, data, onConflict)
}

func (w *ORMWrapper) Exec(query string, args ...interface{}) error {
	return w.orm.Exec(query, args...)
}

func (w *ORMWrapper) RawSql(query string, result interface{}, args ...interface{}) error {
	return w.orm.RawSql(query, result, args...)
}

func (w *ORMWrapper) FindByQuery(query string, selectFields []string, where []sql.Query, result interface{}) sql.ORM {
	return w.orm.FindByQuery(query, selectFields, where, result)
}

func (w *ORMWrapper) FillQuery(tableName string, query []sql.Query) sql.ORM {
	return w.orm.FillQuery(tableName, query)
}

func (w *ORMWrapper) Count() int64 {
	return w.orm.Count()
}

func (w *ORMWrapper) Select(fields []string) sql.ORM {
	return w.orm.Select(fields)
}

func (w *ORMWrapper) Sort(sorts sql.Sorts) sql.ORM {
	return w.orm.Sort(sorts)
}

func (w *ORMWrapper) Join(joins sql.Joins) sql.ORM {
	return w.orm.Join(joins)
}

func (w *ORMWrapper) Group(fields []string) sql.ORM {
	return w.orm.Group(fields)
}

func (w *ORMWrapper) HasTable(tableName string) bool {
	return w.orm.HasTable(tableName)
}

func (w *ORMWrapper) CreateTable(data interface{}) error {
	return w.orm.CreateTable(data)
}

func (w *ORMWrapper) CreateTableWithName(name string, data interface{}) error {
	return w.orm.CreateTableWithName(name, data)
}

func (w *ORMWrapper) DropTable(data interface{}) error {
	return w.orm.DropTable(data)
}

func (w *ORMWrapper) DropTableWithName(name string, data interface{}) error {
	return w.orm.DropTableWithName(name, data)
}

func (w *ORMWrapper) HasColumn(data interface{}, column string) bool {
	return w.orm.HasColumn(data, column)
}

func (w *ORMWrapper) AddColumn(data interface{}, column string) error {
	return w.orm.AddColumn(data, column)
}

func (w *ORMWrapper) DropColumn(data interface{}, column string) error {
	return w.orm.DropColumn(data, column)
}

func (w *ORMWrapper) AlterColumn(data interface{}, column string) error {
	return w.orm.AlterColumn(data, column)
}

func (w *ORMWrapper) Table(tableName string) sql.ORM {
	return w.orm.Table(tableName)
}

func (w *ORMWrapper) Begin() sql.ORM {
	return w.orm.Begin()
}

func (w *ORMWrapper) Commit() error {
	return w.orm.Commit()
}

func (w *ORMWrapper) Rollback() error {
	return w.orm.Rollback()
}

func (w *ORMWrapper) Ping() error {
	return w.orm.Ping()
}

func (w *ORMWrapper) Close() error {
	return w.orm.Close()
}

func (w *ORMWrapper) Error() error {
	return w.orm.Error()
}

func (w *ORMWrapper) RowsAffected() int64 {
	return w.orm.RowsAffected()
}

func (w *ORMWrapper) Monitor(ctx context.Context, mntr monitor.Monitor, requestId string, captureError bool) sql.ORM {
	return w.orm.Monitor(ctx, mntr, requestId, captureError)
}
