/*
 * Copyright 2018 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"flag"
	"os"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/knative/eventing/pkg/buses"
	"github.com/knative/eventing/pkg/buses/kafka"
	"github.com/knative/pkg/signals"
	"go.uber.org/zap"
)

const (
	threadsPerReconciler = 1
)

func main() {
	busRef := buses.NewBusReferenceFromNames(
		os.Getenv("BUS_NAME"),
		os.Getenv("BUS_NAMESPACE"),
	)

	config := buses.NewLoggingConfig()
	logger := buses.NewBusLoggerFromConfig(config)
	defer logger.Sync()
	logger = logger.With(
		zap.String("channels.knative.dev/bus", busRef.String()),
		zap.String("channels.knative.dev/busType", kafka.BusType),
		zap.String("channels.knative.dev/busComponent", buses.Dispatcher),
	)
	sarama.Logger = zap.NewStdLog(logger.With(zap.Namespace("Sarama")).Desugar())

	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(brokers) == 0 {
		logger.Fatalf("Environment variable KAFKA_BROKERS not set")
	}

	opts := &buses.BusOpts{
		Logger: logger,
	}

	flag.StringVar(&opts.KubeConfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&opts.MasterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.Parse()

	bus, err := kafka.NewKafkaBusDispatcher(busRef, brokers, opts)
	if err != nil {
		logger.Fatalf("Error starting kafka bus dispatcher: %v", err)
	}

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()
	bus.Run(threadsPerReconciler, stopCh)
}
