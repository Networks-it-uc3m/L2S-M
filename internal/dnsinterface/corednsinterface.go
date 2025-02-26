package dnsinterface

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Networks-it-uc3m/L2S-M/internal/env"
	"github.com/Networks-it-uc3m/l2sm-dns/pkg/configmapmanager"
)

func AddServerToLocalCoreDNS(c client.Client, networkName, remoteServerDomain, remoteServerPort string) error {
	// Create a new DNSManager using the factory.
	// Since we already have a controller-runtime client,
	// pass nil for the rest.Config parameter.
	dnsManager, err := configmapmanager.NewDNSManager(
		env.GetIntraConfigmapNamespace(),
		env.GetIntraConfigmapName(),
		nil, // no k8sConfig needed if crClient is provided
		c,
	)
	if err != nil {
		return fmt.Errorf("could not create DNS manager: %v", err)
	}

	domainName := fmt.Sprintf("%s.%s.l2sm", networkName, "inter")
	return dnsManager.AddServerToConfigMap(context.TODO(), domainName, remoteServerDomain, remoteServerPort)
}
