package prometheusclient

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Networks-it-uc3m/LPM/pkg/collector"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type PrometheusClient struct {
	ApiClient v1.API
}

func NewClient(address string) *PrometheusClient {

	client, err := api.NewClient(api.Config{
		Address: fmt.Sprintf("http://%s", address),
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}

	apiClient := v1.NewAPI(client)
	return &PrometheusClient{ApiClient: apiClient}
}

func (promClient *PrometheusClient) GetNetworkMetrics() []collector.MetricData {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Construct a query to fetch metrics starting with "net_"
	query := `{__name__=~"net_.*"}`

	result, warnings, err := promClient.ApiClient.Query(ctx, query, time.Now())
	if err != nil {
		fmt.Printf("Error querying Prometheus: %v\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		for _, warning := range warnings {
			fmt.Printf("Warning: %s\n", warning)
		}
	}

	var metrics []collector.MetricData

	// Handle the type of the result and construct MetricData array
	switch v := result.(type) {
	case model.Vector:
		for _, item := range v {
			metricName := strings.Join(strings.Split(string(item.Metric["__name__"]), "_")[0:3], "_")
			identifier := strings.Split(string(item.Metric["__name__"]), "_")[3]
			sourceNode := string(item.Metric["source_node"])
			targetNode := string(item.Metric["target_node"])
			item, _ := strconv.ParseFloat(item.Value.String(), 32)
			metric := collector.MetricData{
				MetricId: collector.MetricId{
					Name:           metricName,
					SourceNodeName: sourceNode,
					TargetNodeName: targetNode,
					Identifier:     identifier,
				},
				Value: item,
			}

			metrics = append(metrics, metric)
		}
	default:
		fmt.Println("Unknown format of the result")
	}

	return metrics
}
