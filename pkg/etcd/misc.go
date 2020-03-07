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
	"crypto/tls"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.etcd.io/etcd/pkg/transport"

	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/snapshot"
)

const (
	defaultClientPort  = 2379
	defaultPeerPort    = 2380
	defaultMetricsPort = 2381

	defaultDialTimeout    = 5 * time.Second
	defaultRequestTimeout = 5 * time.Second
	defaultAutoSync       = 1 * time.Second
)

// EtcdConfiguration contains the configuration related to the underlying etcd
// server.
type EtcdConfiguration struct {
	AdvertiseAddress        string         `yaml:"advertise-address"`
	DataDir                 string         `yaml:"data-dir"`
	ClientTransportSecurity SecurityConfig `yaml:"client-transport-security"`
	PeerTransportSecurity   SecurityConfig `yaml:"peer-transport-security"`
	BackendQuota            int64          `yaml:"backend-quota"`
	InitACL                 *ACLConfig     `yaml:"init-acl,omitempty"`
}

type SecurityConfig struct {
	CertFile      string `yaml:"cert-file"`
	KeyFile       string `yaml:"key-file"`
	CertAuth      bool   `yaml:"client-cert-auth"`
	TrustedCAFile string `yaml:"trusted-ca-file"`
	AutoTLS       bool   `yaml:"auto-tls"`
}

// ACLConfig defines the acl configuration for etcd,
// which will be applied to the etcd during provisioning.
// --client-cert-auth must be set to true.
type ACLConfig struct {
	RootPassword *string `yaml:"rootPassword,omitempty"`
	Roles        []Role  `yaml:"roles"`
	Users        []User  `yaml:"users"`
}

// Role defines an etcd ACL role with its permissions.
type Role struct {
	Name        string       `yaml:"name"`
	Permissions []Permission `yaml:"permissions"`
}

// Users defines an etcd ACL user with its password(optional) and binding roles.
type User struct {
	Name     string   `yaml:"name"`
	Password string   `yaml:"password"`
	Roles    []string `yaml:"roles"`
}

// Permission defines the permission.
type Permission struct {
	Mode     string `yaml:"mode"`
	Key      string `yaml:"key"`
	RangeEnd string `yaml:"rangeEnd"`
	Prefix   bool   `yaml:"prefix"`
}

func (sc SecurityConfig) TLSInfo() transport.TLSInfo {
	return transport.TLSInfo{
		CertFile:       sc.CertFile,
		KeyFile:        sc.KeyFile,
		ClientCertAuth: sc.CertAuth,
		TrustedCAFile:  sc.TrustedCAFile,
	}
}

func (sc SecurityConfig) ClientConfig() (*tls.Config, error) {
	// Because of the nature of the operator, that relies on auto-scaling groups,
	// it is inconvenient (at best) to generate certificates that will match the
	// instances. Therefore, we also use InsecureSkipVerify and make the
	// assumption that the instances present in the auto-scaling group can be
	// trusted.
	if !sc.TLSInfo().Empty() {
		tc, err := sc.TLSInfo().ClientConfig()
		if err != nil {
			return nil, err
		}
		tc.InsecureSkipVerify = true
		return tc, nil
	}
	return &tls.Config{InsecureSkipVerify: true}, nil
}

func (sc SecurityConfig) TLSEnabled() bool {
	return sc.AutoTLS || !sc.TLSInfo().Empty()
}

func ClientsURLs(addresses []string, tlsEnabled bool) (cURLs []string) {
	for _, address := range addresses {
		cURLs = append(cURLs, ClientURL(address, tlsEnabled))
	}
	return
}

func ClientURL(address string, tlsEnabled bool) string {
	return fmt.Sprintf("%s://%s:%d", scheme(tlsEnabled), address, defaultClientPort)
}

func peerURL(address string, tlsEnabled bool) string {
	return fmt.Sprintf("%s://%s:%d", scheme(tlsEnabled), address, defaultPeerPort)
}

func URL2Address(pURL string) string {
	pURLu, _ := url.Parse(pURL)
	if i := strings.Index(pURLu.Host, ":"); i > 0 {
		return pURLu.Host[:i]
	}
	return pURLu.Host
}

func metricsURLs(address string) []url.URL {
	u, _ := url.Parse(fmt.Sprintf("http://%s:%d", address, defaultMetricsPort))
	return []url.URL{*u}
}

func initialCluster(pURLs map[string]string) string {
	var ic []string
	for name, pURL := range pURLs {
		ic = append(ic, fmt.Sprintf("%s=%s", name, pURL))
	}
	return strings.Join(ic, ",")
}

func scheme(tlsEnabled bool) string {
	if tlsEnabled {
		return "https"
	}
	return "http"
}

func toMB(s int64) float64 {
	sn := fmt.Sprintf("%.2f", float64(s)/(1024*1024))
	n, _ := strconv.ParseFloat(sn, 64)
	return n
}

func getSameValue(vals map[string]int64) bool {
	var rv int64
	for _, v := range vals {
		if rv == 0 {
			rv = v
		}
		if rv != v {
			return false
		}
	}
	return true
}

func localSnapshotProvider(dataDir string) snapshot.Provider {
	lsp := snapshot.AsMap()["etcd"]
	lsp.Configure(snapshot.Config{Params: map[string]interface{}{"data-dir": dataDir}})
	return lsp
}

func (a *ACLConfig) Equal(b *ACLConfig) bool {
	return reflect.DeepEqual(a, b)
}
