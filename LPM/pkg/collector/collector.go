package collector

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func StartCollector() {

	for index := range lpmInstance.Metrics {
		go func(lpmDataIndex int) {
			lpmInstance.Metrics[lpmDataIndex].RunPeriodicTests()
		}(index)
	}

	for index := range lpmInstance.Servers {
		go func(lpmDataIndex int) {
			lpmInstance.Servers[lpmDataIndex]()
		}(index)
	}
	handler := promhttp.HandlerFor(
		lpmInstance.promReg,
		promhttp.HandlerOpts{
			EnableOpenMetrics: false,
		})

	http.Handle("/metrics", handler)

	http.ListenAndServe(":8090", nil)
}

func (metric *Metric) RunPeriodicTests() {

	log.Infof("Testing %s in network link between node %s and node %s", metric.Name, metric.SourceNodeName, metric.TargetNodeName)

	if metric.TestTimeInterval != -1 {
		for {
			for i := 0; i < 3; i++ {
				randomDelay := time.Duration(rand.Intn(120)) * time.Second
				time.Sleep(randomDelay)

				metric.Value = metric.method(metric.TargetNodeIp)

				if metric.Value != 0 {
					break
				}
				log.Infof("Couldn't measure %s between node %s and node %s. Trying again.", metric.Name, metric.SourceNodeName, metric.TargetNodeName, metric.Value)
			}

			log.Infof(" %s between node %s and node %s is %f.", metric.Name, metric.SourceNodeName, metric.TargetNodeName, metric.Value)

			time.Sleep(time.Duration(metric.TestTimeInterval) * time.Minute)
		}
	}

}
