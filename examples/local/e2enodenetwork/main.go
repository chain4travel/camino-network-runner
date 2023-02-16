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
var (
	//go:embed node6
	node6DirPath   embed.FS
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
		"avaxAddr": "X-kopernikus1s93gzmzuvv7gz8q4l83xccrdchh8mtm3xm5s2g",
		"addressStates": map[string]interface{}{
			"consortiumMember": true,
			"kycVerified":      true,
		},
		"platformAllocations": []interface{}{map[string]interface{}{"amount": 4000000000000}},
	})
	camino["allocations"] = allocations

	// now we can marshal the *whole* thing into bytes
	updatedGenesis, err := json.Marshal(genesisMap)
	if err != nil {
		panic(err)
	}
	config.Genesis = string(updatedGenesis)
}
