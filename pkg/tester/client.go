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

package tester

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"

	"github.com/quentin-m/etcd-cloud-operator/pkg/etcd"
)

func isHealthy(c *etcd.Client, clusterSize int) (uint64, error) {
	if !c.IsHealthy(5, 5*time.Second) {
		return 0, fmt.Errorf("cluster doesn't have quorum")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var healthyMembers, unhealthyMembers []string
	var leader uint64
	var leaderChanged bool

	c.ForEachMember(func(c *etcd.Client, m *etcdserverpb.Member) error {
		cURL := etcd.URL2Address(m.PeerURLs[0])

		resp, err := c.Status(ctx, etcd.ClientURL(cURL, c.SC.TLSEnabled()))
		if err != nil {
			unhealthyMembers = append(unhealthyMembers, cURL)
			return nil
		}
		healthyMembers = append(healthyMembers, cURL)

		if leader == 0 {
			leader = resp.Leader
		} else if resp.Leader != leader {
			leaderChanged = true
		}

		return nil
	})

	if leaderChanged {
		return leader, fmt.Errorf("leader has changed during member listing, or all members do not agree on one")
	}
	if len(unhealthyMembers) > 0 {
		return leader, fmt.Errorf("cluster has unhealthy members: %v", unhealthyMembers)
	}
	if len(healthyMembers) != clusterSize {
		return leader, fmt.Errorf("cluster has %d healthy members, expected %d", len(healthyMembers), clusterSize)
	}

	// Verify no alarms are firing.
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	alarms, err := c.AlarmList(ctx)
	if err != nil {
		return leader, fmt.Errorf("failed to retrieve alarms: %s", err)
	}
	if len(alarms.Alarms) > 0 {
		return leader, errors.New("cluster has alarms firing")
	}

	return leader, nil
}

func execRemote(address string, cmd string) error {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return err
	}

	connection, err := ssh.Dial("tcp", address+":22", &ssh.ClientConfig{
		User: "core",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return err
	}

	session, err := connection.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	err = session.RequestPty("xterm", 80, 40, ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	})
	if err != nil {
		return err
	}

	return session.Run(cmd)
}
