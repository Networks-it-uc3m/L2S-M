package main

import (

	// Core of the application has been isolated from this code so its easier to update and mantain. It shouldn't be required to understand it if you want to
	// add more metrics, but if youre curious about the implementation, feel free to read the code :)
	"github.com/Tjaarda1/lpmcore"
)

func main() {

	// Load configuration from config.json file
	configuration, _ := loadConfiguration()

	// Load core instance of the lpm app, that has the core utilites of running the metric tests, launching the according prometheus collectors and registries
	lpmInstance := lpmcore.GetInstance()

	// We set the instance node name. This is useful for the correct identification of the metrics.
	lpmInstance.SetNodeName(configuration.NodeName)

	// For every neighbour node defined in the configuration file, we add a metric. Note: If the metric wasn't added, the interval will be set to -1, and the lpmInstance won't run the test.
	for _, neighbourNode := range configuration.NeighbourNodes {

		// About the AddMetric method:
		// The first parameter is the name of the metric, it should be unique between different metrics as it will help us identify what was measured.
		// The second parameter is the name of the node we want to measure the metrics from the parent node where the instance is deployed.
		// The third parameter the interval that will be taken between measurements, in minutes. So if neighborNode.rttInterval = 10, every 10 minutes the RTT
		// measurement method will be called (in this case measureRtt)
		// The foruth parameter is the IP of the neighbor node, that will be used as an argument for the measurement method. Should be as a string.
		// The fifth parameter is the function itself. should be of with the following layout 'func measure(neighborIP string) float64' You have to implement it, and I recommend doing so
		// in the metricmethods.go section, in order to keep the code clean
		lpmInstance.AddMetric("net_rtt", neighbourNode.Name, neighbourNode.RTT, neighbourNode.IP, measureRtt)
		lpmInstance.AddMetric("net_jitter", neighbourNode.Name, neighbourNode.Jitter, neighbourNode.IP, measureJitter)
		lpmInstance.AddMetric("net_throughput", neighbourNode.Name, neighbourNode.Throughput, neighbourNode.IP, measureThroughput)

	}

	lpmInstance.AddServer(iperfTCP)
	lpmInstance.AddServer(iperfUDP)

	// We have the instance correctly initiated, we can now run the collector. The collector will:
	// 1 Run the specified measurements with the addmetric, in the intervals specified
	// 2 Serve over http in localhost:8090/metrics the saved metrics, so prometheus can call the endpoint and get our custom measurements.
	lpmcore.StartCollector()

}
