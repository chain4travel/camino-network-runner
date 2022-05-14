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
	"context"
	"crypto"
	"fmt"
	"net"
	"os/exec"
	"syscall"
	"time"

	"github.com/chain4travel/camino-network-runner/api"
	"github.com/chain4travel/camino-network-runner/network/node"
	"github.com/chain4travel/caminogo/ids"
	"github.com/chain4travel/caminogo/message"
	"github.com/chain4travel/caminogo/network/peer"
	"github.com/chain4travel/caminogo/network/throttling"
	"github.com/chain4travel/caminogo/snow/networking/router"
	"github.com/chain4travel/caminogo/snow/validators"
	"github.com/chain4travel/caminogo/staking"
	avago_utils "github.com/chain4travel/caminogo/utils"
	"github.com/chain4travel/caminogo/utils/constants"
	"github.com/chain4travel/caminogo/utils/logging"
	"github.com/chain4travel/caminogo/version"
	"github.com/prometheus/client_golang/prometheus"
)

// interface compliance
var (
	_ node.Node   = (*localNode)(nil)
	_ NodeProcess = (*nodeProcessImpl)(nil)
	_ getConnFunc = defaultGetConnFunc
)

type getConnFunc func(context.Context, node.Node) (net.Conn, error)

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
// CaminoGo binaries in tests
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
	// The ID of the network this node exists in
	networkID uint32
	// Allows user to make API calls to this node.
	client api.Client
	// The process running this node.
	process NodeProcess
	// The API port
	apiPort uint16
	// The P2P (staking) port
	p2pPort uint16
	// Returns a connection to this node
	getConnFunc getConnFunc
}

func defaultGetConnFunc(ctx context.Context, node node.Node) (net.Conn, error) {
	dialer := net.Dialer{}
	return dialer.DialContext(ctx, constants.NetworkType, net.JoinHostPort(node.GetURL(), fmt.Sprintf("%d", node.GetP2PPort())))
}

// AttachPeer: see Network
func (node *localNode) AttachPeer(ctx context.Context, router router.InboundHandler) (peer.Peer, error) {
	tlsCert, err := staking.NewTLSCert()
	if err != nil {
		return nil, err
	}
	tlsConfg := peer.TLSConfig(*tlsCert)
	clientUpgrader := peer.NewTLSClientUpgrader(tlsConfg)
	conn, err := node.getConnFunc(ctx, node)
	if err != nil {
		return nil, err
	}
	mc, err := message.NewCreator(
		prometheus.NewRegistry(),
		true,
		"",
		10*time.Second,
	)
	if err != nil {
		return nil, err
	}

	metrics, err := peer.NewMetrics(
		logging.NoLog{},
		"",
		prometheus.NewRegistry(),
	)
	if err != nil {
		return nil, err
	}
	ip := avago_utils.IPDesc{
		IP:   net.IPv6zero,
		Port: 0,
	}
	config := &peer.Config{
		Metrics:              metrics,
		MessageCreator:       mc,
		Log:                  logging.NoLog{},
		InboundMsgThrottler:  throttling.NewNoInboundThrottler(),
		OutboundMsgThrottler: throttling.NewNoOutboundThrottler(),
		Network: peer.NewTestNetwork(
			mc,
			node.networkID,
			ip,
			version.ModuleVersionApp,
			tlsCert.PrivateKey.(crypto.Signer),
			ids.Set{},
			100,
		),
		Router:               router,
		VersionCompatibility: version.GetCompatibility(node.networkID),
		VersionParser:        version.NewDefaultApplicationParser(),
		MySubnets:            ids.Set{},
		Beacons:              validators.NewSet(),
		NetworkID:            node.networkID,
		PingFrequency:        constants.DefaultPingFrequency,
		PongTimeout:          constants.DefaultPingPongTimeout,
		MaxClockDifference:   time.Minute,
	}
	peerID, conn, cert, err := clientUpgrader.Upgrade(conn)
	if err != nil {
		return nil, err
	}

	p := peer.Start(
		config,
		conn,
		cert,
		peerID,
	)
	if err != nil {
		return nil, err
	}

	return p, nil
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
	return "127.0.0.1"
}

// See node.Node
func (node *localNode) GetP2PPort() uint16 {
	return node.p2pPort
}

// See node.Node
func (node *localNode) GetAPIPort() uint16 {
	return node.apiPort
}
