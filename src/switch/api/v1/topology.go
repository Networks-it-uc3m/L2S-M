package v1

type Node struct {
	Name          string   `json:"name"`
	NodeIP        string   `json:"nodeIP"`
	NeighborNodes []string `json:"neighborNodes,omitempty"`
}

type Link struct {
	EndpointNodeA string `json:"endpointA"`
	EndpointNodeB string `json:"endpointB"`
}

type Topology struct {
	Nodes []Node `json:"Nodes"`
	Links []Link `json:"Links"`
}
