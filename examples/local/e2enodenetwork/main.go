package main

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ava-labs/avalanche-network-runner/network/node"
	"github.com/ava-labs/avalanchego/config"
	"io/fs"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ava-labs/avalanche-network-runner/local"
	"github.com/ava-labs/avalanche-network-runner/network"
	"github.com/ava-labs/avalanchego/utils/logging"
	"go.uber.org/zap"
)

const (
	healthyTimeout = 2 * time.Minute
)

// node6: PrivateKey-UfV3iPVP8ThZuSXmUacsahdzePs5VkXct4XoQKsW9mffN1d8J -> X-kopernikus1nnptptd6l2a4ty69jgv9ng70va72lyx2xq7ddx
// node7: PrivateKey-2DXzE36hZ3MSKxk1Un5mBHGwcV69CqkKvbVvSwFBhDRtnbFCDX -> X-kopernikus1v3ujye6nv5ufd23s6a3cl9323n7mkt3hmw46gz
// Admin: PrivateKey-vmRQiZeXEXYMyJhEiqdC2z5JhuDbxL8ix9UVvjgMu2Er1NepE => X-kopernikus1g65uqn6t77p656w64023nh8nd9updzmxh8ttv3
// Admin C-chain: 0x1f0e5c64afdf53175f78846f7125776e76fa8f34
// KYC: PrivateKey-ewoqjP7PxY4yr3iLTpLisriqt94hdyDFNgchSxGGztUrTXtNN => X-kopernikus18jma8ppw3nhx5r4ap8clazz0dps7rv5uuvjh68
// KYC C-chain: 0x8db97c7cece249c2b98bdc0226cc4c2a57bf52fc
// Gas Fee: PrivateKey-Ge71NJhUY3TjZ9dLohijSnNq46QxobjqxHGMUDAPoVsNFA93w -> X-kopernikus13kyf72ftu4l77kss7xm0kshm0au29s48zjaygq
// Gas Fee C-chain: 0x305cea207112c0561033133f816d7a2233699f06
// MultiSig Owner 1: PrivateKey-2Vtf2ZhTRz6WcVcSH7cS7ghKneZxZ2L5W8assdCcaNDVdpoYfY -> X-kopernikus1jla8ty5c9ud6lsj8s4re2dvzvfxpzrxdcrd8q7
// MultiSig Owner 2: PrivateKey-XQFgPzByKfqFfpVTafmZHBqfaw4hsDTGbbcArUg4unMiEKvrD -> X-kopernikus15hscuhlg5tkv4wwrujqgarne3tau83wrpp2d0d
var (
	//go:embed node6
	//go:embed node7
	node6DirPath   embed.FS
	node7DirPath   embed.FS
	caminoNodePath = os.ExpandEnv("$CAMINO_NODE_PATH")
)

// Blocks until a signal is received on [signalChan], upon which
// [n.Stop()] is called. If [signalChan] is closed, does nothing.
// Closes [closedOnShutdownChan] amd [signalChan] when done shutting down network.
// This function should only be called once.
func shutdownOnSignal(
	log logging.Logger,
	n network.Network,
	signalChan chan os.Signal,
	closedOnShutdownChan chan struct{},
) {
	sig := <-signalChan
	log.Info("got OS signal", zap.Stringer("signal", sig))
	if err := n.Stop(context.Background()); err != nil {
		log.Info("error stopping network", zap.Error(err))
	}
	signal.Reset()
	close(signalChan)
	close(closedOnShutdownChan)
}

// Shows example usage of the Camino Network Runner.
// Creates a local five node Camino network
// and waits for all nodes to become healthy.
// The network runs until the user provides a SIGINT or SIGTERM.
func main() {
	// Create the logger
	logFactory := logging.NewFactory(logging.Config{
		DisplayLevel: logging.Info,
		LogLevel:     logging.Debug,
	})
	log, err := logFactory.Make("main")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if caminoNodePath == "" {
		log.Fatal("fatal error, CAMINO_NODE_PATH is not set")
		os.Exit(1)
	}

	if err := run(log, caminoNodePath); err != nil {
		log.Fatal("fatal error", zap.Error(err))
		os.Exit(1)
	}
}

func run(log logging.Logger, binaryPath string) error {
	// Create the network
	nw, err := local.NewCustomNetwork(log, binaryPath, true, postProcessConfig)
	if err != nil {
		return err
	}
	defer func() { // Stop the network when this function returns
		if err := nw.Stop(context.Background()); err != nil {
			log.Info("error stopping network", zap.Error(err))
		}
	}()

	// When we get a SIGINT or SIGTERM, stop the network and close [closedOnShutdownCh]
	signalsChan := make(chan os.Signal, 1)
	signal.Notify(signalsChan, syscall.SIGINT)
	signal.Notify(signalsChan, syscall.SIGTERM)
	closedOnShutdownCh := make(chan struct{})
	go func() {
		shutdownOnSignal(log, nw, signalsChan, closedOnShutdownCh)
	}()
	// Print the node names
	nodeNames, err := nw.GetNodeNames()
	if err != nil {
		return err
	}
	log.Info("current network's nodes", zap.Strings("nodes", nodeNames))

	node6Dir, err := fs.Sub(node6DirPath, "node6")
	if err != nil {
		panic(err)
	}
	flagsBytes, err := fs.ReadFile(node6Dir, "flags.json")
	if err != nil {
		panic(err)
	}
	flags := map[string]interface{}{}
	if err = json.Unmarshal(flagsBytes, &flags); err != nil {
		panic(err)
	}

	// Add 6th node
	stakingKey, err := fs.ReadFile(node6Dir, "staking.key")
	if err != nil {
		panic(err)
	}
	stakingCert, err := fs.ReadFile(node6Dir, "staking.crt")
	if err != nil {
		panic(err)
	}
	stakingSigningKey, err := fs.ReadFile(node6Dir, "signer.key")
	if err != nil {
		panic(err)
	}
	encodedStakingSigningKey := base64.StdEncoding.EncodeToString(stakingSigningKey)

	nodeConfig := node.Config{
		Name:              "node6",
		BinaryPath:        binaryPath,
		StakingKey:        string(stakingKey),
		StakingCert:       string(stakingCert),
		StakingSigningKey: encodedStakingSigningKey,
		IsBeacon:          true,
		// The flags below would override the config in this node's config file,
		// if it had one.
		Flags: map[string]interface{}{
			config.LogLevelKey:    logging.Debug,
			config.HTTPHostKey:    "0.0.0.0",
			config.HTTPPortKey:    flags["http-port"],
			config.StakingPortKey: flags["staking-port"],
		},
	}
	if _, err := nw.AddNode(nodeConfig); err != nil {
		return err
	}

	node7Dir, err := fs.Sub(node6DirPath, "node7")
	if err != nil {
		panic(err)
	}
	flagsBytes, err = fs.ReadFile(node7Dir, "flags.json")
	if err != nil {
		panic(err)
	}
	flags = map[string]interface{}{}
	if err = json.Unmarshal(flagsBytes, &flags); err != nil {
		panic(err)
	}

	// Add 7th node
	stakingKeyN7, err := fs.ReadFile(node7Dir, "staker.key")
	if err != nil {
		panic(err)
	}
	stakingCertN7, err := fs.ReadFile(node7Dir, "staker.crt")
	if err != nil {
		panic(err)
	}
	stakingSigningKeyN7, err := fs.ReadFile(node7Dir, "signer.key")
	if err != nil {
		panic(err)
	}
	encodedStakingSigningKeyN7 := base64.StdEncoding.EncodeToString(stakingSigningKeyN7)

	nodeConfig7 := node.Config{
		Name:              "node7",
		BinaryPath:        binaryPath,
		StakingKey:        string(stakingKeyN7),
		StakingCert:       string(stakingCertN7),
		StakingSigningKey: encodedStakingSigningKeyN7,
		IsBeacon:          true,
		// The flags below would override the config in this node's config file,
		// if it had one.
		Flags: map[string]interface{}{
			config.LogLevelKey:    logging.Debug,
			config.HTTPHostKey:    "0.0.0.0",
			config.HTTPPortKey:    flags["http-port"],
			config.StakingPortKey: flags["staking-port"],
		},
	}
	if _, err := nw.AddNode(nodeConfig7); err != nil {
		return err
	}

	// Wait until the nodes in the network are ready
	ctx, cancel := context.WithTimeout(context.Background(), healthyTimeout)
	defer cancel()
	log.Info("waiting for all nodes to report healthy...")
	if err := nw.Healthy(ctx); err != nil {
		return err
	}

	// Print the node names
	nodeNames, err = nw.GetNodeNames()
	if err != nil {
		return err
	}
	// Will have the new node but not the removed one
	log.Info("All nodes healthy. Updated network's nodes", zap.Strings("nodes", nodeNames))
	log.Info("Network will run until you CTRL + C to exit...")
	// Wait until done shutting down network after SIGINT/SIGTERM
	<-closedOnShutdownCh
	return nil
}

func postProcessConfig(config *network.Config) {
	var genesisMap map[string]interface{}
	json.Unmarshal([]byte(config.Genesis), &genesisMap)

	camino, ok := genesisMap["camino"].(map[string]interface{})
	if !ok {
		panic(errors.New("could not get camino"))
	}
	depositOffers, ok := camino["depositOffers"].([]interface{})
	if !ok {
		panic(errors.New("could not get depositOffers in genesis"))
	}
	depositOffers = append(depositOffers, map[string]interface{}{
		"memo":                    "presale3y",
		"interestRateNominator":   80000,
		"startOffset":             0,
		"endOffset":               112795200,
		"minAmount":               1,
		"minDuration":             110376000,
		"maxDuration":             110376000,
		"unlockPeriodDuration":    31536000,
		"noRewardsPeriodDuration": 15768000,
		"flags":                   map[string]interface{}{"locked": true},
	})
	camino["depositOffers"] = depositOffers

	var cChainGenesis map[string]interface{}
	cChainGenesisString, ok := genesisMap["cChainGenesis"].(string)
	if !ok {
		panic(errors.New("could not get cChainGenesis in genesis"))
	}

	err := json.Unmarshal([]byte(cChainGenesisString), &cChainGenesis)
	if err != nil {
		panic(err)
	}

	alloc, ok := cChainGenesis["alloc"].(map[string]interface{})
	if !ok {
		panic(errors.New("could not get alloc in genesis"))
	}

	initialMultisigAddresses := make([]interface{}, 0)

	if caminoInitialMultisigAddresses, ok := camino["initialMultisigAddresses"].([]interface{}); ok {
		initialMultisigAddresses = caminoInitialMultisigAddresses
	}

	initialMultisigAddresses = append(initialMultisigAddresses, map[string]interface{}{
		"Alias":     "X-kopernikus1fwrv3kj5jqntuucw67lzgu9a9tkqyczxgcvpst", //alias
		"Threshold": 1,
		"Addresses": []string{
			"X-kopernikus1jla8ty5c9ud6lsj8s4re2dvzvfxpzrxdcrd8q7",
			"X-kopernikus15hscuhlg5tkv4wwrujqgarne3tau83wrpp2d0d",
		},
	})

	camino["initialMultisigAddresses"] = initialMultisigAddresses

	allocations, ok := camino["allocations"].([]interface{})
	if !ok {
		panic(errors.New("could not get allocations in genesis"))
	}

	// add funds to admin address
	for _, allocation := range allocations {

		allocationMap, ok := allocation.(map[string]interface{})
		if !ok {
			panic(errors.New("could not get allocation in genesis"))
		}
		if allocationMap["avaxAddr"] != "X-kopernikus1g65uqn6t77p656w64023nh8nd9updzmxh8ttv3" {
			continue
		}
		platformAllocations, ok := allocationMap["platformAllocations"].([]interface{})
		if !ok {
			panic(errors.New("could not get platformAllocations in genesis"))
		}
		platformAllocations = append(platformAllocations, map[string]interface{}{"amount": 4000000000000})
		allocationMap["platformAllocations"] = platformAllocations
	}

	// add new address with p-funds
	allocations = append(allocations, map[string]interface{}{
		"ethAddr":  "0x0000000000000000000000000000000000000000",
		"avaxAddr": "Χ-kopernikus1fwrv3kj5jqntuucw67lzgu9a9tkqyczxgcvpst",
		"amount":   1000000000000,
		"addressStates": map[string]interface{}{
			"consortiumMember": true,
			"kycVerified":      true,
		},
		"platformAllocations": []interface{}{map[string]interface{}{
			"amount": 200000000000000000,
		}},
	})

	allocations = append(allocations, map[string]interface{}{
		"ethAddr":  "0x0000000000000000000000000000000000000000",
		"avaxAddr": "X-kopernikus1s93gzmzuvv7gz8q4l83xccrdchh8mtm3xm5s2g",
		"addressStates": map[string]interface{}{
			"consortiumMember": true,
			"kycVerified":      true,
		},
		"platformAllocations": []interface{}{map[string]interface{}{"amount": 4000000000000}},
	})

	allocations = append(allocations, map[string]interface{}{
		"ethAddr":  "0x0000000000000000000000000000000000000000",
		"avaxAddr": "Χ-kopernikus1jla8ty5c9ud6lsj8s4re2dvzvfxpzrxdcrd8q7",
		"amount":   1000000000000,
		"addressStates": map[string]interface{}{
			"consortiumMember": true,
			"kycVerified":      true,
		},
		"platformAllocations": []interface{}{map[string]interface{}{
			"amount": 200000000000000000,
		}},
	})

	camino["allocations"] = allocations

	cChainGenesis["initialAdmin"] = "0x1f0e5c64afdf53175f78846f7125776e76fa8f34"

	alloc["1f0e5c64afdf53175f78846f7125776e76fa8f34"] = map[string]interface{}{ // adminAddress
		"balance": "0x295BE96E64066972000000",
	}
	alloc["305cea207112c0561033133f816d7a2233699f06"] = map[string]interface{}{ // gasFeeAddress
		"balance": "0x295BE96E64066972000000",
	}
	cChainGenesis["alloc"] = alloc

	cChainGenesisBytes, err := json.Marshal(cChainGenesis)
	if err != nil {
		panic(err)
	}

	genesisMap["cChainGenesis"] = string(cChainGenesisBytes)

	// now we can marshal the *whole* thing into bytes
	updatedGenesis, err := json.Marshal(genesisMap)
	if err != nil {
		panic(err)
	}
	config.Genesis = string(updatedGenesis)
}
