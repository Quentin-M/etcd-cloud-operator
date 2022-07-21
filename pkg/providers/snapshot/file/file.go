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
	"sort"
	"time"

	"go.uber.org/zap"

	"github.com/quentin-m/etcd-cloud-operator/pkg/providers"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/snapshot"
)

const (
	filePermissions     = 0600
	directoryPermission = 0700
)

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
	if err := os.MkdirAll(f.config.Dir, directoryPermission); err != nil {
		return fmt.Errorf("invalid configuration: failed to create directory %q: %v", f.config.Dir, err)
	}
	return nil
}

func (f *file) Save(r io.ReadCloser, metadata *snapshot.Metadata) error {
	tmpF, err := ioutil.TempFile(f.config.Dir, metadata.Filename())
	if err != nil {
		return err
	}

	n, err := io.Copy(tmpF, r)
	if err != nil {
		tmpF.Close()
		os.Remove(tmpF.Name())
		return err
	}

	tmpF.Sync()
	tmpF.Close()

	fpath := filepath.Join(f.config.Dir, metadata.Filename())
	if err = os.Rename(tmpF.Name(), fpath); err != nil {
		os.Remove(tmpF.Name())
		return err
	}
	os.Chmod(fpath, filePermissions)

	metadata.Size = n
	return nil
}

func (f *file) Info() (*snapshot.Metadata, error) {
	files, err := ioutil.ReadDir(f.config.Dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list dir: %s", err)
	}

	var metadatas []*snapshot.Metadata
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		metadata, err := snapshot.NewMetadata(file.Name(), -1, file.Size(), f)
		if err != nil {
			zap.S().Warnf("failed to parse metadata for snapshot %v", file.Name())
			continue
		}
		metadatas = append(metadatas, metadata)
	}
	if len(metadatas) == 0 {
		return nil, snapshot.ErrNoSnapshot
	}
	sort.Sort(snapshot.MetadataSorter(metadatas))

	return metadatas[len(metadatas)-1], nil
}

func (f *file) Get(metadata *snapshot.Metadata) (string, bool, error) {
	return filepath.Join(f.config.Dir, metadata.Name), false, nil
}

func (f *file) Purge(ttl time.Duration) error {
	files, err := ioutil.ReadDir(f.config.Dir)
	if err != nil {
		return fmt.Errorf("failed to list dir: %s", err)
	}

	for _, file := range files {
		if time.Since(file.ModTime()) > ttl {
			zap.S().Infof("purging snapshot file %q because it is that older than %v", file.Name(), ttl)
			os.Remove(filepath.Join(f.config.Dir, file.Name()))
		}
	}
	return nil
}
