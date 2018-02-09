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
	"fmt"
	"io"
	"os"
	"time"

	etcdcl "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/coreos/etcd/embed"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/coreos/etcd/pkg/types"
	log "github.com/sirupsen/logrus"
)

const (
	defaultStartTimeout          = 120 * time.Second
	defaultStartHealthyThreshold = 10 * time.Second
)

func SeedCluster(name, dataDir string, dataQuota int64, publicAddress, privateAddress string, clientSC, peerSC SecurityConfig, snapshotRC io.ReadCloser) (func(), func() bool, error) {
	os.RemoveAll(dataDir)
	if snapshotRC != nil {
		err := restore(name, dataDir, privateAddress, peerSC, snapshotRC)
		if err != nil {
			return nil, notRunning, fmt.Errorf("failed to restore snapshot: %v", err)
		}
	}

	return startServer(
		"new",
		name,
		dataDir,
		dataQuota,
		publicAddress,
		privateAddress,
		clientSC,
		peerSC,
		map[string]string{name: privateAddress},
	)
}

func JoinCluster(name, dataDir string, dataQuota int64, publicAddress, privateAddress string, clientSC, peerSC SecurityConfig, memberID uint64, clientsAddresses []string, peersAddresses map[string]string) (func(), func() bool, error) {
	ss := func() (func(), func() bool, error) {
		return startServer(
			"existing",
			name,
			dataDir,
			dataQuota,
			publicAddress,
			privateAddress,
			clientSC,
			peerSC,
			peersAddresses,
		)
	}

	if memberID != 0 {
		if stop, isRunning, err := ss(); err == nil {
			return stop, isRunning, nil
		}
		log.Warn("failed to join as an existing member, resetting")
		if err := RemoveMember(clientsAddresses, clientSC, memberID); err != nil {
			return nil, notRunning, fmt.Errorf("failed to join cluster as an existing member, and failed to remove membership")
		}
	}
	os.RemoveAll(dataDir)

	unlock, err := addMember(clientsAddresses, clientSC, []string{peerURL(privateAddress, peerSC.AutoTLS || !peerSC.TLSInfo().Empty())})
	if err != nil {
		return nil, notRunning, fmt.Errorf("failed to add member to cluster: %v", err)
	}
	defer unlock()

	stop, isRunning, err := ss()
	if err != nil {
		return nil, notRunning, fmt.Errorf("failed to start server: %v", err)
	}
	return stop, isRunning, err
}

func startServer(state, name, dataDir string, dataQuota int64, publicAddress, privateAddress string, clientSC, peerSC SecurityConfig, peersAddresses map[string]string) (func(), func() bool, error) {
	cfg := embed.NewConfig()
	cfg.ClusterState = state
	cfg.Name = name
	cfg.Dir = dataDir
	cfg.PeerAutoTLS = peerSC.AutoTLS
	cfg.PeerTLSInfo = peerSC.TLSInfo()
	cfg.ClientAutoTLS = clientSC.AutoTLS
	cfg.ClientTLSInfo = clientSC.TLSInfo()
	cfg.InitialCluster = initialCluster(peersAddresses, peerSC.TLSEnabled())
	cfg.LPUrls, _ = types.NewURLs([]string{peerURL(privateAddress, peerSC.TLSEnabled())})
	cfg.APUrls, _ = types.NewURLs([]string{peerURL(privateAddress, peerSC.TLSEnabled())})
	cfg.LCUrls, _ = types.NewURLs([]string{clientURL(privateAddress, clientSC.TLSEnabled())})
	cfg.ACUrls, _ = types.NewURLs([]string{clientURL(publicAddress, clientSC.TLSEnabled())})
	cfg.ListenMetricsUrls = metricsURLs(privateAddress)
	cfg.Metrics = "extensive"
	cfg.QuotaBackendBytes = dataQuota

	tTimeOut := time.Now().Add(defaultStartTimeout)
	server, err := embed.StartEtcd(cfg)
	if err != nil {
		return nil, notRunning, err
	}

	running := true
	stopServerCh := make(chan struct{})

	isRunning := func() bool {
		return running
	}
	stopServer := func() {
		close(stopServerCh)
		server.Server.Stop()
		server.Close()
		running = false
	}

	// Wait until the server announces its ready, or until timeout is exceeded.
	select {
	case <-server.Server.ReadyNotify():
		break
	case <-time.After(time.Until(tTimeOut)):
		stopServer()
		return nil, notRunning, fmt.Errorf("server took too long to start")
	}

	// Wait until the member is healthy, or until timeout is exceeded.
	healthyNotify := make(chan struct{})
	defer close(healthyNotify)

	clientTC, err := clientSC.ClientConfig()
	if err != nil {
		return nil, notRunning, fmt.Errorf("failed to read client transport security: %s", err)
	}

	go func() {
		healthySince := time.Time{}
		for {
			if time.Now().After(tTimeOut) {
				break
			}
			if isMemberHealthy(clientURL(privateAddress, clientSC.TLSEnabled()), clientTC) {
				if healthySince.IsZero() {
					healthySince = time.Now()
				}
			} else {
				healthySince = time.Time{}
			}
			if !healthySince.IsZero() && time.Since(healthySince) >= defaultStartHealthyThreshold {
				healthyNotify <- struct{}{}
				break
			}
			time.Sleep(1 * time.Second)
		}
	}()

	select {
	case <-healthyNotify:
		break
	case <-time.After(time.Until(tTimeOut)):
		stopServer()
		return nil, notRunning, fmt.Errorf("member never became healthy")
	}

	// Start go-routine to re-start the etcd server immediately if it crashed.
	go func() {
		for {
			select {
			case err := <-server.Err():
				log.WithError(err).Warnf("etcd server crashed")
				running = false
			case <-stopServerCh:
				return
			}
		}
	}()

	return stopServer, isRunning, nil
}

func addMember(clientsAddresses []string, sc SecurityConfig, pURLs []string) (func(), error) {
	tc, err := sc.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read client transport security: %s", err)
	}

	c, err := newClient(clientsURLs(clientsAddresses, sc.TLSEnabled()), tc)
	if err != nil {
		return nil, err
	}

	unlock, err := lock(c, "add_member", defaultRequestTimeout)
	if err != nil {
		return nil, fmt.Errorf("unable to acquire lock to join Cluster: %v", err)
		c.Close()
	}
	unlockClose := func() {
		unlock()
		c.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	_, err = c.MemberAdd(ctx, pURLs)
	if err != nil {
		unlockClose()
		return nil, err
	}

	return unlockClose, nil
}

func RemoveMember(clientsAddresses []string, sc SecurityConfig, memberID uint64) error {
	tc, err := sc.ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to read client transport security: %s", err)
	}

	c, err := newClient(clientsURLs(clientsAddresses, sc.TLSEnabled()), tc)
	if err != nil {
		return err
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	_, err = c.MemberRemove(ctx, memberID)
	if err != nil && err != rpctypes.ErrMemberNotFound {
		return err
	}
	return nil
}

func lock(c *etcdcl.Client, name string, maxWait time.Duration) (func(), error) {
	session, err := concurrency.NewSession(c)
	if err != nil {
		return nil, err
	}
	mutex := concurrency.NewMutex(session, name)

	ctx, cancel := context.WithTimeout(context.Background(), maxWait)
	defer cancel()

	err = mutex.Lock(ctx)
	if err != nil {
		session.Close()
		return nil, err
	}

	unlock := func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
		mutex.Unlock(ctx)
		cancel()
		session.Close()
	}
	return unlock, nil
}

func notRunning() bool {
	return false
}
