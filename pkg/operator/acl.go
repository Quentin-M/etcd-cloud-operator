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
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
	"time"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/quentin-m/etcd-cloud-operator/pkg/etcd"
)

const (
	defaultRequestTimeout = 5 * time.Second
	initACLConfigKeyPath  = "/etcd-cloud-operator/init-acl-config"
)

func (s *Operator) applyACLConfig(ctx context.Context, config *etcd.ACLConfig) error {
	var err error
	for _, role := range config.Roles {
		resp, err := s.etcdClient.RoleGet(ctx, role.Name)
		if err == nil && resp != nil {
			if _, err := s.etcdClient.RoleDelete(ctx, role.Name); err != nil {
				return err
			}
		}
		if _, err := s.etcdClient.RoleAdd(ctx, role.Name); err != nil {
			return err
		}

		for _, perm := range role.Permissions {
			var rangeEnd string
			var mode clientv3.PermissionType

			if perm.Prefix {
				rangeEnd = clientv3.GetPrefixRangeEnd(perm.Key)
			}

			switch strings.ToLower(perm.Mode) {
			case "read":
				mode = clientv3.PermissionType(clientv3.PermRead)
			case "write":
				mode = clientv3.PermissionType(clientv3.PermWrite)
			case "readwrite":
				mode = clientv3.PermissionType(clientv3.PermReadWrite)
			default:
				return fmt.Errorf("invalid permission mode %q", perm.Mode)
			}

			_, err = s.etcdClient.RoleGrantPermission(ctx, role.Name, perm.Key, rangeEnd, mode)
			if err != nil {
				return err
			}
		}
	}

	for _, user := range config.Users {
		var opt clientv3.UserAddOptions

		if user.Password == "" {
			opt.NoPassword = true
		}

		resp, err := s.etcdClient.UserGet(ctx, user.Name)
		if err == nil && resp != nil {
			if _, err := s.etcdClient.UserDelete(ctx, user.Name); err != nil {
				return err
			}
		}
		if _, err := s.etcdClient.UserAddWithOptions(ctx, user.Name, user.Password, &opt); err != nil {
			return err
		}

		for _, role := range user.Roles {
			_, err = s.etcdClient.UserGrantRole(ctx, user.Name, role)
			if err != nil {
				return err
			}
		}
	}

	configBytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	if _, err := s.etcdClient.Put(ctx, initACLConfigKeyPath, string(configBytes)); err != nil {
		return err
	}

	return nil
}

func (s *Operator) removeACLConfig(ctx context.Context, config *etcd.ACLConfig) error {
	for _, user := range config.Users {
		_, err := s.etcdClient.UserDelete(ctx, user.Name)
		if err != nil && err != rpctypes.ErrUserNotFound {
			return err
		}
	}

	for _, role := range config.Roles {
		if _, err := s.etcdClient.RoleDelete(ctx, role.Name); err != nil {
			if err == rpctypes.ErrRoleNotFound {
				continue
			}
			return err
		}
	}

	_, err := s.etcdClient.Delete(ctx, initACLConfigKeyPath)
	if err != nil && err != rpctypes.ErrKeyNotFound {
		return err
	}
	return nil
}

func (s *Operator) reconcileInitACLConfig(config *etcd.ACLConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	if err := s.enableACL(ctx, config); err != nil {
		zap.S().With(zap.Error(err)).Error("failed to enable ACL")
		return err
	}

	oldACLConfig, err := s.getOldACLConfig(ctx)
	if err != nil {
		zap.S().With(zap.Error(err)).Error("failed to get old ACL config")
		return err
	}

	if !oldACLConfig.Equal(config) {
		if oldACLConfig != nil {
			if err := s.removeACLConfig(ctx, oldACLConfig); err != nil {
				zap.S().With(zap.Error(err)).Error("failed to remove ACL config")
				return err
			}
		}
		if err := s.applyACLConfig(ctx, config); err != nil {
			zap.S().With(zap.Error(err)).Error("failed to apply ACL config")
			return err
		}
	}

	return nil
}

func (s *Operator) enableACL(ctx context.Context, config *etcd.ACLConfig) error {
	commonName, err := s.getCertCommonName()
	if err != nil {
		return err
	}

	resp, err := s.etcdClient.UserGet(ctx, commonName)
	if err == nil && resp != nil {
		for _, role := range resp.Roles {
			if role == "root" {
				return nil
			}
		}
		if _, err := s.etcdClient.UserDelete(ctx, commonName); err != nil {
			return err
		}
	}

	_, err = s.etcdClient.RoleAdd(ctx, "root")
	if err != nil && err != rpctypes.ErrRoleAlreadyExist {
		return err
	}

	if config.RootPassword == nil || *config.RootPassword == "" {
		_, err = s.etcdClient.UserAddWithOptions(ctx, "root", "", &clientv3.UserAddOptions{NoPassword: true})
		if err != nil && err != rpctypes.ErrUserAlreadyExist {
			return err
		}
	} else {
		_, err = s.etcdClient.UserAdd(ctx, "root", *config.RootPassword)
		if err != nil && err != rpctypes.ErrUserAlreadyExist {
			return err
		}
	}

	if _, err := s.etcdClient.UserGrantRole(ctx, "root", "root"); err != nil {
		return err
	}

	if _, err := s.etcdClient.UserAddWithOptions(ctx, commonName, "", &clientv3.UserAddOptions{NoPassword: true}); err != nil {
		return err
	}

	if _, err := s.etcdClient.UserGrantRole(ctx, commonName, "root"); err != nil {
		return err
	}

	if _, err := s.etcdClient.UserAddWithOptions(ctx, "etcd", "", &clientv3.UserAddOptions{NoPassword: true}); err != nil {
			return err
	}

	if _, err := s.etcdClient.UserGrantRole(ctx, "etcd", "root"); err != nil {
			return err
	}

	if _, err := s.etcdClient.AuthEnable(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Operator) getOldACLConfig(ctx context.Context) (*etcd.ACLConfig, error) {
	resp, err := s.etcdClient.Get(ctx, initACLConfigKeyPath)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	if len(resp.Kvs) != 1 {
		return nil, fmt.Errorf("expect one kv for %q, got %v", initACLConfigKeyPath, len(resp.Kvs))
	}

	var oldACLConfig etcd.ACLConfig
	if err := yaml.Unmarshal(resp.Kvs[0].Value, &oldACLConfig); err != nil {
		return nil, err
	}
	return &oldACLConfig, nil
}

func (s *Operator) getCertCommonName() (string, error) {
	bytes, err := ioutil.ReadFile(s.cfg.Etcd.ClientTransportSecurity.CertFile)
	if err != nil {
		return "", err
	}

	block, _ := pem.Decode(bytes)
	if block == nil {
		return "", fmt.Errorf("failed to decode cert")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", err
	}

	return cert.Subject.CommonName, nil
}
