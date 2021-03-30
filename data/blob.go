package data

import (
    "os"
    "fmt"
    "path"
    "io/ioutil"
	"sync"

    "github.com/mingkaic/accretion/proto/storage"
    "github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
)

const storageDir = "blobs"

func SaveBlob(id string, blob *storage.BlobStorage) error {
    fname := path.Join(storageDir, id)
    b, err := proto.Marshal(blob)
    if err != nil {
        return err
    }
    if err := ioutil.WriteFile(fname, b, 0644); err != nil {
        return err
    }
    return nil
}

func AsyncSaveBlob(wg *sync.WaitGroup, errChan chan error, id string, blob *storage.BlobStorage) {
    wg.Add(1)
    go func() {
        defer wg.Done()
        err := SaveBlob(id, blob)
        if err != nil {
            errChan <- fmt.Errorf("Save Job %s failed: %+v", id, err)
        }
    }()
}

func initBlob() {
    err := os.MkdirAll(storageDir, os.ModePerm)
    if err != nil {
        log.Error(err)
    }
}
