package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	doOnce sync.Once
)

const (
	dbUrl        = "127.0.0.1:9080"
	dbUser       = "groot"
	dbPwd        = "password"
	dbSchemaFile = "data/schema.dql"
	enabledAcl   = false
)

func init() {
	doOnce.Do(func() {
		dg, cancel := getDgraphClient()
		defer cancel()

		log.Info("Publishing schema")
		b, err := ioutil.ReadFile(dbSchemaFile)
		if err != nil {
			log.Fatal(err)
		}
		op := &api.Operation{}
		op.Schema = string(b)
		ctx := context.Background()
		if err := dg.Alter(ctx, op); err != nil {
			log.Fatal(err)
		}
	})
}

func getDgraphClient() (*dgo.Dgraph, func()) {
	conn, err := grpc.Dial(dbUrl, grpc.WithInsecure())
	if err != nil {
		log.Fatal("While trying to dial gRPC")
	}

	dc := api.NewDgraphClient(conn)
	dg := dgo.NewDgraphClient(dc)

	if enabledAcl {
		// Perform login call. If the Dgraph cluster does not have ACL and
		// enterprise features enabled, this call should be skipped.
		ctx := context.Background()
		for {
			// Keep retrying until we succeed or receive a non-retriable error.
			err = dg.Login(ctx, dbUser, dbPwd)
			if err == nil || !strings.Contains(err.Error(), "Please retry") {
				break
			}
			time.Sleep(time.Second)
		}
		if err != nil {
			log.Fatalf("While trying to login %v", err.Error())
		}
	}

	return dg, func() {
		if err := conn.Close(); err != nil {
			log.Infof("Error while closing connection:%v", err)
		}
	}
}

func CreateNode(tx Txn, node interface{}) error {
	mu := &api.Mutation{}
	pb, err := json.Marshal(node)
	if err != nil {
		return err
	}
	log.Debugf("creating dgraph node with %s", pb[:50])
	mu.SetJson = pb
	_, err = tx.Mutate(mu)
	if err != nil {
		return err
	}
	return nil
}

func AsyncCreateNode(wg *sync.WaitGroup, errChan chan error, id string, tx Txn, node interface{}) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := CreateNode(tx, node)
		if err != nil {
			errChan <- fmt.Errorf("Job %s failed: %+v", id, err)
		}
	}()
}
