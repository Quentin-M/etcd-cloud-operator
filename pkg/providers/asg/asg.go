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

package asg

import (
	"sync"
)

var (
	providers  = make(map[string]Provider)
	providersM sync.RWMutex
)

type Instance interface {
	Name() string
	Address() string
}

type Provider interface {
	Configure(Config) error

	AutoScalingGroupStatus() ([]Instance, Instance, Instance, bool, error)
}

// Config represents the configuration of the auto-scaling group provider.
type Config struct {
	Provider string                 `yaml:"provider"`
	Params   map[string]interface{} `yaml:",inline"`
}

// Register makes a Provider available by the provided name.
//
// If called twice with the same name, the name is blank, or if the provided
// Provider is nil, this function panics.
func Register(name string, p Provider) {
	if name == "" {
		panic("provider: could not register an Provider with an empty name")
	}

	if p == nil {
		panic("provider: could not register a nil Provider")
	}

	providersM.Lock()
	defer providersM.Unlock()

	if _, dup := providers[name]; dup {
		panic("provider: RegisterUpdater called twice for " + name)
	}

	providers[name] = p
}

// AsMap returns a map representing the registered Providers.
func AsMap() map[string]Provider {
	providersM.RLock()
	defer providersM.RUnlock()

	ret := make(map[string]Provider)
	for k, v := range providers {
		ret[k] = v
	}

	return ret
}

// AsList returns the names of registered providers.
func AsList() []string {
	r := []string{}
	for u := range providers {
		r = append(r, u)
	}
	return r
}
