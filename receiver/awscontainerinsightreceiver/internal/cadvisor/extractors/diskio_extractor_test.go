// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extractors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/containerinsight"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/awscontainerinsightreceiver/internal/cadvisor/testutils"
)

func TestDiskIOStats(t *testing.T) {

	result := testutils.LoadContainerInfo(t, "./testdata/PreInfoContainer.json")
	result2 := testutils.LoadContainerInfo(t, "./testdata/CurInfoContainer.json")
	// for eks node-level metrics
	containerType := TypeNode
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	extractor := NewDiskIOMetricExtractor(ctx, nil)

	var cMetrics []*CAdvisorMetric
	if extractor.HasValue(result[0]) {
		cMetrics = extractor.GetValue(result[0], nil, containerType)
	}

	if extractor.HasValue(result2[0]) {
		cMetrics = extractor.GetValue(result2[0], nil, containerType)
	}

	expectedFieldsService := map[string]interface{}{
		"node_diskio_io_service_bytes_write": float64(10000),
		"node_diskio_io_service_bytes_total": float64(10010),
		"node_diskio_io_service_bytes_async": float64(10000),
		"node_diskio_io_service_bytes_sync":  float64(10000),
		"node_diskio_io_service_bytes_read":  float64(10),
	}
	expectedFieldsServiced := map[string]interface{}{
		"node_diskio_io_serviced_async": float64(10),
		"node_diskio_io_serviced_sync":  float64(10),
		"node_diskio_io_serviced_read":  float64(10),
		"node_diskio_io_serviced_write": float64(10),
		"node_diskio_io_serviced_total": float64(20),
	}
	expectedTags := map[string]string{
		"device": "/dev/xvda",
		"Type":   "NodeDiskIO",
	}
	AssertContainsTaggedField(t, cMetrics[0], expectedFieldsService, expectedTags)
	AssertContainsTaggedField(t, cMetrics[1], expectedFieldsServiced, expectedTags)

	// for ecs node-level metrics
	containerType = TypeInstance
	extractor = NewDiskIOMetricExtractor(ctx, nil)

	if extractor.HasValue(result[0]) {
		cMetrics = extractor.GetValue(result[0], nil, containerType)
	}

	if extractor.HasValue(result2[0]) {
		cMetrics = extractor.GetValue(result2[0], nil, containerType)
	}

	expectedFieldsService = map[string]interface{}{
		"instance_diskio_io_service_bytes_write": float64(10000),
		"instance_diskio_io_service_bytes_total": float64(10010),
		"instance_diskio_io_service_bytes_async": float64(10000),
		"instance_diskio_io_service_bytes_sync":  float64(10000),
		"instance_diskio_io_service_bytes_read":  float64(10),
	}
	expectedFieldsServiced = map[string]interface{}{
		"instance_diskio_io_serviced_async": float64(10),
		"instance_diskio_io_serviced_sync":  float64(10),
		"instance_diskio_io_serviced_read":  float64(10),
		"instance_diskio_io_serviced_write": float64(10),
		"instance_diskio_io_serviced_total": float64(20),
	}
	expectedTags = map[string]string{
		"device": "/dev/xvda",
		"Type":   "InstanceDiskIO",
	}
	AssertContainsTaggedField(t, cMetrics[0], expectedFieldsService, expectedTags)
	AssertContainsTaggedField(t, cMetrics[1], expectedFieldsServiced, expectedTags)

	// for non supported type
	containerType = TypeContainerDiskIO
	extractor = NewDiskIOMetricExtractor(ctx, nil)

	if extractor.HasValue(result[0]) {
		cMetrics = extractor.GetValue(result[0], nil, containerType)
	}

	if extractor.HasValue(result2[0]) {
		cMetrics = extractor.GetValue(result2[0], nil, containerType)
	}

	assert.Equal(t, len(cMetrics), 0)
}
