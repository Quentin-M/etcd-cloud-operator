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
	"fmt"
	"os"

	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/asg"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var i instance

func init() {
	asg.Register("sts", &sts{})
}

type sts struct {
	name, namespace string
}

type instance struct {
	id, name, address string
}

func (i *instance) Name() string {
	return i.name
}

func (i *instance) Address() string {
	return i.address
}

func (a *sts) Configure(providerConfig asg.Config) (err error) {
	a.name, err = envOrErr("STATEFULSET_NAME")
	if err != nil {
		return
	}
	a.namespace, err = envOrErr("STATEFULSET_NAMESPACE")
	if err != nil {
		return
	}

	i.name, err = envOrErr("HOSTNAME")
	if err != nil {
		return
	}

	i.address, err = envOrErr("POD_IP")

	return
}

func (a *sts) AutoScalingGroupStatus() (instances []asg.Instance, self asg.Instance, size int, err error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	log.Debugf("Running in pod: %s owned by statefulset: %s in namespace: %s with ip: %s",
		i.name, a.name, a.namespace, i.address)

	self = &i

	s, err := cs.AppsV1().StatefulSets(a.namespace).Get(a.name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("There was an error retrieving statefulset: %s", err)
		return
	}

	size = int(*s.Spec.Replicas)
	log.Debugf("Determined desired cluster size: %v", size)

	instances, err = getInstancesFromStatefulSet(s, cs, a.namespace, a.name)
	if err != nil {
		log.Errorf("There was a problem getting instances from the statefulset: %s", err)
	}

	log.Debugf("Instances: %+v, Self: %+v, Size: %v", instances, self, size)

	return
}

func getInstancesFromStatefulSet(s *v1.StatefulSet, cs *kubernetes.Clientset, namespace, name string) (instances []asg.Instance, err error) {
	selector, err := metav1.LabelSelectorAsSelector(s.Spec.Selector)
	if err != nil {
		return
	}

	replicas, err := cs.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return
	}

	for _, r := range replicas.Items {
		log.Debugf("Identified peer %s at IP: %s", r.Name, r.Status.PodIP)
		instances = append(instances, &instance{name: r.Name, address: r.Status.PodIP})
	}

	return
}

func envOrErr(key string) (value string, err error) {
	value = os.Getenv(key)
	if value == "" {
		err = fmt.Errorf("Required environment variable: %s was not set", key)
	}

	return
}
