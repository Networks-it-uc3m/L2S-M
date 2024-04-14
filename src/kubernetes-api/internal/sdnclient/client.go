package sdnclient

import (
	"errors"

	l2smv1 "l2sm.k8s.local/l2sm-kapi/api/v1"
)

type ClientType string

const (
	InternalType ClientType = "internal"
	ExternalType ClientType = "external"
)

// NetworkStrategy defines the interface for network strategies
type Client interface {
	CreateNetwork(networkType l2smv1.NetworkType, config interface{}) error
	DeleteNetwork(networkType l2smv1.NetworkType, networkID string) error
	CheckNetworkExists(networkType l2smv1.NetworkType, networkID string) (bool, error)
}

type ClientConfig struct {
	BaseURL  string
	Username string
	Password string
}

func NewClient(clientType ClientType, config ClientConfig) (Client, error) {
	sessionClient := NewSessionClient(config.BaseURL, config.Username, config.Password)

	switch clientType {
	case InternalType:
		client := &InternalClient{Session: sessionClient}
		if !client.beginSessionController() {
			return nil, errors.New("could not initialize session with SDN controller. Please check the connection details and credentials.")
		}
		return client, nil
	case ExternalType:
		client := &ExternalClient{Session: sessionClient} // Adjust ExternalClient struct accordingly
		if !client.beginSessionController() {
			return nil, errors.New("could not initialize session with SDN controller. Please check the connection details and credentials.")
		}
		return client, nil
	default:
		return nil, errors.New("unsupported client type")
	}
}
