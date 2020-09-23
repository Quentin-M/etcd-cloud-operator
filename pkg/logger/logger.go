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

package logger

import (
	log "github.com/sirupsen/logrus"
	"go.etcd.io/etcd/embed"
	"go.uber.org/zap"
)

// var lbConnEOF = regexp.MustCompile(`embed: rejected connection from "[^"]*" \(error "EOF", ServerName ""\)`)

// BuildZapLogger builds a full zap.Logger that can be used in the server.
func BuildZapLogger(lvl string) *zap.Logger {
	// Mute ClientV3 hammering broken endpoints, which is common in ECO.
	//
	// Unfortunately, the etcd clientv3 configuration only takes a zap.Config, rather than a zapcore.Core, which means
	// we cannot filter chatty logs such as `"caller":"clientv3/retry_interceptor.go:62"`. At least, we can build a
	// generic filter to reference purposes - in case etcd decides to accepts a proper zap.Logger.
	//
	// etcd ref: https://github.com/etcd-io/etcd/issues/12326
	//
	//zl, err := BuildZapConfig(lvl).Build(zap.WrapCore(func (z zapcore.Core) zapcore.Core {
	//	return NewFilteringCore(z, func(e zapcore.Entry) bool {
	//		return e.Caller.File != "clientv3/retry_interceptor.go"
	//	})
	//}))

	// Mute bogus LB Health Checks connections.
	// Unfortunately, the etcd embed server only take a zap.Config rather than a zapcore.Core, which means we cannot
	// filter chatty logs such as `embed: rejected connection from "10.3.65.87:6298" (error "EOF", ServerName "")`
	// At least, we can build a generic filter to reference purposes - in case etcd decides to accepts a proper
	// zap.Logger.
	//
	// etcd ref: https://github.com/etcd-io/etcd/issues/12326
	//
	//zl, err := BuildZapConfig(lvl).Build(zap.WrapCore(func(z zapcore.Core) zapcore.Core {
	//	return NewFilteringCore(z, func(e zapcore.Entry) bool {
	//		return !lbConnEOF.MatchString(e.Message)
	//	})
	//}))

	zl, err := BuildZapConfig(lvl).Build()
	if err != nil {
		log.WithError(err).Fatal("unable to parse zap's logging level")
	}
	return zl
}

// BuildZapConfig creates a uniform zap.Config for the etcd server & client.
func BuildZapConfig(lvl string) *zap.Config {
	zlCfg := zap.NewProductionConfig()
	zlCfg.Level.UnmarshalText([]byte(lvl))
	zlCfg.Sampling = nil
	zlCfg.Encoding = "console"
	return &zlCfg
}

// BuildZapConfigBuilder returns a configuration builder for the etcd server.
//
// Given the current complexity of the builder and the irreproducibility of the default configuration, we better
// stay away from overriding it for now and until a better API is exposed. We can handle the extra noise, but we can
// keep the wiring.
//
// etcd ref: https://github.com/etcd-io/etcd/pull/11147/files
// etcd ref: https://github.com/etcd-io/etcd/issues/12326
func BuildZapConfigBuilder() func(*embed.Config) error {
	return nil
}
