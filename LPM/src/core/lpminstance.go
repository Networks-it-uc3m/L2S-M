package lpmcore

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type MeasureMethod func(neighborNodeIp string) float64

type ServerMethod func()

type Metric struct {
	Name             string
	NodeName         string
	Ip               string
	TestTimeInterval int
	MetricId         string
	method           MeasureMethod
	value            float64
}

type LPMInstance struct {
	NodeName string
	promReg  *prometheus.Registry
	Metrics  []Metric
	Servers  []ServerMethod
}

var lock = &sync.Mutex{}

var lpmInstance *LPMInstance

func GetInstance() *LPMInstance {
	if lpmInstance == nil {
		lock.Lock()
		defer lock.Unlock()
		if lpmInstance == nil {
			fmt.Println("Creating single instance now.")
			lpmInstance = &LPMInstance{}
		} else {
			fmt.Println("Single instance already created.")
		}
	} else {
		fmt.Println("Single instance already created.")
	}

	lpmInstance.promReg = prometheus.NewRegistry()
	return lpmInstance
}

func (lpmInstance *LPMInstance) SetNodeName(name string) {
	lpmInstance.NodeName = name
}

func (lpmInstance *LPMInstance) AddMetric(metricName string, nodeName string, metricInterval int, neighborNodeIp string, measureMethod MeasureMethod) {
	stringId := fmt.Sprintf("%s_%s:%s", metricName, lpmInstance.NodeName, nodeName)
	lpmInstance.Metrics = append(lpmInstance.Metrics, Metric{metricName, nodeName, neighborNodeIp, metricInterval, stringId, measureMethod, 0.0})
	lpmCollector := lpmExporterCollector(stringId)
	lpmInstance.promReg.MustRegister(lpmCollector)
}

func (lpmInstance *LPMInstance) AddServer(serverMethod ServerMethod) {
	lpmInstance.Servers = append(lpmInstance.Servers, serverMethod)
}

func GetMetricValueFromId(metricId string) float64 {
	for _, metric := range lpmInstance.Metrics {
		if metric.MetricId == metricId {
			return metric.value
		}
	}
	return 0.0
}
