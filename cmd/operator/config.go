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

package main

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/quentin-m/etcd-cloud-operator/pkg/etcd"
	"github.com/quentin-m/etcd-cloud-operator/pkg/operator"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/snapshot"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// config represents a YAML configuration file that namespaces all ECO
// configuration under the top-level "eco" key.
type config struct {
	ECO operator.Config `yaml:"eco"`
}

// defaultConfig is a configuration that can be used as a fallback value.
func defaultConfig() config {
	return config{
		ECO: operator.Config{
			CheckInterval: 30 * time.Second,
			AutoDisasterRecovery: false,
			UnseenInstanceTTL:    60 * time.Second,
			Etcd: etcd.EtcdConfiguration{
				DataDir: "/var/lib/etcd",
				PeerTransportSecurity: etcd.SecurityConfig{
					AutoTLS: true,
				},
			},
			Snapshot: snapshot.Config{
				Interval: 30 * time.Minute,
				TTL:      24 * time.Hour,
			},
		},
	}
}

// loadConfig is a shortcut to open a file, read it, and generate a
// config.
//
// It supports relative and absolute paths. Given "", it returns defaultConfig.
func loadConfig(path string) (config, error) {
	config := defaultConfig()
	if path == "" {
		return config, nil
	}

	f, err := os.Open(os.ExpandEnv(path))
	if err != nil {
		return config, err
	}
	defer f.Close()

	d, err := ioutil.ReadAll(f)
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(d, &config)
	if err != nil {
		return config, err
	}

	log.Infof("loaded configuration file %v", path)
	return config, err
}
