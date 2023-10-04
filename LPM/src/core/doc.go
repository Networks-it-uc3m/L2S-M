// Package lpmcore is the core instrumentation package. Modifications shouldnt be needed.
// The purpose of LPM is to easily integrate a propagation measurement app with the prometheus project. LPMCore doesn't provide any measurment methods,
// but it provides an abstraction layer where adding own custom network metrics is easier than ever.
//
// It provides a singleton instance of LPMInstance, which itself has the metric data taken from the measurements,
// the information of the neighbor nodes and the prometheus registry. LPMCore takes care of creating the necesary prometheus
// collectors, running the measurements in the specified intervals between the specified nodes and serving them over http in localhost:8090/metrics.
//
// # Using LPMCore
//
// Three main things should be implemented when using LPMCore. Here we provide an example of usage:
//
// ## Implementing measurement methods
// As LPMCore doesnt provide any measurement method, an implementation must be added for every metric that you want to take.
// The measurement methods should have the following syntax:
// 'func measureMetric(neighborIP string) float64'
// Where measureMetric is the unique name of the method you are implementing (for example measureRtt or measureJitter)
// As argument it should provide only one, as a string, that it should be used for specifying the ip we want to run the test against,
// as LPMCore will later call this method by using that field. The return should be a float64 always, which contains the result of the final test. Errors should be managed inside
// the method. If an error occur, an invalid value should be used. For example, if while measuring rtt you get an error, you may return a negative value so when
// the metric is logged, the value can be interpreted as such.
// Here a basic example of a measuring method:
//
// -------------------------------------------------------
// func measureRtt(neighborIP string) float64 {
//
// log.Infof("Measuring rtt")
//
// out, err := exec.Command("ping", neighborIP, "-c", "10", "-q").Output()
// if err != nil {
// 	log.Errorf("Could not measure Rtt. Ping responds: %v", err)
// 	return -1
// }
//
// // Regular expression pattern
// pattern := `rtt min/avg/max/mdev = [0-9.]+/([0-9.]+)/`
//
// // Compile the regular expression
// re := regexp.MustCompile(pattern)
//
// // Find the first match in the input string
// match := re.FindStringSubmatch(string(out[:]))
//
// if len(match) < 2 {
// 	log.Errorf("Could not measure Rtt. Check the connection between the two nodes.")
// 	return -1
// }
// // Print the result
// log.Infof("Rtt between two links: %s", match[1])
//
// rtt, _ := strconv.ParseFloat(match[1], 64)
// return rtt
// }
// ------------------------------------------------------------------
//
//
// # HTTP Exposition
//
// The Registry implements the Gatherer interface. The caller of the Gather
// method can then expose the gathered metrics in some way. Usually, the metrics
// are served via HTTP on the /metrics endpoint. That's happening in the example
// above. The tools to expose metrics via HTTP are in the promhttp sub-package.
//
// # Code structure:
//
// The main singleton can be found in the lpminstance.go file,
// along its initiation methods.
//
// In lpmcore.go the methods that run the instance are defined.
//
// In collector.go you can find the implementation of the methods
// that define the prometheus collectors saved in the registry
//
// All exported functions and methods are safe to be used concurrently unless
// specified otherwise.
package lpmcore
