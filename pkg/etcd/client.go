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
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	etcdcl "github.com/coreos/etcd/clientv3"
	concurrency "github.com/coreos/etcd/clientv3/concurrency"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/coreos/etcd/mvcc"
)

type Client struct {
	*clientv3.Client

	clientsAddresses []string
	autoSync         bool

	SC SecurityConfig
	TC *tls.Config
}

// NewClient creates a new etcd client wrapper.
//
// The client must be closed.
func NewClient(clientsAddresses []string, sc SecurityConfig, autoSync bool) (*Client, error) {
	tc, err := sc.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read client transport security: %s", err)
	}

	var autoSyncInterval time.Duration
	if autoSync {
		autoSyncInterval = defaultAutoSync
	}

	client, err := etcdcl.New(etcdcl.Config{
		Endpoints:        ClientsURLs(clientsAddresses, sc.TLSEnabled()),
		DialTimeout:      defaultDialTimeout,
		TLS:              tc,
		AutoSyncInterval: autoSyncInterval,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		Client: client,

		autoSync:         autoSync,
		clientsAddresses: clientsAddresses,

		SC: sc,
		TC: tc,
	}, nil
}

func (c *Client) Members() ([]*etcdserverpb.Member, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	memberList, err := c.MemberList(ctx)
	if err != nil {
		return []*etcdserverpb.Member{}, err
	}
	return memberList.Members, nil
}

func (c *Client) ForEachMember(f func(*Client, *etcdserverpb.Member) error) error {
	members, err := c.Members()
	if err != nil {
		return err
	}

	// For each member, create a client for them and execute the given command in parallel.
	errChan := make(chan string, len(members))
	for _, member := range members {
		go func(member *etcdserverpb.Member) {
			client, err := NewClient([]string{URL2Address(member.PeerURLs[0])}, c.SC, false)
			if err != nil {
				errChan <- fmt.Sprintf("[%s]: %s", member.Name, err)
				return
			}
			defer client.Close()

			if err := f(client, member); err != nil {
				errChan <- fmt.Sprintf("[%s]: %s", member.Name, err)
			}
			errChan <- ""
		}(member)
	}

	// Synchronize, and aggregate the errors.
	var errAgg []string
	for i := 0; i < len(members); i++ {
		if mErr := <-errChan; len(mErr) > 0 {
			errAgg = append(errAgg, mErr)
		}
	}
	if len(errAgg) > 0 {
		return errors.New(strings.Join(errAgg, ","))
	}
	return nil
}

func (c *Client) AddMember(name string, pURLs []string) (uint64, func(), error) {
	unlock, err := c.Lock("/eco/"+name+"/join", defaultRequestTimeout)
	if err != nil {
		return 0, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	resp, err := c.MemberAdd(ctx, pURLs)
	if err != nil {
		unlock()
		return 0, nil, err
	}

	return resp.Member.ID, unlock, nil
}

func (c *Client) RemoveMember(name string, id uint64) error {
	unlock, err := c.Lock("/eco/"+name+"/join", defaultRequestTimeout)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %v", err)
	}
	defer unlock()

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	_, err = c.MemberRemove(ctx, id)
	if err != nil && err != rpctypes.ErrMemberNotFound {
		return err
	}
	return nil
}

func (c *Client) Lock(name string, maxWait time.Duration) (func(), error) {
	session, err := concurrency.NewSession(c.Client)
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

// IsHealthy verifies if the cluster is healthy / has quorum, or when a single address is given and autoSync is off,
// if the member behind that address is responsive (and not that the overall cluster is operating).
//
// For checking the health of the entire cluster, IsHealthy simply leverages linearizable reads, which are reads
// that must go through raft. Even if a few members are unreachable or isolated, IsHealthy is able to respond
// correctly thanks to the smart underlying balancer in the client.
func (c *Client) IsHealthy(retries int, timeout time.Duration) bool {
	opOptions := []clientv3.OpOption{}
	if len(c.clientsAddresses) == 0 && !c.autoSync {
		opOptions = append(opOptions, clientv3.WithSerializable())
	}

	for i := 0; i < retries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		_, err := c.Get(ctx, "health")
		cancel()

		if err == nil || err == rpctypes.ErrPermissionDenied || err == rpctypes.ErrGRPCCompacted {
			return true
		}
	}
	return false
}

func (c *Client) GetHighestRevision() (int64, error) {
	revs, _, err := c.GetRevisionsHashes()
	if err != nil {
		return -1, err
	}

	var maxRev int64
	for _, rev := range revs {
		if rev > maxRev {
			maxRev = rev
		}
	}

	return maxRev, nil
}

func (c *Client) IsConsistent() error {
	var (
		revs   map[string]int64
		hashes map[string]int64
		err    error
	)

	for i := 0; i < 15; i++ {
		revs, hashes, err = c.GetRevisionsHashes()
		if err != nil || !getSameValue(revs) || !getSameValue(hashes) {
			time.Sleep(time.Second)
			continue
		}
		return nil
	}

	return fmt.Errorf("cluster is unconsistent: [revisions: %v] and [hashes: %v]", revs, hashes)
}

func (c *Client) GetRevisionsHashes() (map[string]int64, map[string]int64, error) {
	var rhMutex sync.Mutex
	revs := make(map[string]int64)
	hashes := make(map[string]int64)

	ctx, cancel := context.WithTimeout(context.Background(), 2*defaultRequestTimeout)
	defer cancel()

	f := func(c *Client, m *etcdserverpb.Member) error {
		cURL := ClientURL(URL2Address(m.PeerURLs[0]), c.SC.TLSEnabled())

		s, err := c.Status(ctx, cURL)
		if err != nil {
			return fmt.Errorf("failed to get Status: %v", err)
		}

		h, err := c.Maintenance.HashKV(ctx, cURL, 0)
		if err != nil {
			return fmt.Errorf("failed to get HashKV: %v", err)
		}

		rhMutex.Lock()
		defer rhMutex.Unlock()

		revs[m.Name] = s.Header.Revision
		hashes[m.Name] = int64(h.Hash)
		return nil
	}

	return revs, hashes, c.ForEachMember(f)
}

func (c *Client) Cleanup() error {
	// Get the current revision of the cluster.
	rev, err := c.GetHighestRevision()
	if err != nil {
		return err
	}

	// Compact.
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	if _, err := c.Client.Compact(ctx, rev); err != nil && !strings.Contains(err.Error(), mvcc.ErrCompacted.Error()) {
		return fmt.Errorf("failed to compact: %v", err)
	}

	// Defrag each member one after the other.
	var mu sync.Mutex
	f := func(c *Client, m *etcdserverpb.Member) error {
		mu.Lock()
		defer mu.Unlock()

		if _, err := c.Defragment(ctx, ClientURL(URL2Address(m.PeerURLs[0]), c.SC.TLSEnabled())); err != nil {
			return fmt.Errorf("failed to defragment: %v", err)
		}
		return nil
	}
	return c.ForEachMember(f)
}
