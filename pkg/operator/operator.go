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

package operator

import (
	"fmt"
	"io"
	"time"

	"github.com/quentin-m/etcd-cloud-operator/pkg/etcd"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/asg"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/snapshot"
	log "github.com/sirupsen/logrus"
)

// Config is the global configuration for an instance of ECO.
type Config struct {
	CheckInterval time.Duration `yaml:"check-interval"`
	UnseenInstanceTTL    time.Duration `yaml:"unseen-instance-ttl"`
	AutoDisasterRecovery bool          `yaml:"auto-disaster-recovery"`

	Etcd     etcd.EtcdConfiguration `yaml:"etcd"`
	ASG      asg.Config             `yaml:"asg"`
	Snapshot snapshot.Config        `yaml:"snapshot"`
}

func Run(cfg Config) {
	asgProvider, snapshotProvider := initProviders(cfg)
	etcdStop, etcdRunning, cleanLastSeen, doSnapshot := initFuncs(snapshotProvider, cfg)

	if (snapshotProvider == nil || cfg.Snapshot.Interval == 0) && cfg.AutoDisasterRecovery {
		log.Fatal("snapshots must be enabled for auto disaster recovery")
	}

	for {
		// Fetch cluster status.
		asgInstances, asgSelf, asgLeader, asgScaled, err := asgProvider.AutoScalingGroupStatus()
		if err != nil {
			log.WithError(err).Errorf("failed to sync auto-scaling group")
			time.Sleep(cfg.CheckInterval)
			continue
		}
		etcdMembers, etcdQuorum, err := etcd.ClusterStatus(instancesAddresses(asgInstances), cfg.Etcd.ClientTransportSecurity)
		if err != nil {
			log.WithError(err).Warnf("failed to sync etcd cluster")
		}
		printCluster(asgInstances, asgSelf, asgLeader, asgScaled, etcdMembers, etcdQuorum, etcdRunning())

		// Take action.
		err = nil
		switch {
		case etcdQuorum && etcdRunning():
			log.Info("STATE: Quorum + Running -> Standby")
			go doSnapshot(asgSelf, false)
		case etcdQuorum && !etcdRunning():
			log.Info("STATE: Quorum + !Running -> Join")
			etcdStop, etcdRunning, err = join(asgSelf, asgInstances, etcdMembers, cfg)
		case !etcdQuorum && etcdRunning() && cfg.AutoDisasterRecovery:
			log.Info("STATE: !Quorum + Running + Auto Disaster Recovery -> Snapshot + Leave")
			doSnapshot(asgSelf, true)
			etcdStop()
		case !etcdQuorum && etcdRunning():
			log.Info("STATE: !Quorum + Running + !Auto Disaster Recovery -> Standby")
		case !etcdQuorum && asgLeader.Name() == asgSelf.Name() && asgScaled:
			log.Info("STATE: !Quorum + !Running + ASG Scaled/Leader -> (Restore +) Seed")
			etcdStop, etcdRunning, err = seedOrRestore(asgSelf, snapshotProvider, cfg)
		case !etcdQuorum && (asgLeader.Name() != asgSelf.Name() || !asgScaled):
			log.Info("STATE: !Quorum + !Running + !ASG Scaled/Leader -> Standby")
		}
		if err != nil {
			log.Error(err)
		}

		// Maintain cluster in good state.
		cleanLastSeen(asgInstances, etcdMembers)

		time.Sleep(cfg.CheckInterval)
	}
}

func join(asgSelf asg.Instance, asgInstances []asg.Instance, etcdMembers map[string]etcd.Member, cfg Config) (func(), func() bool, error) {
	return etcd.JoinCluster(
		asgSelf.Name(),
		cfg.Etcd.DataDir,
		stringOverride(asgSelf.Address(), cfg.Etcd.AdvertiseAddress),
		asgSelf.Address(),
		cfg.Etcd.ClientTransportSecurity,
		cfg.Etcd.PeerTransportSecurity,
		etcdMembers[asgSelf.Name()].ID,
		instancesAddresses(asgInstances),
		peerAddresses(etcdMembers, asgSelf),
	)
}

func seedOrRestore(asgSelf asg.Instance, snapshotProvider snapshot.Provider, cfg Config) (func(), func() bool, error) {
	var snapshotRC io.ReadCloser
	if snapshotProvider != nil {
		var size, rev int64
		var err error

		snapshotRC, size, rev, err = snapshotProvider.Latest()
		if err != nil && err != snapshot.ErrNoSnapshot {
			return nil, func() bool { return false }, fmt.Errorf("failed to retrieve latest snapshot: %v", err)
		}

		log.Infof("restoring snapshot %q (%.3f MB)", snapshot.Name(rev, asgSelf.Name()), toMB(size))
	}

	return etcd.SeedCluster(
		asgSelf.Name(),
		cfg.Etcd.DataDir,
		stringOverride(asgSelf.Address(), cfg.Etcd.AdvertiseAddress),
		asgSelf.Address(),
		cfg.Etcd.ClientTransportSecurity,
		cfg.Etcd.PeerTransportSecurity,
		snapshotRC,
	)
}
