// Code generated by mdatagen. DO NOT EDIT.

package metadata

import "go.opentelemetry.io/collector/confmap"

// MetricConfig provides common config for a particular metric.
type MetricConfig struct {
	Enabled bool `mapstructure:"enabled"`

	enabledSetByUser bool
}

func (ms *MetricConfig) Unmarshal(parser *confmap.Conf) error {
	if parser == nil {
		return nil
	}
	err := parser.Unmarshal(ms, confmap.WithErrorUnused())
	if err != nil {
		return err
	}
	ms.enabledSetByUser = parser.IsSet("enabled")
	return nil
}

// MetricsConfig provides config for couchdbreceiver metrics.
type MetricsConfig struct {
	CouchdbAverageRequestTime MetricConfig `mapstructure:"couchdb.average_request_time"`
	CouchdbDatabaseOpen       MetricConfig `mapstructure:"couchdb.database.open"`
	CouchdbDatabaseOperations MetricConfig `mapstructure:"couchdb.database.operations"`
	CouchdbFileDescriptorOpen MetricConfig `mapstructure:"couchdb.file_descriptor.open"`
	CouchdbHttpdBulkRequests  MetricConfig `mapstructure:"couchdb.httpd.bulk_requests"`
	CouchdbHttpdRequests      MetricConfig `mapstructure:"couchdb.httpd.requests"`
	CouchdbHttpdResponses     MetricConfig `mapstructure:"couchdb.httpd.responses"`
	CouchdbHttpdViews         MetricConfig `mapstructure:"couchdb.httpd.views"`
}

func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		CouchdbAverageRequestTime: MetricConfig{
			Enabled: true,
		},
		CouchdbDatabaseOpen: MetricConfig{
			Enabled: true,
		},
		CouchdbDatabaseOperations: MetricConfig{
			Enabled: true,
		},
		CouchdbFileDescriptorOpen: MetricConfig{
			Enabled: true,
		},
		CouchdbHttpdBulkRequests: MetricConfig{
			Enabled: true,
		},
		CouchdbHttpdRequests: MetricConfig{
			Enabled: true,
		},
		CouchdbHttpdResponses: MetricConfig{
			Enabled: true,
		},
		CouchdbHttpdViews: MetricConfig{
			Enabled: true,
		},
	}
}

// ResourceAttributeConfig provides common config for a particular resource attribute.
type ResourceAttributeConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// ResourceAttributesConfig provides config for couchdbreceiver resource attributes.
type ResourceAttributesConfig struct {
	CouchdbNodeName ResourceAttributeConfig `mapstructure:"couchdb.node.name"`
}

func DefaultResourceAttributesConfig() ResourceAttributesConfig {
	return ResourceAttributesConfig{
		CouchdbNodeName: ResourceAttributeConfig{
			Enabled: true,
		},
	}
}

// MetricsBuilderConfig is a configuration for couchdbreceiver metrics builder.
type MetricsBuilderConfig struct {
	Metrics            MetricsConfig            `mapstructure:"metrics"`
	ResourceAttributes ResourceAttributesConfig `mapstructure:"resource_attributes"`
}

func DefaultMetricsBuilderConfig() MetricsBuilderConfig {
	return MetricsBuilderConfig{
		Metrics:            DefaultMetricsConfig(),
		ResourceAttributes: DefaultResourceAttributesConfig(),
	}
}
