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

package file

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/quentin-m/etcd-cloud-operator/pkg/providers"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/snapshot"
	log "github.com/sirupsen/logrus"
)

const filePermissions = 0600

func init() {
	snapshot.Register("file", &file{})
}

type file struct {
	config config
}

type config struct {
	Dir string `yaml:"dir"`
}

func (f *file) Configure(providerConfig snapshot.Config) error {
	f.config = config{Dir: "/var/lib/snapshots"}
	if err := providers.ParseParams(providerConfig.Params, &f.config); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}
	if err := os.MkdirAll(f.config.Dir, filePermissions); err != nil {
		return fmt.Errorf("invalid configuration: failed to create directory %q: %v", f.config.Dir, err)
	}
	return nil
}

func (f *file) Save(r io.ReadCloser, name string, rev int64) (int64, error) {
	fname := snapshot.Name(rev, name)
	fpath := filepath.Join(f.config.Dir, fname)

	tmpF, err := ioutil.TempFile("", fname)
	if err != nil {
		return 0, err
	}

	n, err := io.Copy(tmpF, r)
	if err != nil {
		tmpF.Close()
		os.Remove(tmpF.Name())
		return 0, err
	}

	tmpF.Sync()
	tmpF.Close()

	if err = os.Rename(tmpF.Name(), fpath); err != nil {
		os.Remove(tmpF.Name())
		return 0, err
	}
	os.Chmod(fpath, filePermissions)

	return n, nil
}

func (f *file) Latest() (io.ReadCloser, int64, int64, error) {
	path, rev, err := f.latestSnapshotFileName()
	if err != nil {
		return nil, 0, 0, err
	}

	rc, err := os.Open(filepath.Join(f.config.Dir, path))
	if err != nil {
		return nil, 0, 0, err
	}

	st, _ := rc.Stat()
	return rc, st.Size(), rev, err
}

func (f *file) LatestRev() (int64, error) {
	_, rev, err := f.latestSnapshotFileName()
	return rev, err
}

func (f *file) Purge(ttl time.Duration) error {
	files, err := ioutil.ReadDir(f.config.Dir)
	if err != nil {
		return fmt.Errorf("failed to list dir: %s", err)
	}

	for _, file := range files {
		if time.Since(file.ModTime()) > ttl {
			log.Infof("purging snapshot file %q because it is that older than %v", file.Name(), ttl)
			os.Remove(filepath.Join(f.config.Dir, file.Name()))
		}
	}
	return nil
}

func (f *file) latestSnapshotFileName() (string, int64, error) {
	files, err := ioutil.ReadDir(f.config.Dir)
	if err != nil {
		return "", 0, fmt.Errorf("failed to list dir: %s", err)
	}

	var names []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		names = append(names, f.Name())
	}

	return snapshot.LatestFromNames(names)
}
