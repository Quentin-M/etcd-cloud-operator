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

package docker

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"

	"github.com/quentin-m/etcd-cloud-operator/pkg/providers"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/asg"
)

func init() {
	asg.Register("docker", &docker{})
}

type docker struct {
	config config
}

type config struct {
	Size       int    `yaml:"size"`
	NameFilter string `yaml:"name-filter"`
}

func (d *docker) Configure(providerConfig asg.Config) error {
	d.config = config{Size: 3, NameFilter: "eco-"}
	if err := providers.ParseParams(providerConfig.Params, &d.config); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}
	return nil
}

func (d *docker) AutoScalingGroupStatus() (instances []asg.Instance, self asg.Instance, size int, err error) {
	instancesStr := make([]string, 0, d.config.Size)
	hostname, _ := os.Hostname()

	// List all containers names, which match the filter.
	containerNames, err := containerList(d.config.NameFilter)
	if err != nil {
		return nil, nil, 0, err
	}

	for _, name := range containerNames {
		container, err := containerInspect(name)
		if err != nil {
			return nil, nil, 0, err
		}
		if strings.Contains(container.id, hostname) {
			self = container
		}
		instances = append(instances, container)
		instancesStr = append(instancesStr, container.address)
	}
	size = d.config.Size

	zap.S().Debugf("Discovered %d / %d replicas: %s", len(instances), d.config.Size, strings.Join(instancesStr, ", "))
	return
}
