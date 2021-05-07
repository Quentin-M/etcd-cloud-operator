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

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/quentin-m/etcd-cloud-operator/pkg/tester"
)

// config represents a YAML configuration file that namespaces all ECO tester configuration under the
// top-level "eco-tester" key.
type config struct {
	ECOTester tester.Config `yaml:"eco-tester"`
}

// loadConfig is a shortcut to open a file, read it, and generate a
// config.
//
// It supports relative and absolute paths. Given "", it returns defaultConfig.
func loadConfig(path string) (config, error) {
	config := config{}

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

	zap.S().Infof("loaded configuration file %v", path)
	return config, err
}
