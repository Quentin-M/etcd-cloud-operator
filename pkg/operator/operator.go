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
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/quentin-m/etcd-cloud-operator/pkg/etcd"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/asg"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/snapshot"
)

const (
	loopInterval = 15 * time.Second

	webServerPort = 2378
)

type Operator struct {
	server *etcd.Server

	// New()
	cfg              Config
	asgProvider      asg.Provider
	snapshotProvider snapshot.Provider

	httpClient *http.Client

	shutdownChan chan os.Signal
	shutdown     bool
	ticker       *time.Ticker

	// evaluate()
	etcdHealthy bool
	etcdRunning bool

	etcdClient   *etcd.Client
	etcdSnapshot *snapshot.Metadata

	state  string
	states map[string]int

	isSeeder    bool
	clusterSize int
}

// Config is the global configuration for an instance of ECO.
type Config struct {
	CheckInterval        time.Duration `yaml:"check-interval"`
	UnhealthyMemberTTL 	 time.Duration `yaml:"unhealthy-member-ttl"`

	Etcd     etcd.EtcdConfiguration `yaml:"etcd"`
	ASG      asg.Config             `yaml:"asg"`
	Snapshot snapshot.Config        `yaml:"snapshot"`
}

func New(cfg Config) *Operator {
	// Initialize providers.
	asgProvider, snapshotProvider := initProviders(cfg)
	if snapshotProvider == nil || cfg.Snapshot.Interval == 0 {
		log.Fatal("snapshots must be enabled for disaster recovery")
	}

	// Setup signal handler.
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGTERM)

	return &Operator{
		cfg:              cfg,
		asgProvider:      asgProvider,
		snapshotProvider: snapshotProvider,
		httpClient:       &http.Client{Timeout: isHealthyTimeout},
		state:            "UNKNOWN",
		ticker:           time.NewTicker(cfg.CheckInterval),
		shutdownChan:     shutdownChan,
	}
}

func (s *Operator) Run() {
	go s.webserver()

	for {
		if err := s.evaluate(); err != nil {
			log.WithError(err).Warn("could not evaluate cluster state")
			s.wait()
			continue
		}
		if err := s.execute(); err != nil {
			log.WithError(err).Warn("could not execute action")
		}
		s.wait()
	}
}

func (s *Operator) evaluate() error {
	// Fetch the auto-scaling group state.
	asgInstances, asgSelf, asgSize, err := s.asgProvider.AutoScalingGroupStatus()
	if err != nil {
		return fmt.Errorf("failed to sync auto-scaling group: %v", err)
	}

	// Create the etcd cluster client.
	client, err := etcd.NewClient(instancesAddresses(asgInstances), s.cfg.Etcd.ClientTransportSecurity, true)
	if err != nil {
		log.WithError(err).Warn("failed to create etcd cluster client")
	}

	// Output.
	if s.server == nil {
		s.server = etcd.NewServer(serverConfig(s.cfg, asgSelf, s.snapshotProvider))
	}

	s.etcdRunning = s.server.IsRunning()
	s.etcdHealthy, s.isSeeder, s.states = fetchStatuses(s.httpClient, client, asgInstances, asgSelf)
	s.clusterSize = asgSize

	s.etcdClient = client
	return nil
}

func (s *Operator) execute() error {
	defer func() {
		if s.etcdClient != nil {
			s.etcdClient.Close()
		}
	}()

	switch {
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	case s.shutdown:
		log.Info("STATUS: Received SIGTERM -> Snapshot + Stop")
		s.state = "PENDING"

		s.server.Stop(s.etcdHealthy, true)
		os.Exit(0)
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	case s.etcdHealthy && !s.etcdRunning:
		log.Info("STATUS: Healthy + Not running -> Join")
		s.state = "PENDING"

		if err := s.server.Join(s.etcdClient); err != nil {
			log.WithError(err).Error("failed to join the cluster")
		}
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	case s.etcdHealthy && s.etcdRunning:
		log.Info("STATUS: Healthy + Running -> Standby")
		s.state = "OK"
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	case !s.etcdHealthy && s.etcdRunning && s.states["OK"] >= s.clusterSize/2+1:
		log.Info("STATUS: Unhealthy + Running -> Pending confirmation from other ECO instances")
		s.state = "PENDING"
	case !s.etcdHealthy && s.etcdRunning && s.states["OK"] < s.clusterSize/2+1:
		log.Info("STATUS: Unhealthy + Running + No quorum -> Snapshot + Stop")
		s.state = "PENDING"

		s.server.Stop(false, true)
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	case !s.etcdHealthy && !s.etcdRunning && (s.states["START"] != s.clusterSize || !s.isSeeder):
		if s.state != "START" {
			var err error
			if s.etcdSnapshot, err = s.server.SnapshotInfo(); err != nil && err != snapshot.ErrNoSnapshot {
				return err
			}
		}
		log.Info("STATUS: Unhealthy + Not running -> Ready to start + Pending all ready / seeder")
		s.state = "START"
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	case !s.etcdHealthy && !s.etcdRunning && s.states["START"] == s.clusterSize && s.isSeeder:
		log.Info("STATUS: Unhealthy + Not running + All ready + Seeder status -> Seeding cluster")
		s.state = "START"

		if err := s.server.Seed(s.etcdSnapshot); err != nil {
			log.WithError(err).Error("failed to seed the cluster")
		}
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		s.state = "UNKNOWN"
		return errors.New("no adequate action found")
		////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	}

	if s.state == "OK" && s.isSeeder && s.cfg.Etcd.InitACL != nil {
		if err := s.reconcileInitACLConfig(s.cfg.Etcd.InitACL); err != nil {
			log.WithError(err).Error("failed to reconcile initial ACL config")
			return err
		}
	}

	return nil
}

func (s *Operator) webserver() {
	http.HandleFunc("/status", func(w http.ResponseWriter, _ *http.Request) {
		st := status{State: s.state}
		if s.etcdSnapshot != nil {
			st.Revision = s.etcdSnapshot.Revision
		}
		b, err := json.Marshal(&st)
		if err != nil {
			log.WithError(err).Warn("failed to marshal status")
			return
		}
		if _, err := w.Write(b); err != nil {
			log.WithError(err).Warn("failed to write status")
		}
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", webServerPort), nil))
}

func (s *Operator) wait() {
	if s.etcdClient != nil {
		s.etcdClient.Close()
	}

	select {
	case <-s.ticker.C:
	case <-s.shutdownChan:
		s.shutdown = true
	}
}
