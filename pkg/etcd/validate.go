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

package etcd

import (
	"fmt"
)

func (c *EtcdConfiguration) Validate() error {
	if c.InitACL != nil {
		if !c.ClientTransportSecurity.CertAuth {
			return fmt.Errorf("--client-cert-auth must be set to true to enable the initial ACL config")
		}

		roleNames, userNames := make(map[string]struct{}), make(map[string]struct{})

		for _, role := range c.InitACL.Roles {
			if role.Name == "" {
				return fmt.Errorf("empty role name")
			}

			if len(role.Permissions) == 0 {
				return fmt.Errorf("empty permissions for role %q", role.Name)
			}

			for _, perm := range role.Permissions {
				if perm.Mode == "" {
					return fmt.Errorf("empty permission 'mode' for role %q", role.Name)
				}
				if perm.Key == "" {
					return fmt.Errorf("empty permission 'key' for role %q", role.Name)
				}
			}

			if _, ok := roleNames[role.Name]; ok {
				return fmt.Errorf("duplicated role name %q", role.Name)
			}
			roleNames[role.Name] = struct{}{}
		}

		for _, user := range c.InitACL.Users {
			if user.Name == "" {
				return fmt.Errorf("empty user name")
			}

			for _, role := range user.Roles {
				if _, ok := roleNames[role]; !ok && role != "root" {
					return fmt.Errorf("role %q not existed", role)
				}
			}

			if _, ok := userNames[user.Name]; ok {
				return fmt.Errorf("duplicated user name %q", user.Name)
			}
			userNames[user.Name] = struct{}{}
		}
	}

	return nil
}
