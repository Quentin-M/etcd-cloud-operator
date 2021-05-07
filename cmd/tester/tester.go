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

// Package main implements basic logic to start the etcd-cloud-operator.
package main

import (
	"flag"
	"os"

	"go.uber.org/zap"

	"github.com/quentin-m/etcd-cloud-operator/pkg/logger"
	"github.com/quentin-m/etcd-cloud-operator/pkg/tester"
)

func main() {
	// Parse command-line arguments.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagConfigPath := flag.String("config", "", "Load configuration from the specified file.")
	flagLogLevel := flag.String("log-level", "info", "Define the logging level.")
	flag.Parse()

	// Initialize logging system.
	logger.Configure(*flagLogLevel)

	// Read configuration.
	config, err := loadConfig(*flagConfigPath)
	if err != nil {
		zap.S().With(zap.Error(err)).Fatal("failed to load configuration")
	}

	// Run.
	tester.Run(config.ECOTester)
}
