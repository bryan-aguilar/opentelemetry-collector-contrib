// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prometheus // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/prometheus"

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/common/testutil"
)

func TestSanitize(t *testing.T) {

	defer testutil.SetFeatureGateForTest(t, dropSanitizationGate, false)()
	assert.Equal(t, "__test__", NormalizeLabel("__test__))"))
	assert.Equal(t, "test__", NormalizeLabel("test__))"))
}

func TestSanitizeDropSanitization(t *testing.T) {

	defer testutil.SetFeatureGateForTest(t, dropSanitizationGate, true)()

	require.Equal(t, "", NormalizeLabel(""))
	require.Equal(t, "_test", NormalizeLabel("_test"))
	require.Equal(t, "key_0test", NormalizeLabel("0test"))
	require.Equal(t, "test", NormalizeLabel("test"))
	require.Equal(t, "__test", NormalizeLabel("__test"))
	require.Equal(t, "__test__", NormalizeLabel("__test__"))
}
