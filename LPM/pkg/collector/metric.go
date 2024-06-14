package collector

import (
	"encoding/base64"
	"fmt"
	"hash/fnv"

	"strings"
)

type MeasureMethod func(targetNodeIP string) float64

type MetricId struct {
	Name           string
	SourceNodeName string
	TargetNodeName string
	Identifier     string
}
type MetricData struct {
	MetricId
	Value float64
}
type Metric struct {
	MetricData
	TargetNodeIp     string
	TestTimeInterval int
	MetricId         string
	method           MeasureMethod
}

func (metricId *MetricId) GenerateMetricId() error {
	hashString := fmt.Sprintf("%s%s", metricId.SourceNodeName, metricId.TargetNodeName)
	hash := fnv.New32() // Using FNV hash for a compact hash, but still 32 bits
	hash.Write([]byte(hashString))
	sum := hash.Sum32()

	// Encode the hash as a base32 string and take the first 4 characters
	encoded := fmt.Sprintf("%04x", sum) // Hexadecimal encoding; consider using a different encoding for better character use
	metricId.Identifier = fmt.Sprintf("%s_%s", metricId.Name, encoded)
	// return base64.StdEncoding.EncodeToString([]byte(metricId))
	return nil
}

// DecomposeMetricId takes a metric ID and returns the original metric name, source node, and target node.
func DecomposeMetricId(encodedId string) (string, string, string) {
	decodedId, err := base64.StdEncoding.DecodeString(encodedId)
	if err != nil {
		return "", "", ""
	}

	// Split the string at the first underscore to isolate the metric name.
	parts := strings.SplitN(string(decodedId), "_", 2)
	if len(parts) < 2 {
		return "", "", "" // Return empty strings if the format is incorrect.
	}
	metricName := parts[0]

	// Split the second part at the colon to separate the source and target nodes.
	nodes := strings.Split(parts[1], ":")
	if len(nodes) < 2 {
		return metricName, "", "" // Return the metric name and empty strings for nodes if format is incorrect.
	}

	sourceNode, targetNode := nodes[0], nodes[1]
	return metricName, sourceNode, targetNode
}
