package monitor

import "context"

type Tag struct {
	Key   string
	Value string
}

type Tick struct {
	Operation       string
	TransactionName string
	Tags            []Tag
}

type Transaction interface {
	FinishWithTags(tags []Tag)
}

type Monitor interface {
	NewTransactionFromContext(ctx context.Context, tick Tick) Transaction
	Capture(err error)
}
