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
	"time"

	"github.com/quentin-m/etcd-cloud-operator/pkg/etcd"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/asg"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/snapshot"
	log "github.com/sirupsen/logrus"
)

func initProviders(cfg Config) (asg.Provider, snapshot.Provider) {
	if cfg.ASG.Provider == "" {
		log.Fatal("no auto-scaling group provider configuration given")
	}
	asgProvider, ok := asg.AsMap()[cfg.ASG.Provider]
	if !ok {
		log.Fatalf("unknown auto-scaling group provider %q, available providers: %v", cfg.ASG.Provider, asg.AsList())
	}
	if err := asgProvider.Configure(cfg.ASG); err != nil {
		log.WithError(err).Fatal("failed to configure auto-scaling group provider")
	}

	if cfg.Snapshot.Provider == "" {
		return asgProvider, nil
	}
	snapshotProvider, ok := snapshot.AsMap()[cfg.Snapshot.Provider]
	if !ok {
		log.Fatalf("unknown snapshot provider %q, available providers: %v", cfg.Snapshot.Provider, snapshot.AsList())
	}
	if err := snapshotProvider.Configure(cfg.Snapshot); err != nil {
		log.WithError(err).Fatal("failed to configure snapshot provider")
	}

	return asgProvider, snapshotProvider
}

func initFuncs(snapshotProvider snapshot.Provider, cfg Config) (func(), func() bool, func([]asg.Instance, map[string]etcd.Member), func(asg.Instance, bool)) {
	return func() {}, func() bool { return false }, newLastSeenCleaner(cfg), newSnapshotter(snapshotProvider, cfg)
}

// newLastSeenCleaner returns a function that removes unhealthy members
// associated to unseen instances.
func newLastSeenCleaner(cfg Config) func([]asg.Instance, map[string]etcd.Member) {
	memberInstanceLastSeen := make(map[string]time.Time)

	return func(instances []asg.Instance, members map[string]etcd.Member) {
		for _, instance := range instances {
			memberInstanceLastSeen[instance.Name()] = time.Now()
		}
		for memberName, member := range members {
			if !memberInstanceLastSeen[memberName].IsZero() && time.Since(memberInstanceLastSeen[memberName]) > cfg.UnseenInstanceTTL {
				log.Infof("removing unhealthy member %q associated to instance unseen for %v", memberName, cfg.UnseenInstanceTTL)
				etcd.RemoveMember(instancesAddresses(instances), cfg.Etcd.ClientTransportSecurity, member.ID)
			}
		}
	}
}

// newSnapshotter returns a function that can be used to take snapshots, and
// start a periodic purge
func newSnapshotter(provider snapshot.Provider, cfg Config) func(asg.Instance, bool) {
	if provider == nil || cfg.Snapshot.Interval == 0 {
		log.Warn("snapshots are disabled")
		return func(instance asg.Instance, forceNow bool) {}
	}
	lastSnapshotTime := time.Now()

	doSnapshot := func(instance asg.Instance, forceNow bool) {
		if time.Since(lastSnapshotTime) < cfg.Snapshot.Interval && !forceNow {
			return
		}
		go provider.Purge(cfg.Snapshot.TTL)

		minRev, err := provider.LatestRev()
		if err != nil {
			if err != snapshot.ErrNoSnapshot {
				log.WithError(err).Warn("failed to find latest snapshot revision, continuing anyways")
			}
			minRev = 0
		}
		
		r, close, rev, err := etcd.Snapshot(instance.Address(), cfg.Etcd.ClientTransportSecurity, minRev)
		if err == etcd.ErrMemberRevisionTooOld {
			log.Debugf("skipping snapshot: current revision %016x and latest snapshot %016x", minRev, rev)
			lastSnapshotTime = time.Now()
			return
		}
		if err != nil {
			log.WithError(err).Error("failed to initiate snapshot")
			return
		}
		defer close()

		n, err := provider.Save(r, instance.Name(), rev)
		if err != nil {
			log.WithError(err).Error("failed to save snapshot")
			return
		}

		log.Infof("snapshot %q saved successfully (%.3f MB)", snapshot.Name(rev, instance.Name()), toMB(n))
		lastSnapshotTime = time.Now()
	}

	return doSnapshot
}
