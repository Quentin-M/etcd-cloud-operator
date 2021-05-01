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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/quentin-m/etcd-cloud-operator/pkg/etcd"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/asg"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/snapshot"
)

const (
	isHealthRetries  = 3
	isHealthyTimeout = 10 * time.Second
)

type status struct {
	instance asg.Instance

	State    string `json:"state"`
	Revision int64  `json:"revision"`
}

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

func fetchStatuses(httpClient *http.Client, etcdClient *etcd.Client, asgInstances []asg.Instance, asgSelf asg.Instance) (bool, bool, map[string]int) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	wg.Add(1 + len(asgInstances))

	// Fetch etcd's healthiness.
	var etcdHealthy bool
	go func() {
		defer wg.Done()
		etcdHealthy = etcdClient != nil && etcdClient.IsHealthy(isHealthRetries, isHealthyTimeout)
	}()

	// Fetch ECO statuses.
	var ecoStatuses []*status
	for _, asgInstance := range asgInstances {
		go func(asgInstance asg.Instance) {
			defer wg.Done()

			st, err := fetchStatus(httpClient, asgInstance)
			if err != nil {
				log.WithError(err).Warnf("failed to query %s's ECO instance", asgInstance.Name())
				return
			}

			mu.Lock()
			defer mu.Unlock()

			ecoStatuses = append(ecoStatuses, st)
		}(asgInstance)
	}
	wg.Wait()

	// Sort the ECO statuses so we can systematically find the identity of the seeder.
	sort.Slice(ecoStatuses, func(i, j int) bool {
		if ecoStatuses[i].Revision == ecoStatuses[j].Revision {
			return ecoStatuses[i].instance.Name() < ecoStatuses[j].instance.Name()
		}
		return ecoStatuses[i].Revision < ecoStatuses[j].Revision
	})

	// Count ECO statuses and determine if we are the seeder.
	ecoStates := make(map[string]int)
	for _, ecoStatus := range ecoStatuses {
		if _, ok := ecoStates[ecoStatus.State]; !ok {
			ecoStates[ecoStatus.State] = 0
		}
		ecoStates[ecoStatus.State]++
	}

	return etcdHealthy, ecoStatuses[len(ecoStatuses)-1].instance.Name() == asgSelf.Name(), ecoStates
}

func fetchStatus(httpClient *http.Client, instance asg.Instance) (*status, error) {
	resp, err := httpClient.Get(fmt.Sprintf("http://%s:%d/status", instance.Address(), webServerPort))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var st status
	err = json.Unmarshal(b, &st)
	st.instance = instance
	return &st, err
}

func serverConfig(cfg Config, asgSelf asg.Instance, snapshotProvider snapshot.Provider) etcd.ServerConfig {
	return etcd.ServerConfig{
		Name:                    asgSelf.Name(),
		DataDir:                 cfg.Etcd.DataDir,
		DataQuota:               cfg.Etcd.BackendQuota,
		AutoCompactionMode:      cfg.Etcd.AutoCompactionMode,
		AutoCompactionRetention: cfg.Etcd.AutoCompactionRetention,
		PublicAddress:           stringOverride(asgSelf.Address(), cfg.Etcd.AdvertiseAddress),
		PrivateAddress:          asgSelf.Address(),
		ClientSC:                cfg.Etcd.ClientTransportSecurity,
		PeerSC:                  cfg.Etcd.PeerTransportSecurity,
		UnhealthyMemberTTL:      cfg.UnhealthyMemberTTL,
		SnapshotProvider:        snapshotProvider,
		SnapshotInterval:        cfg.Snapshot.Interval,
		SnapshotTTL:             cfg.Snapshot.TTL,
		JWTAuthTokenConfig:      cfg.Etcd.JWTAuthTokenConfig,
	}
}

func instancesAddresses(instances []asg.Instance) (addresses []string) {
	for _, instance := range instances {
		addresses = append(addresses, instance.Address())
	}
	return
}

func stringOverride(s, override string) string {
	if override != "" {
		return override
	}
	return s
}
