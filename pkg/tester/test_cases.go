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
	"errors"
	"sync"

	"go.etcd.io/etcd/etcdserver/etcdserverpb"
	"github.com/quentin-m/etcd-cloud-operator/pkg/etcd"
)

type failureInjectorConfig struct {
	client       *etcd.Client
	testerConfig Config
	leaderID     uint64
}

type failureInjector func(failureInjectorConfig) error

type testCase struct {
	name    string
	fi      failureInjector
	isLossy bool
}

var testCases = []testCase{
	{name: "killLeader", fi: killLeader},
	{name: "killOneSlave", fi: killOneSlave},
	{name: "killMajority", fi: killMajority},
	{name: "killAll", fi: killAll},
	{name: "killWipeOneSlave", fi: killWipeOneSlave},
	{name: "killWipeMajority", fi: killWipeMajority},
	{name: "killWipeAll", fi: killWipeAll, isLossy: true},
	{name: "stopWipeAll", fi: stopWipeAll},
}

func killLeader(cfg failureInjectorConfig) error {
	var killed bool
	f := func(client *etcd.Client, member *etcdserverpb.Member) error {
		if member.ID == cfg.leaderID {
			killed = true
			return execRemote(etcd.URL2Address(member.PeerURLs[0]), "sudo pkill -9 operator")
		}
		return nil
	}
	if err := cfg.client.ForEachMember(f); err != nil {
		return err
	}
	if !killed {
		return errors.New("could not find leader to kill")
	}
	return nil
}

func killOneSlave(cfg failureInjectorConfig) error {
	var once sync.Once
	f := func(client *etcd.Client, member *etcdserverpb.Member) error {
		if member.ID != cfg.leaderID {
			var err error
			once.Do(func() {
				err = execRemote(etcd.URL2Address(member.PeerURLs[0]), "sudo pkill -9 operator")
			})
			return err
		}
		return nil
	}
	if err := cfg.client.ForEachMember(f); err != nil {
		return err
	}
	return nil
}

func killMajority(cfg failureInjectorConfig) error {
	var i int
	var mi sync.Mutex

	f := func(client *etcd.Client, member *etcdserverpb.Member) error {
		mi.Lock()
		defer mi.Unlock()

		if i < cfg.testerConfig.Cluster.Size/2+1 {
			i++
			return execRemote(etcd.URL2Address(member.PeerURLs[0]), "sudo pkill -9 operator")
		}

		return nil
	}

	err := cfg.client.ForEachMember(f)
	return err
}

func killAll(cfg failureInjectorConfig) error {
	f := func(client *etcd.Client, member *etcdserverpb.Member) error {
		return execRemote(etcd.URL2Address(member.PeerURLs[0]), "sudo pkill -9 operator")
	}
	err := cfg.client.ForEachMember(f)
	return err
}

// killWipeOneSlave kills an ECO instance and wipes its data directory.
//
// The restarting ECO instance will see itself part of the cluster but won't be able to join until the rest of the
// cluster has excluded the unhealthy member. The impact should be limited by the load balancer health check however.
func killWipeOneSlave(cfg failureInjectorConfig) error {
	var once sync.Once
	var err error

	f := func(client *etcd.Client, member *etcdserverpb.Member) error {
		if member.ID != cfg.leaderID {
			once.Do(func() {
				err = execRemote(etcd.URL2Address(member.PeerURLs[0]), "sudo pkill -9 operator; sudo rm -rf /var/lib/eco/*")
			})
			return err
		}
		return nil
	}
	if err := cfg.client.ForEachMember(f); err != nil {
		return err
	}
	return nil
}

// killWipeMajority kills a majority of ECO instances and wipes their data directory.
//
// The remaining ECO instance will notice the cluster has lost quorum, perform a snapshot
// and leave. Once instances are back up, a cluster will be re-seeded from the snapshot.
func killWipeMajority(cfg failureInjectorConfig) error {
	var i int
	var mi sync.Mutex

	f := func(client *etcd.Client, member *etcdserverpb.Member) error {
		mi.Lock()
		defer mi.Unlock()

		if i < cfg.testerConfig.Cluster.Size/2+1 {
			i++
			return execRemote(etcd.URL2Address(member.PeerURLs[0]), "sudo pkill -9 operator; sudo rm -rf /var/lib/eco/*")
		}
		return nil
	}

	err := cfg.client.ForEachMember(f)
	return err
}

func killWipeAll(cfg failureInjectorConfig) error {
	f := func(client *etcd.Client, member *etcdserverpb.Member) error {
		return execRemote(etcd.URL2Address(member.PeerURLs[0]), "sudo pkill -9 operator; sudo rm -rf /var/lib/eco/*")
	}
	if err := cfg.client.ForEachMember(f); err != nil {
		return err
	}
	return nil
}

func stopWipeAll(cfg failureInjectorConfig) error {
	f := func(client *etcd.Client, member *etcdserverpb.Member) error {
		return execRemote(etcd.URL2Address(member.PeerURLs[0]), "sudo pkill -15 operator; tail --pid=$(pgrep operator) -f /dev/null; sudo rm -rf /var/lib/eco/*")
	}
	if err := cfg.client.ForEachMember(f); err != nil {
		return err
	}
	return nil
}

// TODO:
// - Network partition (leader, member)
// - Slow network (leader, member, all)
// - Corrupted network (leader, member)
