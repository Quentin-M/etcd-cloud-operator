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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	snapshotFilenameSuffix = "etcd.backup"
)

var ErrNoSnapshot = errors.New("no snapshot available")

func Name(rev int64, name string) string {
	return fmt.Sprintf("%s_%016x_%s", name, rev, snapshotFilenameSuffix)
}

func LatestFromNames(names []string) (name string, rev int64, err error) {
	for _, n := range names {
		if !IsSnapshot(n) {
			continue
		}

		r, err := strconv.ParseInt(strings.SplitN(n, "_", 3)[1], 16, 64)
		if err != nil {
			logrus.WithError(err).Warnf("failed to parse revision from backup %q", n)
			continue
		}

		if r > rev {
			rev = r
			name = n
		}
	}

	if name == "" {
		err = ErrNoSnapshot
	}
	return
}

func IsSnapshot(name string) bool {
	return strings.HasSuffix(name, snapshotFilenameSuffix)
}
