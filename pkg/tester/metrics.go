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

package tester

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	promRequestsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Number of requests sent to etcd",
		},
	)
	promFailedRequestsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "failed_requests_total",
			Help: "Number of requests sent to etcd that failed",
		},
	)
	promRunningTest = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "running_test",
			Help: "The currently running test",
		},
		[]string{"name"},
	)

	promFailureInjected = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "failure_injected",
			Help: "The currently injected failure",
		},
		[]string{"name"},
	)
)

func promRun() {
	prometheus.MustRegister(promRequestsTotal)
	prometheus.MustRegister(promFailedRequestsTotal)

	prometheus.MustRegister(promRunningTest)
	prometheus.MustRegister(promFailureInjected)

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8000", nil)
}

func promSetRunningTest(name string) {
	for _, t := range testCases {
		promRunningTest.WithLabelValues(t.name).Set(0)
	}
	if name != "" {
		promRunningTest.WithLabelValues(name).Set(1)
	}
}

func promSetInjectedFailure(name string) {
	for _, t := range testCases {
		promFailureInjected.WithLabelValues(t.name).Set(0)
	}
	if name != "" {
		promFailureInjected.WithLabelValues(name).Set(1)
	}
}
