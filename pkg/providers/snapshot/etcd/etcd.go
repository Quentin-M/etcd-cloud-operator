// Copyright 2017 Quentin Machu & eco authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package etcd

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/coreos/bbolt"

	"github.com/quentin-m/etcd-cloud-operator/pkg/providers"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/snapshot"
)

func init() {
	snapshot.Register("etcd", &etcd{})
}

type etcd struct {
	config config
}

type config struct {
	DataDir string `yaml:"data-dir"`
}

func (f *etcd) Configure(providerConfig snapshot.Config) error {
	f.config = config{DataDir: "/var/lib/etcd"}
	if err := providers.ParseParams(providerConfig.Params, &f.config); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}
	return nil
}

func (f *etcd) Save(r io.ReadCloser, metadata *snapshot.Metadata) error {
	panic("not implemented")
}

// https://github.com/coreos/etcd/blob/master/snapshot/v3_snapshot.go
func (f *etcd) Info() (*snapshot.Metadata, error) {
	dbPath := filepath.Join(f.config.DataDir, "member/snap/db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, snapshot.ErrNoSnapshot
	}

	db, err := bbolt.Open(dbPath, 0400, &bbolt.Options{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var revision int64
	var size int64
	err = db.View(func(tx *bbolt.Tx) error {
		size = tx.Size()
		c := tx.Cursor()
		for next, _ := c.First(); next != nil; next, _ = c.Next() {
			b := tx.Bucket(next)
			if b == nil {
				return fmt.Errorf("cannot get hash of bucket %s", string(next))
			}
			b.ForEach(func(k, v []byte) error {
				if string(next) == "key" {
					revision = int64(binary.BigEndian.Uint64(k[0:8]))
				}
				return nil
			})
		}
		return nil
	})

	return snapshot.NewMetadata(dbPath, revision, size, f)
}

func (f *etcd) Get(metadata *snapshot.Metadata) (string, bool, error) {
	in, err := os.Open(metadata.Name)
	if err != nil {
		return "", true, err
	}
	defer in.Close()

	dst, err := ioutil.TempFile("", "")
	if err != nil {
		return "", true, err
	}

	_, err = io.Copy(dst, in)
	if err != nil {
		dst.Close()
		os.Remove(dst.Name())
		return "", true, err
	}

	dst.Sync()
	dst.Close()

	return dst.Name(), true, err
}

func (f *etcd) Purge(ttl time.Duration) error {
	panic("not implemented")
}
