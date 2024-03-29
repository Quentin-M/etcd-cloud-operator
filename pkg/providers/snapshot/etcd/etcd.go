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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	etcdsnap "go.etcd.io/etcd/etcdutl/v3/snapshot"
	"go.uber.org/zap"

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

func (f *etcd) Info() (*snapshot.Metadata, error) {
	dbPath := filepath.Join(f.config.DataDir, "member/snap/db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, snapshot.ErrNoSnapshot
	}

	status, err := etcdsnap.NewV3(zap.L()).Status(dbPath)
	if err != nil {
		return nil, err
	}

	return snapshot.NewMetadata(dbPath, status.Revision, status.TotalSize, f)
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
