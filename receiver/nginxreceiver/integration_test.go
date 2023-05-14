// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build integration
// +build integration

package nginxreceiver

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/scraperinttest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
)

func nginxContainer(t *testing.T) testcontainers.Container {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    filepath.Join("testdata", "integration"),
			Dockerfile: "Dockerfile.nginx",
		},
		ExposedPorts: []string{"8080:80"},
		WaitingFor:   wait.ForListeningPort("80"),
	}

	require.NoError(t, req.Validate())

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	time.Sleep(time.Second * 6)
	return container
}

func TestNginxIntegration(t *testing.T) {
	t.Parallel()
	container := nginxContainer(t)
	defer func() {
		require.NoError(t, container.Terminate(context.Background()))
	}()
	hostname, err := container.Host(context.Background())
	require.NoError(t, err)

	expectedFile := filepath.Join("testdata", "integration", "expected.yaml")
	expectedMetrics, err := golden.ReadMetrics(expectedFile)
	require.NoError(t, err)

	f := NewFactory()
	cfg := f.CreateDefaultConfig().(*Config)
	cfg.ScraperControllerSettings.CollectionInterval = 100 * time.Millisecond
	cfg.Endpoint = fmt.Sprintf("http://%s:8080/status", hostname)

	consumer := new(consumertest.MetricsSink)
	settings := receivertest.NewNopCreateSettings()

	rcvr, err := f.CreateMetricsReceiver(context.Background(), settings, cfg, consumer)
	require.NoError(t, err, "failed creating metrics receiver")

	require.NoError(t, rcvr.Start(context.Background(), componenttest.NewNopHost()))
	defer func() {
		require.NoError(t, rcvr.Shutdown(context.Background()))
	}()

	compareOpts := []pmetrictest.CompareMetricsOption{
		pmetrictest.IgnoreMetricValues(),
		pmetrictest.IgnoreStartTimestamp(),
		pmetrictest.IgnoreTimestamp(),
	}

	require.Eventually(t, scraperinttest.EqualsLatestMetrics(expectedMetrics, consumer, compareOpts), 30*time.Second, 1*time.Second)
}
