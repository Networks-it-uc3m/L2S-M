package talpainterface

import (
	talpav1 "github.com/Networks-it-uc3m/l2sm-switch/api/v1"
	dp "github.com/Networks-it-uc3m/l2sm-switch/pkg/datapath"
)

func GetIfName(nodeName, providerName string, index int) string {

	ifid := dp.New(dp.GetSwitchName(dp.DatapathParams{NodeName: nodeName, ProviderName: providerName}))
	return ifid.Port(index)
}
func ProbeInterface(node, provider string) string {

	ifid := dp.New(dp.GetSwitchName(dp.DatapathParams{NodeName: node, ProviderName: provider}))
	return ifid.Probe(talpav1.RESERVED_PROBE_ID)
}
