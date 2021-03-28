package data

import (
	"context"
	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"time"
)

type (
	Txn struct {
		*dgo.Txn
	}
)

func WithTx(txFn func(Txn) error) (err error) {
	dg, cancel := getDgraphClient()
	defer cancel()
	tx := Txn{dg.NewTxn()}

	err = txFn(tx)
	if err != nil {
		tx.Discard()
	} else {
		tx.Commit()
	}
	return
}

func (tx Txn) Mutate(mu *api.Mutation) (*api.Response, error) {
	ctx := context.Background()
	clientDeadline := time.Now().Add(time.Duration(500) * time.Second)
	ctx, cancel := context.WithDeadline(ctx, clientDeadline)
	defer cancel()
	return tx.Txn.Mutate(ctx, mu)
}

func (tx Txn) Commit() error {
	ctx := context.Background()
	return tx.Txn.Commit(ctx)
}

func (tx Txn) Discard() error {
	ctx := context.Background()
	return tx.Txn.Discard(ctx)
}
