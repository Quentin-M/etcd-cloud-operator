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

package sts

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/asg"
)

func init() {
	asg.Register("sts", &sts{})
}

type sts struct {
	namespace, name, serviceName, dnsClusterSuffix string
	replicas int

	self instance
}

type instance struct {
	id, name, address, bindAddress string
}

func (i *instance) Name() string {
	return i.name
}

func (i *instance) Address() string {
	return i.address
}

func (i *instance) BindAddress() string {
	return "0.0.0.0"
}

func (a *sts) Configure(providerConfig asg.Config) (err error) {
	a.namespace, err = envOrErr("STATEFULSET_NAMESPACE")
	if err != nil {
		return
	}

	a.name, err = envOrErr("STATEFULSET_NAME")
	if err != nil {
		return
	}

	a.serviceName, err = envOrErr("STATEFULSET_SERVICE_NAME")
	if err != nil {
		return
	}

	a.dnsClusterSuffix, err = envOrErr("STATEFULSET_DNS_CLUSTER_SUFFIX")
	if err != nil {
		return
	}

	replicas, err := envOrErr("STATEFULSET_REPLICAS")
	if err != nil {
		return
	}
	a.replicas, err = strconv.Atoi(replicas)
	if err != nil {
		return errors.New("STATEFULSET_REPLICAS should be an integer")
	}

	a.self.name, err = envOrErr("HOSTNAME")
	if err != nil {
		return
	}
	a.self.address = fmt.Sprintf("%s.%s.%s.svc.%s", a.self.name, a.serviceName, a.namespace, a.dnsClusterSuffix)

	zap.S().Debugf("Running as %s within Statefulset %s of %d replicas, with headless service %s.%s.svc.%s", a.self.address, a.name, a.replicas, a.serviceName, a.namespace, a.dnsClusterSuffix)
	return
}

func (a *sts) AutoScalingGroupStatus() ([]asg.Instance, asg.Instance, int, error) {
	instances := make([]asg.Instance, 0, a.replicas)
	instancesStr := make([]string, 0, a.replicas)

	for i:=0; i<a.replicas; i++ {
		instance := instance{
			name: fmt.Sprintf("%s-%d", a.name, i),
			address: fmt.Sprintf("%s-%d.%s.%s.svc.%s", a.name, i, a.serviceName, a.namespace, a.dnsClusterSuffix),
		}
		instances = append(instances, &instance)
		instancesStr = append(instancesStr, instance.address)
	}
	zap.S().Debugf("Discovered %d / %d replicas: %s", len(instances), a.replicas, strings.Join(instancesStr, ", "))

	return instances, &a.self, a.replicas, nil
}

func envOrErr(key string) (value string, err error) {
	value = os.Getenv(key)
	if value == "" {
		err = fmt.Errorf("Required environment variable: %s was not set", key)
	}

	return
}
