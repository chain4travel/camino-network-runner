package network

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	coreth_params "github.com/ava-labs/coreth/params"
)

// PrivateKey-12bQFG6mSUVLsq2H1EAxGbu8p6mYi7zEA7QvQzJezL12JS8j5 -> X-kopernikus1zy075lddftstzpwzvt627mvc0tep0vyk7y9v4l
// PrivateKey-BhnbhFKyDjhW8r3v9ZY6wPWkrAphTVPzniLjLrviZV8ndHMBe -> X-kopernikus1lx58kettrnt2kyr38adyrrmxt5x57u4vg4cfky
// PrivateKey-Ge71NJhUY3TjZ9dLohijSnNq46QxobjqxHGMUDAPoVsNFA93w -> X-kopernikus13kyf72ftu4l77kss7xm0kshm0au29s48zjaygq
//go:embed default/genesis.json
var genesisBytes []byte

// LoadLocalGenesis loads the local network genesis from disk
// and returns it as a map[string]interface{}
func LoadLocalGenesis() (map[string]interface{}, error) {
	var (
		genesisMap map[string]interface{}
		err        error
	)
	if err = json.Unmarshal(genesisBytes, &genesisMap); err != nil {
		return nil, err
	}

	cChainGenesis := genesisMap["cChainGenesis"]
	// set the cchain genesis directly from coreth
	// the whole of `cChainGenesis` should be set as a string, not a json object...
	corethCChainGenesis := coreth_params.AvalancheLocalChainConfig
	if _, ok := os.LookupEnv("CAMINO_NETWORK"); ok {
		corethCChainGenesis.SunrisePhase0BlockTimestamp = big.NewInt(0)
	}

	// but the part in coreth is only the "config" part.
	// In order to set it easily, first we get the cChainGenesis item
	// convert it to a map
	cChainGenesisMap, ok := cChainGenesis.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(
			"expected field 'cChainGenesis' of genesisMap to be a map[string]interface{}, but it failed with type %T", cChainGenesis)
	}
	// set the `config` key to the actual coreth object
	cChainGenesisMap["config"] = corethCChainGenesis
	// and then marshal everything into a string
	configBytes, err := json.Marshal(cChainGenesisMap)
	if err != nil {
		return nil, err
	}
	// this way the whole of `cChainGenesis` is a properly escaped string
	genesisMap["cChainGenesis"] = string(configBytes)
	return genesisMap, nil
}
