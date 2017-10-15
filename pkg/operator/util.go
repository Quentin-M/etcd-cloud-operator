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
	"fmt"
	"strconv"
	"strings"

	"github.com/quentin-m/etcd-cloud-operator/pkg/etcd"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/asg"
	log "github.com/sirupsen/logrus"
)

func instancesAddresses(instances []asg.Instance) (addresses []string) {
	for _, instance := range instances {
		addresses = append(addresses, instance.Address())
	}
	return
}

func peerAddresses(members map[string]etcd.Member, self asg.Instance) map[string]string {
	peers := make(map[string]string, len(members))
	for memberName, member := range members {
		if member.Healthy {
			peers[memberName] = member.PeerAddress
		}
	}
	if self != nil {
		peers[self.Name()] = self.Address()
	}
	return peers
}

func stringOverride(s, override string) string {
	if override != "" {
		return override
	}
	return s
}

func toMB(s int64) float64 {
	sn := fmt.Sprintf("%.3f", float64(s)/(1024*1024))
	n, _ := strconv.ParseFloat(sn, 64)
	return n
}

func printCluster(asgInstances []asg.Instance, asgSelf, asgLeader asg.Instance, asgScaled bool, etcdMembers map[string]etcd.Member, etcdQuorum bool, etcdRunning bool) {
	stringOrPeriod := func(s string, c bool) string {
		if c {
			return s
		}
		return strings.Repeat(".", len(s))
	}
	type instanceMember struct {
		address                         string
		member, instance, lead, healthy bool
	}

	imM := make(map[string]*instanceMember)
	for _, asgInstance := range asgInstances {
		imM[asgInstance.Name()] = &instanceMember{
			address:  asgInstance.Address(),
			instance: true,
			lead:     asgInstance.Name() == asgLeader.Name(),
		}
	}
	for memberName, member := range etcdMembers {
		if _, ok := imM[memberName]; !ok {
			imM[memberName] = &instanceMember{address: member.PeerAddress}
		}
		imM[memberName].member = true
		imM[memberName].healthy = member.Healthy
	}

	imStrings := make([]string, 0)
	var healthyMembersCount int
	for name, im := range imM {
		imStrings = append(imStrings, fmt.Sprintf("(%s %s %s%s%s|%s)",
			name,
			im.address,
			stringOrPeriod("I", im.instance),
			stringOrPeriod("M", im.member),
			stringOrPeriod("L", im.lead),
			stringOrPeriod("H", im.healthy),
		))
		if im.healthy {
			healthyMembersCount++
		}
	}

	log.Infof(
		"CLUSTER: [%d/%d %s%s|%s%s] %s",
		healthyMembersCount,
		len(etcdMembers),
		stringOrPeriod("R", etcdRunning),
		stringOrPeriod("Q", etcdQuorum),
		stringOrPeriod("S", asgScaled),
		stringOrPeriod("L", asgLeader.Name() == asgSelf.Name()),
		strings.Join(imStrings, " "),
	)
}
