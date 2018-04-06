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

package snapshot

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	snapshotFilenameSuffix = "etcd.backup"
)

type Metadata struct {
	Name     string
	Revision int64
	Size     int64

	Source Provider
}

func NewMetadata(name string, revision int64, size int64, source Provider) (*Metadata, error) {
	if revision < 0 {
		var err error
		revision, err = strconv.ParseInt(strings.SplitN(name, "_", 3)[1], 16, 64)
		if err != nil {
			return nil, err
		}
	}
	return &Metadata{
		Name:     name,
		Revision: revision,
		Size:     size,
		Source:   source,
	}, nil
}

func (m *Metadata) Filename() string {
	return fmt.Sprintf("%s_%016x_%s", m.Name, m.Revision, snapshotFilenameSuffix)
}

func (m *Metadata) IsValid() bool {
	return strings.HasSuffix(m.Name, snapshotFilenameSuffix)
}

type MetadataSorter []*Metadata

func (ms MetadataSorter) Len() int           { return len(ms) }
func (ms MetadataSorter) Swap(i, j int)      { ms[i], ms[j] = ms[j], ms[i] }
func (ms MetadataSorter) Less(i, j int) bool { return ms[i].Revision < ms[j].Revision }
