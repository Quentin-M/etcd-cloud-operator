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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func init() {
	asg.Register("sts", &sts{})
}

type sts struct {
	asgName, instanceID string
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

func (a *sts) Configure(providerConfig asg.Config) error {
	return nil
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

	log.Infof("Connected to Kubernetes API")

	namespace := os.Getenv("STATEFULSET_NAMESPACE")
	name := os.Getenv("STATEFULSET_NAME")
	podAddress := os.Getenv("POD_IP")
	podName := os.Getenv("HOSTNAME")

	log.Infof("Running in pod: %s owned by statefulset: %s in namespace: %s with ip: %s",
		podName, name, namespace, podAddress)

	s, err := cs.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{})

	if err != nil {
		log.Errorf("There was an error retrieving statefulset: %s", err)
		return instances, self, size, err
	}

	size = int(s.Status.Replicas)
	self = &instance{name: podName, address: podAddress}

	for i := 0; i < size; i++ {
		p := fmt.Sprintf("%s-%v", name, i)
		instance := instanceFromPod(cs, p, namespace)
		instances = append(instances, instance)
	}
	return instances, self, size, nil
}

func instanceFromPod(cs *kubernetes.Clientset, name string, namespace string) asg.Instance {
	pod, _ := cs.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	address := pod.Status.PodIP
	log.Infof("Identifying peer: %s at address: %s", name, address)
	return &instance{name: name, address: address}
}
