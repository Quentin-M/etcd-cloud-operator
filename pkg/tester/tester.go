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
	"time"

	"go.uber.org/zap"

	"github.com/quentin-m/etcd-cloud-operator/pkg/etcd"
)

// Config is the global configuration for an instance of ECO.
type Config struct {
	Cluster Cluster `yaml:"cluster"`
}

type Cluster struct {
	Address string `yaml:"address"`
	Size    int    `yaml:"size"`

	TLS etcd.SecurityConfig `yaml:"tls"`
}

func Run(cfg Config) {
	// Start monitoring.
	go promRun()

	// Create the etcd cluster client.
	c, err := etcd.NewClient(
		[]string{cfg.Cluster.Address},
		cfg.Cluster.TLS,
		false,
	)
	if err != nil {
		zap.S().With(zap.Error(err)).Fatal("failed to create etcd cluster client")
	}
	defer c.Close()

	// Run all tests.
	for _, testCase := range testCases {
		runTest(testCase, c, cfg, &dataMarker{client: c})
	}

	// Compact and defrag the cluster up to this point.
	if err := c.Cleanup(); err != nil {
		zap.S().With(zap.Error(err)).Fatal("failed to compact/defrag the cluster")
	}
}

func runTest(testCase testCase, client *etcd.Client, cfg Config, dm *dataMarker) {
	zap.S().Infof("starting test %q", testCase.name)

	promSetRunningTest(testCase.name)
	defer promSetRunningTest("")
	defer promSetInjectedFailure("")

	// Compact and defrag the cluster up to this point, and set a data marker.
	if err := client.Cleanup(); err != nil {
		zap.S().With(zap.Error(err)).Warn("failed to compact/defrag the cluster, next test might be affected")
	}

	if err := dm.setMarker(); err != nil {
		zap.S().With(zap.Error(err)).Warn("failed to put data marker, skipping test")
		return
	}

	// Verify that the cluster is healthy in the first place, and get the current leader.
	leaderID, err := isHealthy(client, cfg.Cluster.Size)
	if err != nil {
		zap.S().With(zap.Error(err)).Fatal("cluster is unhealthy")
	}

	// Start stressing the cluster.
	stresser := newStresser()
	stresser.Start(client.Client)

	time.Sleep(15 * time.Second)

	// Inject the failure.
	promSetInjectedFailure(testCase.name)

	if err := testCase.fi(failureInjectorConfig{client: client, testerConfig: cfg, leaderID: leaderID}); err != nil {
		zap.S().With(zap.Error(err)).Warn("failed to inject failure, skipping test")
		return
	}

	time.Sleep(time.Second)
	promSetInjectedFailure("")

	// Wait for cluster to be healthy for at least 60 seconds (+ the time it takes to check for healthiness).
	for i := 0; i < 4; i++ {
		time.Sleep(15 * time.Second)
		if _, err := isHealthy(client, cfg.Cluster.Size); err != nil {
			zap.S().With(zap.Error(err)).Warn("cluster is unhealthy")
			i = -1
		}
	}

	// Stop stressing the cluster.
	stresser.Stop()

	// Verify cluster's consistency and the presence of the data marker.
	if err := client.IsConsistent(); err != nil {
		zap.S().With(zap.Error(err)).Fatal("cluster is un-consistent, exiting")
	}
	if err := dm.verify(testCase.isLossy); err != nil {
		zap.S().Error(err)
	}
}

type dataMarker struct {
	client *etcd.Client
	time   time.Time
}

func (dm *dataMarker) setMarker() error {
	dm.time = time.Now()
	data, _ := dm.time.MarshalText()

	_, err := dm.client.Put(context.Background(), "/marker", string(data))
	return err
}

func (dm *dataMarker) verify(isLossy bool) error {
	var t time.Time

	resp, err := dm.client.Get(context.Background(), "/marker")
	if err != nil || len(resp.Kvs) == 0 {
		return errors.New("failed to verify data marker: it has disappeared - cluster might have reset")
	}
	if err := t.UnmarshalText(resp.Kvs[0].Value); err != nil {
		return errors.New("failed to verify data marker: unexpected read value")
	}

	if !t.Equal(dm.time) {
		zap.S().Infof("data marker was reset the value it held %v ago", time.Since(t))

		if !isLossy {
			return errors.New("failed to verify data marker: unexpected value - cluster might have been restored to an old snapshot unexpectedly")
		}
	}
	return nil
}
