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
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	etcdcl "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/embed"
)

var ErrMemberRevisionTooOld = errors.New("member revision older than the minimum desired revision")

func Snapshot(address string, sc SecurityConfig, minRev int64) (io.ReadCloser, func(), int64, error) {
	tc, err := sc.ClientConfig()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to read client transport security: %s", err)
	}

	c, err := newClient([]string{clientURL(address, sc.TLSEnabled())}, tc)
	if err != nil {
		return nil, nil, 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	resp, err := c.Get(ctx, "/", etcdcl.WithSerializable())
	cancel()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to get revision from member: %s", err)
	}

	if resp.Header.Revision <= minRev {
		return nil, nil, resp.Header.Revision, ErrMemberRevisionTooOld
	}

	ctx, cancel = context.WithTimeout(context.Background(), defaultRequestTimeout)
	rc, err := c.Snapshot(ctx)
	if err != nil {
		cancel()
		c.Close()
		return nil, nil, 0, fmt.Errorf("failed to receive snapshot (%v)", err)
	}

	close := func() {
		cancel()
		rc.Close()
		c.Close()
	}

	return rc, close, resp.Header.Revision, nil
}

func restore(name, dataDir, privateAddress string, peerSC SecurityConfig, rc io.ReadCloser) error {
	snapshotFile, err := writeTmpFile(rc)
	if err != nil {
		return fmt.Errorf("failed to save snapshot to disk: %v", err)
	}

	cmd := exec.Command("/bin/sh", "-ec",
		fmt.Sprintf("ETCDCTL_API=3 etcdctl snapshot restore %[1]s"+
			" --name %[2]s"+
			" --initial-cluster %[2]s=%[3]s"+
			" --initial-cluster-token %[4]s"+
			" --initial-advertise-peer-urls %[3]s"+
			" --data-dir %[5]s",
			snapshotFile, name, peerURL(privateAddress, peerSC.TLSEnabled()),
			embed.NewConfig().InitialClusterToken, dataDir,
		),
	)

	if _, err := cmd.CombinedOutput(); err != nil {
		return errors.New("etcdctl failed to restore")
	}
	return nil
}

func writeTmpFile(rc io.ReadCloser) (string, error) {
	defer rc.Close()

	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}

	_, err = io.Copy(f, rc)
	if err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	f.Close()

	return f.Name(), nil
}
