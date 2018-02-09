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
	"strings"
	"time"

	etcdcl "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/transport"
)

const (
	defaultClientPort  = 2379
	defaultPeerPort    = 2380
	defaultMetricsPort = 2381

	defaultDialTimeout    = 5 * time.Second
	defaultRequestTimeout = 5 * time.Second
	defaultAutoSync       = 5 * time.Second
)

// EtcdConfiguration contains the configuration related to the underlying etcd
// server.
type EtcdConfiguration struct {
	AdvertiseAddress        string         `yaml:"advertise-address"`
	DataDir                 string         `yaml:"data-dir"`
	ClientTransportSecurity SecurityConfig `yaml:"client-transport-security"`
	PeerTransportSecurity   SecurityConfig `yaml:"peer-transport-security"`
	BackendQuota            int64          `yaml:"backend-quota"`
}

type SecurityConfig struct {
	CAFile        string `yaml:"ca-file"`
	CertFile      string `yaml:"cert-file"`
	KeyFile       string `yaml:"key-file"`
	CertAuth      bool   `yaml:"client-cert-auth"`
	TrustedCAFile string `yaml:"trusted-ca-file"`
	AutoTLS       bool   `yaml:"auto-tls"`
}

func (sc SecurityConfig) TLSInfo() transport.TLSInfo {
	return transport.TLSInfo{
		CAFile:         sc.CAFile,
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

func newClient(clientsURLs []string, tlsConfig *tls.Config) (*etcdcl.Client, error) {
	return etcdcl.New(etcdcl.Config{
		Endpoints:        clientsURLs,
		DialTimeout:      defaultDialTimeout,
		TLS:              tlsConfig,
		AutoSyncInterval: defaultAutoSync,
	})
}

func clientsURLs(addresses []string, tlsEnabled bool) (cURLs []string) {
	for _, address := range addresses {
		cURLs = append(cURLs, clientURL(address, tlsEnabled))
	}
	return
}

func clientURL(address string, tlsEnabled bool) string {
	return fmt.Sprintf("%s://%s:%d", scheme(tlsEnabled), address, defaultClientPort)
}

func peerURL(address string, tlsEnabled bool) string {
	return fmt.Sprintf("%s://%s:%d", scheme(tlsEnabled), address, defaultPeerPort)
}

func metricsURLs(address string) []url.URL {
	u, _ := url.Parse(fmt.Sprintf("http://%s:%d", address, defaultMetricsPort))
	return []url.URL{*u}
}

func initialCluster(addresses map[string]string, tlsEnabled bool) string {
	var ic []string
	for name, address := range addresses {
		ic = append(ic, fmt.Sprintf("%s=%s://%s:%d", name, scheme(tlsEnabled), address, defaultPeerPort))
	}
	return strings.Join(ic, ",")
}

func scheme(tlsEnabled bool) string {
	if tlsEnabled {
		return "https"
	}
	return "http"
}
