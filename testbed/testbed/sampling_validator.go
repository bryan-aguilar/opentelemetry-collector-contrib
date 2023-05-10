package testbed

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

/**
We copied a lot of the assertions here from correctness test and only did slight modifications.  We should be seeing
how we could use reuse a lot of these functions. Maybe use a callback pattern in the correctness and polluted validator
structs.
*/

// SamplingTestValidator implements TestCaseValidator for test suites using CorrectnessResults for summarizing results.
type SamplingTestValidator struct {
	// I changed this away from the DataProvider interface. It makes this validator less generic
	// and deeply coupled with the data provider. I don't think this is necessarily a bad thing. yet.
	// TODO: Is this a bad thing?
	dataProvider         *samplingDataProvider
	assertionFailures    []*TraceAssertionFailure
	ignoreSpanLinksAttrs bool
}

func NewPollutedTestValidator(senderName string, receiverName string, provider DataProvider) *SamplingTestValidator {
	// TODO: Fix Jaeger span links attributes and tracestate.
	return &SamplingTestValidator{
		dataProvider:         provider.(*samplingDataProvider),
		ignoreSpanLinksAttrs: senderName == "jaeger" || receiverName == "jaeger",
	}
}

func (v *SamplingTestValidator) Validate(tc *TestCase) {
	if assert.EqualValues(tc.t,
		int64(v.dataProvider.ToBeSampledSpansGenerated()),
		int64(tc.MockBackend.DataItemsReceived()),
		"Received and sent counters do not match.") {
		log.Printf("Sent and received data counters match.")
	}
	if len(tc.MockBackend.ReceivedTraces) > 0 {
		v.assertSentRecdTracingDataEqual(tc.MockBackend.ReceivedTraces)
	}
	assert.EqualValues(tc.t, 0, len(v.assertionFailures), "There are span data mismatches.")
}

func (v *SamplingTestValidator) RecordResults(tc *TestCase) {
	var result string
	if tc.t.Failed() {
		result = "FAIL"
	} else {
		result = "PASS"
	}

	// Remove "Test" prefix from test name.
	testName := tc.t.Name()[4:]
	tc.resultsSummary.Add(tc.t.Name(), &CorrectnessTestResult{
		testName:                   testName,
		result:                     result,
		duration:                   time.Since(tc.startTime),
		receivedSpanCount:          tc.MockBackend.DataItemsReceived(),
		sentSpanCount:              v.dataProvider.ToBeSampledSpansGenerated(),
		traceAssertionFailureCount: uint64(len(v.assertionFailures)),
		traceAssertionFailures:     v.assertionFailures,
	})
}

func (v *SamplingTestValidator) assertSentRecdTracingDataEqual(tracesList []ptrace.Traces) {
	spansMap := make(map[string]ptrace.Span)
	populateSpansMap(spansMap, v.dataProvider.expectedSampledTraces)

	for _, td := range tracesList {
		rss := td.ResourceSpans()
		for i := 0; i < rss.Len(); i++ {
			ilss := rss.At(i).ScopeSpans()
			for j := 0; j < ilss.Len(); j++ {
				spans := ilss.At(j).Spans()
				for k := 0; k < spans.Len(); k++ {
					recdSpan := spans.At(k)
					if sentSpan, ok := spansMap[traceIDAndSpanIDToString(recdSpan.TraceID(), recdSpan.SpanID())]; ok {
						v.diffSpan(sentSpan, recdSpan)
					}

				}
			}
		}
	}
}

func (v *SamplingTestValidator) diffSpan(sentSpan ptrace.Span, recdSpan ptrace.Span) {
	v.diffSpanTraceID(sentSpan, recdSpan)
	v.diffSpanSpanID(sentSpan, recdSpan)
	v.diffSpanTraceState(sentSpan, recdSpan)
	v.diffSpanParentSpanID(sentSpan, recdSpan)
	v.diffSpanName(sentSpan, recdSpan)
	v.diffSpanKind(sentSpan, recdSpan)
	v.diffSpanTimestamps(sentSpan, recdSpan)
	v.diffSpanAttributes(sentSpan, recdSpan)
	v.diffSpanEvents(sentSpan, recdSpan)
	v.diffSpanLinks(sentSpan, recdSpan)
	v.diffSpanStatus(sentSpan, recdSpan)
}

func (v *SamplingTestValidator) diffSpanTraceID(sentSpan ptrace.Span, recdSpan ptrace.Span) {
	if sentSpan.TraceID() != recdSpan.TraceID() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "TraceId",
			expectedValue: sentSpan.TraceID(),
			actualValue:   recdSpan.TraceID(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *SamplingTestValidator) diffSpanSpanID(sentSpan ptrace.Span, recdSpan ptrace.Span) {
	if sentSpan.SpanID() != recdSpan.SpanID() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "SpanId",
			expectedValue: sentSpan.SpanID(),
			actualValue:   recdSpan.SpanID(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *SamplingTestValidator) diffSpanTraceState(sentSpan ptrace.Span, recdSpan ptrace.Span) {
	if sentSpan.TraceState().AsRaw() != recdSpan.TraceState().AsRaw() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "TraceState",
			expectedValue: sentSpan.TraceState().AsRaw(),
			actualValue:   recdSpan.TraceState().AsRaw(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *SamplingTestValidator) diffSpanParentSpanID(sentSpan ptrace.Span, recdSpan ptrace.Span) {
	if sentSpan.ParentSpanID() != recdSpan.ParentSpanID() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "ParentSpanId",
			expectedValue: sentSpan.ParentSpanID(),
			actualValue:   recdSpan.ParentSpanID(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *SamplingTestValidator) diffSpanName(sentSpan ptrace.Span, recdSpan ptrace.Span) {
	// Because of https://github.com/openzipkin/zipkin-go/pull/166 compare lower cases.
	if !strings.EqualFold(sentSpan.Name(), recdSpan.Name()) {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "Name",
			expectedValue: sentSpan.Name(),
			actualValue:   recdSpan.Name,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *SamplingTestValidator) diffSpanKind(sentSpan ptrace.Span, recdSpan ptrace.Span) {
	if sentSpan.Kind() != recdSpan.Kind() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "Kind",
			expectedValue: sentSpan.Kind,
			actualValue:   recdSpan.Kind,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *SamplingTestValidator) diffSpanTimestamps(sentSpan ptrace.Span, recdSpan ptrace.Span) {
	if notWithinOneMillisecond(sentSpan.StartTimestamp(), recdSpan.StartTimestamp()) {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "StartTimestamp",
			expectedValue: sentSpan.StartTimestamp(),
			actualValue:   recdSpan.StartTimestamp(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if notWithinOneMillisecond(sentSpan.EndTimestamp(), recdSpan.EndTimestamp()) {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "EndTimestamp",
			expectedValue: sentSpan.EndTimestamp(),
			actualValue:   recdSpan.EndTimestamp(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *SamplingTestValidator) diffSpanAttributes(sentSpan ptrace.Span, recdSpan ptrace.Span) {
	if sentSpan.Attributes().Len() != recdSpan.Attributes().Len() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "Attributes",
			expectedValue: sentSpan.Attributes().Len(),
			actualValue:   recdSpan.Attributes().Len(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	} else {
		v.diffAttributeMap(sentSpan.Name(), sentSpan.Attributes(), recdSpan.Attributes(), "Attributes[%s]")
	}
	if sentSpan.DroppedAttributesCount() != recdSpan.DroppedAttributesCount() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "DroppedAttributesCount",
			expectedValue: sentSpan.DroppedAttributesCount(),
			actualValue:   recdSpan.DroppedAttributesCount(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *SamplingTestValidator) diffSpanEvents(sentSpan ptrace.Span, recdSpan ptrace.Span) {
	if sentSpan.Events().Len() != recdSpan.Events().Len() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "Events",
			expectedValue: sentSpan.Events().Len(),
			actualValue:   recdSpan.Events().Len(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	} else {
		sentEventMap := convertEventsSliceToMap(sentSpan.Events())
		recdEventMap := convertEventsSliceToMap(recdSpan.Events())
		for name, sentEvents := range sentEventMap {
			recdEvents, match := recdEventMap[name]
			if match {
				match = len(sentEvents) == len(recdEvents)
			}
			if !match {
				af := &TraceAssertionFailure{
					typeName:      "Span",
					dataComboName: sentSpan.Name(),
					fieldPath:     fmt.Sprintf("Events[%s]", name),
					expectedValue: len(sentEvents),
					actualValue:   len(recdEvents),
				}
				v.assertionFailures = append(v.assertionFailures, af)
			} else {
				for i, sentEvent := range sentEvents {
					recdEvent := recdEvents[i]
					if notWithinOneMillisecond(sentEvent.Timestamp(), recdEvent.Timestamp()) {
						af := &TraceAssertionFailure{
							typeName:      "Span",
							dataComboName: sentSpan.Name(),
							fieldPath:     fmt.Sprintf("Events[%s].TimeUnixNano", name),
							expectedValue: sentEvent.Timestamp(),
							actualValue:   recdEvent.Timestamp(),
						}
						v.assertionFailures = append(v.assertionFailures, af)
					}
					v.diffAttributeMap(sentSpan.Name(), sentEvent.Attributes(), recdEvent.Attributes(),
						"Events["+name+"].Attributes[%s]")
				}
			}
		}
	}
	if sentSpan.DroppedEventsCount() != recdSpan.DroppedEventsCount() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "DroppedEventsCount",
			expectedValue: sentSpan.DroppedEventsCount(),
			actualValue:   recdSpan.DroppedEventsCount(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *SamplingTestValidator) diffSpanLinks(sentSpan ptrace.Span, recdSpan ptrace.Span) {
	if sentSpan.Links().Len() != recdSpan.Links().Len() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "Links",
			expectedValue: sentSpan.Links().Len(),
			actualValue:   recdSpan.Links().Len(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	} else {
		recdLinksMap := convertLinksSliceToMap(recdSpan.Links())
		sentSpanLinks := sentSpan.Links()
		for i := 0; i < sentSpanLinks.Len(); i++ {
			sentLink := sentSpanLinks.At(i)
			recdLink, ok := recdLinksMap[traceIDAndSpanIDToString(sentLink.TraceID(), sentLink.SpanID())]
			if ok {
				if v.ignoreSpanLinksAttrs {
					return
				}
				if sentLink.TraceState().AsRaw() != recdLink.TraceState().AsRaw() {
					af := &TraceAssertionFailure{
						typeName:      "Span",
						dataComboName: sentSpan.Name(),
						fieldPath:     "Link.TraceState",
						expectedValue: sentLink,
						actualValue:   recdLink,
					}
					v.assertionFailures = append(v.assertionFailures, af)
				}
				v.diffAttributeMap(sentSpan.Name(), sentLink.Attributes(), recdLink.Attributes(),
					"Link.Attributes[%s]")
			} else {
				af := &TraceAssertionFailure{
					typeName:      "Span",
					dataComboName: sentSpan.Name(),
					fieldPath:     fmt.Sprintf("Links[%d]", i),
					expectedValue: sentLink,
					actualValue:   nil,
				}
				v.assertionFailures = append(v.assertionFailures, af)
			}

		}
	}
	if sentSpan.DroppedLinksCount() != recdSpan.DroppedLinksCount() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "DroppedLinksCount",
			expectedValue: sentSpan.DroppedLinksCount(),
			actualValue:   recdSpan.DroppedLinksCount(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *SamplingTestValidator) diffSpanStatus(sentSpan ptrace.Span, recdSpan ptrace.Span) {
	if sentSpan.Status().Code() != recdSpan.Status().Code() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "Status.Code",
			expectedValue: sentSpan.Status().Code(),
			actualValue:   recdSpan.Status().Code(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *SamplingTestValidator) diffAttributeMap(spanName string,
	sentAttrs pcommon.Map, recdAttrs pcommon.Map, fmtStr string) {
	sentAttrs.Range(func(sentKey string, sentVal pcommon.Value) bool {
		recdVal, ok := recdAttrs.Get(sentKey)
		if !ok {
			af := &TraceAssertionFailure{
				typeName:      "Span",
				dataComboName: spanName,
				fieldPath:     fmt.Sprintf(fmtStr, sentKey),
				expectedValue: sentVal,
				actualValue:   nil,
			}
			v.assertionFailures = append(v.assertionFailures, af)
			return true
		}
		switch sentVal.Type() {
		case pcommon.ValueTypeMap:
			v.compareKeyValueList(spanName, sentVal, recdVal, fmtStr, sentKey)
		default:
			v.compareSimpleValues(spanName, sentVal, recdVal, fmtStr, sentKey)
		}
		return true
	})
}

func (v *SamplingTestValidator) compareSimpleValues(spanName string, sentVal pcommon.Value, recdVal pcommon.Value,
	fmtStr string, attrKey string) {
	if reflect.DeepEqual(sentVal.AsRaw(), recdVal.AsRaw()) {
		sentStr := sentVal.AsString()
		recdStr := recdVal.AsString()
		if !strings.EqualFold(sentStr, recdStr) {
			af := &TraceAssertionFailure{
				typeName:      "Span",
				dataComboName: spanName,
				fieldPath:     fmt.Sprintf(fmtStr, attrKey),
				expectedValue: sentVal.AsString(),
				actualValue:   recdVal.AsString(),
			}
			v.assertionFailures = append(v.assertionFailures, af)
		}
	}
}

func (v *SamplingTestValidator) compareKeyValueList(
	spanName string, sentVal pcommon.Value, recdVal pcommon.Value, fmtStr string, attrKey string) {
	switch recdVal.Type() {
	case pcommon.ValueTypeMap:
		v.diffAttributeMap(spanName, sentVal.Map(), recdVal.Map(), fmtStr)
	case pcommon.ValueTypeStr:
		v.compareSimpleValues(spanName, sentVal, recdVal, fmtStr, attrKey)
	default:
		af := &TraceAssertionFailure{
			typeName:      "SimpleAttribute",
			dataComboName: spanName,
			fieldPath:     fmt.Sprintf(fmtStr, attrKey),
			expectedValue: sentVal,
			actualValue:   recdVal,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}
