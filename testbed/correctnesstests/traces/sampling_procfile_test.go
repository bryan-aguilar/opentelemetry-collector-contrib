package traces

import (
	"log"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/correctnesstests"
	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/testbed"
	"github.com/stretchr/testify/require"
)

// This file just exists to test providing the processor definitions through a yaml file instead of defining them
// in a formatted string.
func TestTailSamplingDataProcessorFile(t *testing.T) {
	procFile, err := filepath.Abs("./testdata/samplingTest.yaml")
	require.NoError(t, err)
	var resourceSpec testbed.ResourceSpec
	dataSender := correctnesstests.ConstructTraceSender(t, "otlp")
	dataReceiver := correctnesstests.ConstructReceiver(t, "otlp")
	testWithSampledDataProcessorFile(t, dataSender, dataReceiver, resourceSpec, procFile)
}

func testWithSampledDataProcessorFile(
	t *testing.T,
	sender testbed.DataSender,
	receiver testbed.DataReceiver,
	resourceSpec testbed.ResourceSpec,
	processorFile string,
) {
	//TODO: The optional functions for the data providershould be provided to this generic function.
	// This function acts as a test runner of sorts
	dataProvider := testbed.NewSamplingDataProvider(testbed.WithStringAttribute("key", "value"))
	factories, err := testbed.Components()
	require.NoError(t, err, "default components resulted in: %v", err)
	runner := testbed.NewInProcessCollector(factories)
	validator := testbed.NewPollutedTestValidator(sender.ProtocolName(), receiver.ProtocolName(), dataProvider)
	config := correctnesstests.CreateConfigYamlProcFP(sender, receiver, processorFile, "groupbytrace,tail_sampling,batch", "traces")
	log.Println(config)
	configCleanup, cfgErr := runner.PrepareConfig(config)
	require.NoError(t, cfgErr, "collector configuration resulted in: %v", cfgErr)
	defer configCleanup()
	tc := testbed.NewTestCase(
		t,
		dataProvider,
		sender,
		receiver,
		runner,
		validator,
		correctnessResults,
		testbed.WithResourceLimits(resourceSpec),
	)
	defer tc.Stop()

	tc.EnableRecording()
	tc.StartBackend()
	tc.StartAgent()

	//TODO: comment this out now until I better understand where the best spot to set load options is for this test
	// scenario
	// forcing this to match what is defined in data provider. Will need to dive deeper into the correct usage around
	// testbed.LoadOptions
	// commenting out the load here forces sent data to go to 0
	tc.StartLoad(testbed.LoadOptions{
		DataItemsPerSecond: 128,
		ItemsPerBatch:      1,
	})

	tc.Sleep(5 * time.Second)

	tc.StopLoad()

	//TODO: Figure out the correct amount of wait times. 120s may be too long here.
	// This does not work because DataItemsSent will not always equal when it comes to a sampling test.
	// We also cannot access the pollutedDataProvider methods because that struct is not exported. Dilemma.
	//tc.WaitForN(func() bool { return tc.LoadGenerator.DataItemsSent() == tc.MockBackend.DataItemsReceived() },
	//	120*time.Second, "all data items received")

	tc.Sleep(15 * time.Second)
	tc.StopAgent()

	tc.ValidateData()
}
