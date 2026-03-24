package monitoringnetwork

import (
	"fmt"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/env"
	"github.com/Networks-it-uc3m/L2S-M/internal/sdnclient"
	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
)

type ClientFactory interface {
	Internal() (sdnclient.Client, error)
	ForProvider(provider *l2smv1.ProviderSpec) (sdnclient.Client, error)
}

type DefaultClientFactory struct{}

func (DefaultClientFactory) Internal() (sdnclient.Client, error) {
	clientConfig := sdnclient.ClientConfig{
		BaseURL:  fmt.Sprintf("http://%s:%s/onos", env.GetControllerIP(), env.GetControllerPort()),
		Username: "karaf",
		Password: "karaf",
	}

	return sdnclient.NewClient(sdnclient.InternalType, clientConfig)
}

func (DefaultClientFactory) ForProvider(provider *l2smv1.ProviderSpec) (sdnclient.Client, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider is nil")
	}
	if len(provider.Domain) == 0 || provider.Domain[0] == "" {
		return nil, fmt.Errorf("provider %q has no domain configured", provider.Name)
	}

	clientConfig := sdnclient.ClientConfig{
		BaseURL:  fmt.Sprintf("http://%s:%s/onos", provider.Domain[0], utils.DefaultIfEmpty(provider.SDNPort, "30808")),
		Username: "karaf",
		Password: "karaf",
	}

	// Monitoring networks still rely on the vnets API plus port attachment.
	// Keep provider selection isolated here so the concrete client type can be
	// switched once the external provider exposes equivalent capabilities.
	return sdnclient.NewClient(sdnclient.InternalType, clientConfig)
}
