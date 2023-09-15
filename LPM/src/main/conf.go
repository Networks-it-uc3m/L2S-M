package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type MetricConfiguration struct {
	Name       string `json:"name"`
	IP         string `json:"ip"`
	RTT        int    `json:"rttInterval,omitempty"`
	Jitter     int    `json:"jitterInterval,omitempty"`
	Throughput int    `json:"throughputInterval,omitempty"`
	OTD        int    `json:"otdInterval,omitempty"`
}

type NodeConfig struct {
	NodeName       string                `json:"Nodename"`
	NeighbourNodes []MetricConfiguration `json:"NeighbourNodes"`
}

func loadConfiguration() (NodeConfig, error) {

	file, err := os.Open("config.json")

	if err != nil {
		return NodeConfig{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	configuration := NodeConfig{}
	err = decoder.Decode(&configuration)

	fmt.Println("Decoded Config Data:")
	fmt.Printf("%v+", configuration)
	// for _, cfg := range configuration {
	// 	//fmt.Printf("IP: %s, RTT: %d, Jitter: %d, Throughput: %d\n", cfg.IP, cfg.RTT, cfg.Jitter, cfg.Throughput)
	// }
	return configuration, nil
}

func (conf *MetricConfiguration) UnmarshalJSON(data []byte) error {
	type confAlias MetricConfiguration
	defaultConf := &confAlias{RTT: -1, Jitter: -1, Throughput: -1}

	err := json.Unmarshal(data, defaultConf)
	if err != nil {
		return err
	}

	*conf = MetricConfiguration(*defaultConf)
	return nil
}
