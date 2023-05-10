package testbed

import (
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/idutils"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// samplingDataProvider is an implementation of DataProvider for use in trace tests
// This data provider will generator some traces that fall within the provided option set
// and some data that does not. This is useful for testing filter and sampling processors.
type samplingDataProvider struct {
	dataItemsGenerated *atomic.Uint64
	// It may not make sense for this to be a map. A map will allow us to easily attach all polluted attributes
	// at once but may make it hard if we only want to apply a subset of attributes to a trace.
	sampledAttr           pcommon.Map
	expectedSampledTraces []ptrace.Traces
	// I don't know if this belongs in the data provider. The test may be able to control when to start
	// and stop load. Until then, we just generate data until told to stop.
	numTracesToGenerate int
	// consider using LoadOptions for some configurations.
	loadOptions     LoadOptions
	traceIDSequence atomic.Uint64
	//try to get rid of this and use traceIDSequence instead as a counter
	tracesGenerated int
	randGenerator   *rand.Rand
	// we will use this to ensure that the received spans matches the expected sampled spans
	sampledSpansGenerated atomic.Uint64
}

// SamplingDataProviderOption defines a PollutedDataProvider option.
type SamplingDataProviderOption func(t *samplingDataProvider)

// WithStringAttribute will ensure that some traces include the provided attribute key value pair.
// Question: Should the percentage of traces generated be configurable here?
func WithStringAttribute(key string, value string) SamplingDataProviderOption {
	return func(tc *samplingDataProvider) {
		tc.sampledAttr.EnsureCapacity(tc.sampledAttr.Len() + 1)
		tc.sampledAttr.PutStr(key, value)
	}
}

// TODO: I'm not sure if this is needed yet. I will hard code this in NewSamplingDataProvider for proof of concept
/*
func WithLoadOptions(options LoadOptions) SamplingDataProviderOption {
	return func(tc *samplingDataProvider) {
		tc.loadOptions = options
	}
}
*/

// NewSamplingDataProvider creates a new instance of samplingDataProvider
func NewSamplingDataProvider(opts ...SamplingDataProviderOption) DataProvider {
	pdp := samplingDataProvider{}
	//TODO: figure out the best place to control load options. This should be a single call for this. Currently this is
	// called in NewSamplingDataProvider and also in tc.StartLoad in correctness test package
	pdp.loadOptions = LoadOptions{
		// for now this controls the amount of spans for a single trace id
		ItemsPerBatch:      6,
		DataItemsPerSecond: 1024,
	}
	pdp.sampledAttr = pcommon.NewMap()
	//TODO: idk if numTracesToGenerate is even needed. We can just control the load options and then start/stop
	// consider removing
	pdp.numTracesToGenerate = 5000
	pdp.randGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, opt := range opts {
		opt(&pdp)
	}
	return &pdp
}

// SetLoadGeneratorCounters sets the data providers items generated field to have a ref to the load generators counter
func (dp *samplingDataProvider) SetLoadGeneratorCounters(dataItemsGenerated *atomic.Uint64) {
	dp.dataItemsGenerated = dataItemsGenerated
}

// GenerateTraces the boolean in this return reflects whether we are done generating traces or not.
// TODO: Question how will we know we are done? Does PDP need to have a configurable amount of data to generate?
func (dp *samplingDataProvider) GenerateTraces() (ptrace.Traces, bool) {
	if dp.tracesGenerated >= dp.numTracesToGenerate {
		return ptrace.NewTraces(), true
	}
	dp.tracesGenerated += 1
	var traceData ptrace.Traces

	if dp.shouldGenerateSampledTrace() {
		traceData = dp.generateSampledTrace()
		dp.expectedSampledTraces = append(dp.expectedSampledTraces, traceData)
	} else {
		traceData = dp.generateNotSampledTrace()
	}

	return traceData, false

}

// shouldGenerateCleanTrace returns true if the next trace generated should be sampled.
func (dp *samplingDataProvider) shouldGenerateSampledTrace() bool {
	return dp.randGenerator.Float32() < 0.3
}

func (dp *samplingDataProvider) ToBeSampledSpansGenerated() uint64 {
	return dp.sampledSpansGenerated.Load()
}

func (dp *samplingDataProvider) generateSampledTrace() ptrace.Traces {
	traceData := ptrace.NewTraces()

	spans := traceData.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans()
	spans.EnsureCapacity(dp.loadOptions.ItemsPerBatch)

	traceID := dp.traceIDSequence.Add(1)
	// TODO: should we be using ItemsPerBatch to determine how many spans are in a trace?
	for i := 0; i < dp.loadOptions.ItemsPerBatch; i++ {

		startTime := time.Now()
		endTime := startTime.Add(time.Millisecond)

		spanID := dp.dataItemsGenerated.Add(1)
		dp.sampledSpansGenerated.Add(1)

		span := spans.AppendEmpty()

		// Create a span.
		span.SetTraceID(idutils.UInt64ToTraceID(0, traceID))
		span.SetSpanID(idutils.UInt64ToSpanID(spanID))
		span.SetName("dirty-trace-span")
		span.SetKind(ptrace.SpanKindClient)
		attrs := span.Attributes()

		attrs.PutInt("dirty_trace.span_seq_num", int64(spanID))
		attrs.PutInt("dirty_trace.trace_seq_num", int64(traceID))
		// Additional attributes. Defined in load options. Not sure if we should
		// keep this or not.
		for k, v := range dp.loadOptions.Attributes {
			attrs.PutStr(k, v)
		}

		// add dirty attributes
		dp.sampledAttr.Range(func(k string, v pcommon.Value) bool {
			switch v.Type() {
			case pcommon.ValueTypeInt:
				attrs.PutInt(k, v.Int())
			default:
				attrs.PutStr(k, v.Str())
			}
			return true
		})
		span.SetStartTimestamp(pcommon.NewTimestampFromTime(startTime))
		span.SetEndTimestamp(pcommon.NewTimestampFromTime(endTime))
		// used for debugging.
		// TODO: Delete if no longer required
		//attrs.Range(func(k string, v pcommon.Value) bool {
		//	log.Printf("key: %s, value: %s", k, v.Str())
		//	return true
		//})
	}
	return traceData
}

// TODO: We should establish a strict contract of what this function will generate. This way tests
// know exactly what the NotSampledTrace data will look like. This will lead to more reliable tests.
func (dp *samplingDataProvider) generateNotSampledTrace() ptrace.Traces {
	traceData := ptrace.NewTraces()
	spans := traceData.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans()
	spans.EnsureCapacity(dp.loadOptions.ItemsPerBatch)

	traceID := dp.traceIDSequence.Add(1)
	for i := 0; i < dp.loadOptions.ItemsPerBatch; i++ {

		startTime := time.Now()
		endTime := startTime.Add(time.Millisecond)

		spanID := dp.dataItemsGenerated.Add(1)

		span := spans.AppendEmpty()

		// Create a span.
		span.SetTraceID(idutils.UInt64ToTraceID(0, traceID))
		span.SetSpanID(idutils.UInt64ToSpanID(spanID))
		span.SetName("clean-trace-span")
		span.SetKind(ptrace.SpanKindClient)
		attrs := span.Attributes()
		attrs.PutInt("clean_trace.span_seq_num", int64(spanID))
		attrs.PutInt("clean_trace.trace_seq_num", int64(traceID))
		// Additional attributes. Defined in load options. Not sure if we should
		// keep this or not.
		for k, v := range dp.loadOptions.Attributes {
			attrs.PutStr(k, v)
		}
		span.SetStartTimestamp(pcommon.NewTimestampFromTime(startTime))
		span.SetEndTimestamp(pcommon.NewTimestampFromTime(endTime))
	}
	return traceData
}

// GenerateMetrics TODO
func (dp *samplingDataProvider) GenerateMetrics() (pmetric.Metrics, bool) {
	return pmetric.NewMetrics(), false
}

// GenerateLogs TODO
func (dp *samplingDataProvider) GenerateLogs() (plog.Logs, bool) {
	return plog.NewLogs(), true
}
