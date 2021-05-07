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
	"go.etcd.io/etcd/client/pkg/v3/logutil"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"io/ioutil"
	"moul.io/zapfilter"
	"os"
	"regexp"
	"strings"

	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var lbConnEOF = regexp.MustCompile(`embed: rejected connection from "[^"]*" \(error "EOF", ServerName ""\)`)

var Config *zap.Config
var Logger *zap.Logger

func Configure(lvl string) {
	// Build logger configuration
	buildZapLogger(lvl)

	zap.ReplaceGlobals(Logger)
	if lvl != "debug" {
		grpclog.SetLoggerV2(grpclog.NewLoggerV2(ioutil.Discard, ioutil.Discard, os.Stderr))
	} else {
		grpc.EnableTracing = true
		grpclog.SetLoggerV2(zapgrpc.NewLogger(Logger))
	}
}

// buildZapLogger builds a *zap.Logger end-to-end based on the specified logging level, and stores the logger alongside
// his configuration in global variables, so they can be referred to from other libs who configure logging
// independently.
func buildZapLogger(lvl string) {
	if Logger != nil {
		return
	}

	// Build configuration, the same way etcd's embed package does it; as it is currently not possible to set a
	// *zap.Logger directly into embed, and we'd like to be consistent. https://github.com/etcd-io/etcd/issues/12326
	zlCfg := logutil.DefaultZapLoggerConfig
	zlCfg.ErrorOutputPaths = []string{"stdout"}
	zlCfg.OutputPaths = []string{"stdout"}
	zlCfg.Level = zap.NewAtomicLevelAt(logutil.ConvertToZapLevel(lvl))
	zlCfg.Development = lvl == "debug"
	zlCfg.Sampling = nil
	Config = &zlCfg

	// Build logger.
	zl, err := Config.Build(zap.WrapCore(func (z zapcore.Core) zapcore.Core {
		return zapfilter.NewFilteringCore(z, func(e zapcore.Entry, f []zapcore.Field) bool {
			// Mute ClientV3 hammering broken endpoints, which is common in ECO and bogus LB health checks connections.
			return !strings.Contains(e.Message, "retrying of unary invoker failed") &&
				!strings.Contains(e.Message, "Auto sync endpoints failed") &&
				!lbConnEOF.MatchString(e.Message)})
	}))
	if err != nil {
		zap.S().With(zap.Error(err)).Fatal("unable to build logger")
	}
	Logger = zl
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
