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

package aws

import (
	"errors"
	"fmt"
	"strings"

	aaws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/asg"
)

func init() {
	asg.Register("aws", &aws{})
}

type aws struct {
	asgName, instanceID, region string
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

func (a *aws) Configure(providerConfig asg.Config) error {
	// Fetch the underlying auto-scaling group once to verify the app is
	// indeed running on one, and cache its name.
	_, _, err := a.describeASG()
	if err != nil {
		return err
	}

	return nil
}

func (a *aws) AutoScalingGroupStatus() (instances []asg.Instance, self, asgLeader asg.Instance, asgScaled bool, err error) {
	asg, reservations, err := a.describeASG()
	if err != nil {
		return nil, nil, nil, false, err
	}

	for _, reservation := range reservations {
		for _, awsInstance := range reservation.Instances {
			if strings.ToLower(*awsInstance.State.Name) != "running" {
				continue
			}

			instance := &instance{name: *awsInstance.InstanceId, address: *awsInstance.PrivateIpAddress}
			instances = append(instances, instance)

			if instance.name == a.instanceID {
				self = instance
			}
			if asgLeader == nil || strings.Compare(instance.name, asgLeader.Name()) < 0 {
				asgLeader = instance
			}
		}
	}
	asgScaled = int(*asg.DesiredCapacity) == len(instances)

	return
}

func (a *aws) describeASG() (*autoscaling.Group, []*ec2.Reservation, error) {
	if a.region == "" || a.instanceID == "" {
		sess, err := session.NewSession()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create aws session: %v", err)
		}
		
		ec2meta := ec2metadata.New(sess)
		if !ec2meta.Available() {
			return nil, nil, errors.New("application is not running on aws ec2")
		}
		
		instanceIdentity, err := ec2meta.GetInstanceIdentityDocument()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to retrieve aws ec2 instance identity: %v", err)
		}
		a.instanceID = instanceIdentity.InstanceID
		
		a.region, err = ec2meta.Region()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to retrieve aws ec2 region: %v", err)
		}
	}
	
	sess, err := session.NewSession(aaws.NewConfig().WithRegion(a.region))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create aws session: %v", err)
	}
	as := autoscaling.New(sess)
	ec2s := ec2.New(sess)
	
	if a.asgName == "" {
		asgInstance, err := as.DescribeAutoScalingInstances(&autoscaling.DescribeAutoScalingInstancesInput{
			InstanceIds: []*string{aaws.String(a.instanceID)},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to retrieve aws auto-scaling group: %v", err)
		}
		if len(asgInstance.AutoScalingInstances) == 0 {
			return nil, nil, errors.New("application is not running inside an aws ec2 auto-scaling group")
		}
		a.asgName = *asgInstance.AutoScalingInstances[0].AutoScalingGroupName
	}

	asg, err := as.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aaws.String(a.asgName)},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to describe aws auto-scaling group: %v", err)
	}

	reservations, err := ec2s.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aaws.String("tag:aws:autoscaling:groupName"),
				Values: []*string{aaws.String(a.asgName)},
			},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to describe aws auto-scaling group's instances: %v", err)
	}

	return asg.AutoScalingGroups[0], reservations.Reservations, nil
}
