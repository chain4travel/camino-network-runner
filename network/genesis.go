package network

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/ava-labs/avalanchego/utils/constants"

	coreth_params "github.com/ava-labs/coreth/params"
)

var Kopernikus = func() bool {
	return ID == strconv.FormatUint(uint64(constants.KopernikusID), 10)
}

//go:embed default/genesis.json
var genesisBytes []byte

//go:embed default/genesis_kopernikus.json
var genesisKopernikusBytes []byte

var ID = os.ExpandEnv("$NETWORK_ID")

// LoadLocalGenesis loads the local network genesis from disk
// and returns it as a map[string]interface{}
func LoadLocalGenesis() (map[string]interface{}, error) {
	var (
		genesisMap map[string]interface{}
		err        error
	)
	genesis := genesisBytes
	if Kopernikus() {
		genesis = genesisKopernikusBytes
	}
	if err = json.Unmarshal(genesis, &genesisMap); err != nil {
		return nil, err
	}

	cChainGenesis := genesisMap["cChainGenesis"]
	// set the cchain genesis directly from coreth
	// the whole of `cChainGenesis` should be set as a string, not a json object...
	corethCChainGenesis := coreth_params.AvalancheLocalChainConfig
	// but the part in coreth is only the "config" part.
	// In order to set it easily, first we get the cChainGenesis item
	// convert it to a map
	cChainConfigStr, ok := cChainGenesis.(string)
	if !ok {
		return nil, fmt.Errorf(
			"expected field 'cChainGenesis' of genesisMap to be a string, but it failed with type %T", cChainGenesis)
	}
	cChainConfigBytes := []byte(cChainConfigStr)
	err = json.Unmarshal(cChainConfigBytes, &cChainConfig)
	if err != nil {
		panic(err)
	}
	// set the `config` key to the actual coreth object
	cChainConfig["config"] = corethCChainGenesis
	// and then marshal everything into a string
	configBytes, err := json.Marshal(cChainConfig)
	if err != nil {
		return nil, err
	}
	// this way the whole of `cChainGenesis` is a properly escaped string
	genesisMap["cChainGenesis"] = string(configBytes)
	return genesisMap, nil
}
