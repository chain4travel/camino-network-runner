// Copyright (C) 2022, Chain4Travel AG. All rights reserved.
//
// This file is a derived work, based on ava-labs code whose
// original notices appear below.
//
// It is distributed under the same license conditions as the
// original code from which it is derived.
//
// Much love to the original authors for their work.
// **********************************************************

package local

import (
	"os/exec"
	"syscall"

	"github.com/ava-labs/avalanche-network-runner/api"
	"github.com/ava-labs/avalanche-network-runner/network/node"
	"github.com/chain4travel/caminogo/ids"
)

// interface compliance
var (
	_ node.Node   = (*localNode)(nil)
	_ NodeProcess = (*nodeProcessImpl)(nil)
)

// NodeConfig configurations which are specific to the
// local implementation of a network / node.
type NodeConfig struct {
	// What type of node this is
	BinaryPath string `json:"binaryPath"`
	// If non-nil, direct this node's Stdout to os.Stdout
	RedirectStdout bool `json:"redirectStdout"`
	// If non-nil, direct this node's Stderr to os.Stderr
	RedirectStderr bool `json:"redirectStderr"`
}

// NodeProcess as an interface so we can mock running
// AvalancheGo binaries in tests
type NodeProcess interface {
	// Start this process
	Start() error
	// Send a SIGTERM to this process
	Stop() error
	// Returns when the process finishes exiting
	Wait() error
}

type nodeProcessImpl struct {
	cmd *exec.Cmd
}

func (p *nodeProcessImpl) Start() error {
	return p.cmd.Start()
}

func (p *nodeProcessImpl) Wait() error {
	return p.cmd.Wait()
}

func (p *nodeProcessImpl) Stop() error {
	return p.cmd.Process.Signal(syscall.SIGTERM)
}

// Gives access to basic nodes info, and to most caminogo apis
type localNode struct {
	// Must be unique across all nodes in this network.
	name string
	// [nodeID] is this node's Avalannche Node ID.
	// Set in network.AddNode
	nodeID ids.ShortID
	// Allows user to make API calls to this node.
	client api.Client
	// The process running this node.
	process NodeProcess
	// The API port
	apiPort uint16
	// The P2P (staking) port
	p2pPort uint16
}

// See node.Node
func (node *localNode) GetName() string {
	return node.name
}

// See node.Node
func (node *localNode) GetNodeID() ids.ShortID {
	return node.nodeID
}

// See node.Node
func (node *localNode) GetAPIClient() api.Client {
	return node.client
}

// See node.Node
func (node *localNode) GetURL() string {
	return "localhost"
}

// See node.Node
func (node *localNode) GetP2PPort() uint16 {
	return node.p2pPort
}

// See node.Node
func (node *localNode) GetAPIPort() uint16 {
	return node.apiPort
}
