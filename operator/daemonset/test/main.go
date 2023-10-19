package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type Node struct {
	Name          string   `json:"name"`
	NodeIP        string   `json:"nodeIP"`
	NeighborNodes []string `json:"neighborNodes"`
}

func main() {
	data, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		fmt.Println("Error reading input file:", err)
		return
	}

	var nodes []Node
	err = json.Unmarshal(data, &nodes)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	nodeVniRef := 5
	for _, node := range nodes {
		if node.Name == os.Args[1] {
			nodeIP := strings.TrimSpace(node.NodeIP)
			neighborVniRef := 5
			for _, neighbor := range node.NeighborNodes {
				for _, n := range nodes {
					if n.Name == neighbor {
						vni := fmt.Sprintf("%d00%d", nodeVniRef, neighborVniRef)
						neighborIP := strings.TrimSpace(n.NodeIP)
						command := fmt.Sprintf("ovs-vsctl add-port brtun vxlan1 -- set interface vxlan1 type=vxlan options:key=%s options:remote_ip=%s options:local_ip=%s options:dst_port=7000", vni, neighborIP, nodeIP)
						fmt.Println(command)
					}
					neighborVniRef++
				}
			}
		}
		nodeVniRef++
	}
}
