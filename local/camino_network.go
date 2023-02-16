package local

import (
	"github.com/ava-labs/avalanche-network-runner/network"
	"github.com/ava-labs/avalanchego/utils/logging"
)

func NewCustomNetwork(
	log logging.Logger,
	binaryPath string,
	reassignPortsIfUsed bool,
	postProcessConfig func(*network.Config),
) (network.Network, error) {
	config := NewDefaultConfig(binaryPath)
	postProcessConfig(&config)
	return NewNetwork(log, config, "", "", reassignPortsIfUsed)
}
