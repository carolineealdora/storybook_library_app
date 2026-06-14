// Package sql provides the ORM interface and supporting types for database access.
// The package name is "sql" (not "orm") so callers use sql.ORM, sql.Query, etc.
package sql

import (
	"context"

	"codebase/pkg/monitor"
)

type Query struct {
	Field    string
	Operator string
	Value    interface{}
}

type Joins struct {
	Table     string
	Condition string
	Type      string
}

type Sort struct {
	Field string
	Order string
}

type Sorts []Sort

type OnConflict struct {
	Columns   []string
	UpdateAll bool
	DoNothing bool
}

type Connection struct {
	Host     string
	Username string
	Password string
	DBName   string
	Port     string
	SSLMode  string
	Timezone string
	SQLDebug bool
	ORMDebug bool
}

type ORM interface {
	Where(condition interface{}, args ...interface{}) ORM
	OrWhere(query interface{}, args ...interface{}) ORM
	First(result interface{}) error
	All(result interface{}) (int64, error)
	Order(order interface{}) ORM
	Limit(limit int) ORM
	Offset(offset int) ORM
	Create(data interface{}) error
	Update(data interface{}) error
	Patch(data interface{}, fields ...string) error
	Delete(data interface{}, args ...interface{}) error
	Upsert(chunkSize int, data interface{}, onConflict OnConflict) error
	Exec(query string, args ...interface{}) error
	RawSql(query string, result interface{}, args ...interface{}) error
	FindByQuery(tableName string, selectFields []string, where []Query, result interface{}) ORM
	FillQuery(tableName string, query []Query) ORM
	Count() int64
	Select(fields []string) ORM
	Sort(sorts Sorts) ORM
	Join(joins Joins) ORM
	Group(fields []string) ORM
	HasTable(tableName string) bool
	CreateTable(data interface{}) error
	CreateTableWithName(name string, data interface{}) error
	DropTable(data interface{}) error
	DropTableWithName(name string, data interface{}) error
	HasColumn(data interface{}, column string) bool
	AddColumn(data interface{}, column string) error
	DropColumn(data interface{}, column string) error
	AlterColumn(data interface{}, column string) error
	Table(tableName string) ORM
	Begin() ORM
	Commit() error
	Rollback() error
	Ping() error
	Close() error
	Error() error
	RowsAffected() int64
	Monitor(ctx context.Context, mntr monitor.Monitor, requestId string, captureError bool) ORM
}
