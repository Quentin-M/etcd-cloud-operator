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
	"fmt"
	"net/url"

	etcdcl "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	etcdpb "github.com/coreos/etcd/etcdserver/etcdserverpb"
	log "github.com/sirupsen/logrus"
)

type Member struct {
	ID          uint64
	PeerAddress string
	Healthy     bool
}

func ClusterStatus(addresses []string, sc SecurityConfig) (map[string]Member, bool, error) {
	tc, err := sc.ClientConfig()
	if err != nil {
		return nil, false, fmt.Errorf("failed to read client transport security: %s", err)
	}

	c, err := newClient(clientsURLs(addresses, sc.TLSEnabled()), tc)
	if err != nil {
		return nil, false, err
	}
	defer c.Close()

	members, err := listMembers(c)
	if err != nil {
		return nil, false, err
	}
	
	membersM := make(map[string]Member, len(members))
	healthyC := 0
	for _, member := range members {
		if member.Name == "" || len(member.PeerURLs) == 0 {
			continue
		}
		peerURL, _ := url.Parse(member.PeerURLs[0])
		
		membersM[member.Name] = Member{
			member.ID,
			peerURL.Hostname(),
			isMemberHealthy(clientURL(peerURL.Hostname(), sc.TLSEnabled()), tc),
		}
		if membersM[member.Name].Healthy {
			healthyC++
		}
	}

	return membersM, healthyC >= len(membersM)/2+1, nil
}

func isMemberHealthy(cURL string, tc *tls.Config) bool {
	c, err := newClient([]string{cURL}, tc)
	if err != nil {
		log.Debugf("failed to create client for health check: %v", err)
		return false
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	_, err = c.Get(ctx, "health")
	return err == nil || err == rpctypes.ErrPermissionDenied
}

func listMembers(c *etcdcl.Client) ([]*etcdpb.Member, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	resp, err := c.MemberList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %v", err)
	}
	return resp.Members, nil
}
