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
	"os/exec"
	"strings"
)

type container struct {
	id, name, address string
}

func (c *container) Name() string {
	return c.name
}

func (c *container) Address() string {
	return c.address
}

func containerList(namePattern string) ([]string, error) {
	out, err := exec.Command("docker", "ps", fmt.Sprintf("--filter=name=%s", namePattern), "--format={{ .Names }}").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %s", bytesToTrimmedString(out))
	}
	return strings.Split(bytesToTrimmedString(out), "\n"), nil
}

func containerInspect(name string) (*container, error) {
	out, err := exec.Command("docker", "inspect", "--format={{ .ID}},{{ .Name }},{{ .NetworkSettings.IPAddress }}", name).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %q: %s", name, bytesToTrimmedString(out))
	}
	if outSplit := strings.Split(bytesToTrimmedString(out), ","); len(outSplit) == 3 {
		return &container{id: outSplit[0], name: strings.TrimPrefix(outSplit[1], "/"), address: outSplit[2]}, nil
	}

	return nil, fmt.Errorf("failed to inspect container %q: unexpected output: %s", name, bytesToTrimmedString(out))
}

func bytesToTrimmedString(b []byte) string {
	return strings.TrimSuffix(string(b), "\n")
}
