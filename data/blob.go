package data

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/mingkaic/accretion/proto/storage"
	log "github.com/sirupsen/logrus"
)

const storageDir = "blobs"

func SaveBlob(profileId, id string, blob *storage.BlobStorage) error {
	dir := path.Join(storageDir, profileId)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}
	fname := path.Join(dir, id)
	b, err := proto.Marshal(blob)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(fname, b, 0644); err != nil {
		return err
	}
	return nil
}

func AsyncSaveBlob(wg *sync.WaitGroup, errChan chan error, profileId, id string, blob *storage.BlobStorage) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := SaveBlob(profileId, id, blob)
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
