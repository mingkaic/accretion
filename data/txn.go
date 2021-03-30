package data

import (
	"context"
	"time"

	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	log "github.com/sirupsen/logrus"
)

type (
	// dgo transactions aren't thread-safe, so store in array until
	// commit/discard where array is cleared
	Txn struct {
		client *dgo.Dgraph
		txs    []*dgo.Txn
	}
)

func WithTx(txFn func(*Txn) error) (err error) {
	dg, cancel := getDgraphClient()
	defer cancel()
	tx := &Txn{
		client: dg,
		txs:    make([]*dgo.Txn, 0),
	}

	err = txFn(tx)
	if err != nil {
		log.Debugf("Transaction failed (discarding: %+v", err)
		tx.Discard()
	} else {
		tx.Commit()
	}
	return
}

func (tx *Txn) Mutate(mu *api.Mutation) (*api.Response, error) {
	tranx := tx.client.NewTxn()
	tx.txs = append(tx.txs, tranx)
	ctx := context.Background()
	clientDeadline := time.Now().Add(time.Duration(500) * time.Second)
	ctx, cancel := context.WithDeadline(ctx, clientDeadline)
	defer cancel()
	return tranx.Mutate(ctx, mu)
}

func (tx *Txn) Commit() error {
	defer tx.clear()
	for _, t := range tx.txs {
		ctx := context.Background()
		if err := t.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (tx *Txn) Discard() error {
	defer tx.clear()
	for _, t := range tx.txs {
		ctx := context.Background()
		if err := t.Discard(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (tx *Txn) clear() {
	tx.txs = make([]*dgo.Txn, 0)
}
