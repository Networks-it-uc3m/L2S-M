package collector

import (
	"github.com/prometheus/client_golang/prometheus"
)

//Define a struct for you collector that contains pointers
//to prometheus descriptors for each metric you wish to expose.
//Note you can also include fields of other types if they provide utility
//but we just won't be exposing them as metrics.
type lpmCollector struct {
	metric           *prometheus.Desc
	metricName       string
	sourceNode       string
	targetNode       string
	metricIdentifier string
}

//You must create a constructor for you collector that
//initializes every descriptor and returns a pointer to the collector
func lpmExporterCollector(metricId MetricId) *lpmCollector {
	return &lpmCollector{
		metric: prometheus.NewDesc(metricId.Identifier,
			"",
			[]string{"source_node", "target_node"}, nil,
		),

		metricName:       metricId.Name,
		sourceNode:       metricId.SourceNodeName,
		targetNode:       metricId.TargetNodeName,
		metricIdentifier: metricId.Identifier,
	}
}

//Each and every collector must implement the Describe function.
//It essentially writes all descriptors to the prometheus desc channel.
func (collector *lpmCollector) Describe(ch chan<- *prometheus.Desc) {

	//Update this section with the each metric you create for a given collector
	ch <- collector.metric
}

//Collect implements required collect function for all promehteus collectors
func (collector *lpmCollector) Collect(ch chan<- prometheus.Metric) {

	//Implement logic here to determine proper metric value to return to prometheus
	//for each descriptor or call other functions that do so.

	metricValue := GetMetricValue(collector.metricName, collector.sourceNode, collector.targetNode)
	//Write latest value for each metric in the prometheus metric channel.
	//Note that you can pass CounterValue, GaugeValue, or UntypedValue types here.

	ch <- prometheus.MustNewConstMetric(collector.metric, prometheus.CounterValue, metricValue, collector.sourceNode, collector.targetNode)

}
