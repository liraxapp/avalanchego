// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"

	"github.com/liraxapp/avalanchego/nat"
	"github.com/liraxapp/avalanchego/node"
	"github.com/liraxapp/avalanchego/utils"
	"github.com/liraxapp/avalanchego/utils/constants"
	"github.com/liraxapp/avalanchego/utils/crypto"
	"github.com/liraxapp/avalanchego/utils/dynamicip"
	"github.com/liraxapp/avalanchego/utils/logging"
)

const (
	header = "" +
		`     _____               .__                       .__` + "\n" +
		`    /  _  \___  _______  |  | _____    ____   ____ |  |__   ____    ,_ o` + "\n" +
		`   /  /_\  \  \/ /\__  \ |  | \__  \  /    \_/ ___\|  |  \_/ __ \   / //\,` + "\n" +
		`  /    |    \   /  / __ \|  |__/ __ \|   |  \  \___|   Y  \  ___/    \>> |` + "\n" +
		`  \____|__  /\_/  (____  /____(____  /___|  /\___  >___|  /\___  >    \\` + "\n" +
		`          \/           \/          \/     \/     \/     \/     \/`
)

var (
	stakingPortName = fmt.Sprintf("%s-staking", constants.AppName)
	httpPortName    = fmt.Sprintf("%s-http", constants.AppName)
)

// main is the primary entry point to Avalanche.
func main() {
	// parse config using viper
	if err := parseViper(); err != nil {
		fmt.Printf("parsing parameters returned with error %s\n", err)
		return
	}

	logFactory := logging.NewFactory(Config.LoggingConfig)
	defer logFactory.Close()

	log, err := logFactory.Make()
	if err != nil {
		fmt.Printf("starting logger failed with: %s\n", err)
		return
	}
	fmt.Println(header)

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered panic from", r)
		}
	}()

	defer func() {
		if err := Config.DB.Close(); err != nil {
			log.Warn("failed to close the node's DB: %s", err)
		}
		log.StopOnPanic()
		log.Stop()
	}()

	// Track if sybil control is enforced
	if !Config.EnableStaking && Config.EnableP2PTLS {
		log.Warn("Staking is disabled. Sybil control is not enforced.")
	}
	if !Config.EnableStaking && !Config.EnableP2PTLS {
		log.Warn("Staking and p2p encryption are disabled. Packet spoofing is possible.")
	}

	// Check if transaction signatures should be checked
	if !Config.EnableCrypto {
		log.Warn("transaction signatures are not being checked")
	}
	crypto.EnableCrypto = Config.EnableCrypto

	if err := Config.ConsensusParams.Valid(); err != nil {
		log.Error("consensus parameters are invalid: %s", err)
		return
	}

	// Track if assertions should be executed
	if Config.LoggingConfig.Assertions {
		log.Debug("assertions are enabled. This may slow down execution")
	}

	// SupportsNAT() for NoRouter is false.
	// Which means we tried to perform a NAT activity but we were not successful.
	if Config.AttemptedNATTraversal && !Config.Nat.SupportsNAT() {
		log.Error("UPnP or NAT-PMP router attach failed, you may not be listening publicly," +
			" please confirm the settings in your router")
	}

	mapper := nat.NewPortMapper(log, Config.Nat)
	defer mapper.UnmapAllPorts()

	// Open staking port we want for NAT Traversal to have the external port
	// (Config.StakingIP.Port) to connect to our internal listening port
	// (Config.InternalStakingPort) which should be the same in most cases.
	mapper.Map(
		"TCP",
		Config.StakingIP.IP().Port,
		Config.StakingIP.IP().Port,
		stakingPortName,
		&Config.StakingIP,
		Config.DynamicUpdateDuration,
	)

	// Open the HTTP port iff the HTTP server is not listening on localhost
	if Config.HTTPHost != "127.0.0.1" && Config.HTTPHost != "localhost" {
		// For NAT Traversal we want to route from the external port
		// (Config.ExternalHTTPPort) to our internal port (Config.HTTPPort)
		mapper.Map(
			"TCP",
			Config.HTTPPort,
			Config.HTTPPort,
			httpPortName,
			nil,
			Config.DynamicUpdateDuration,
		)
	}

	// Regularly updates our public IP (or does nothing, if configured that way)
	externalIPUpdater := dynamicip.NewDynamicIPManager(
		Config.DynamicPublicIPResolver,
		Config.DynamicUpdateDuration,
		log,
		&Config.StakingIP,
	)
	defer externalIPUpdater.Stop()

	log.Info("this node's IP is set to: %s", Config.StakingIP.IP())

	for {
		shouldRestart, err := run(log, logFactory)
		if err != nil {
			break
		}
		// If the node says it should restart, do that. Otherwise, end the program.
		if !shouldRestart {
			break
		}
		log.Info("restarting node")
	}
}

// Initialize and run the node.
// Returns true if the node should restart after this function returns.
func run(log logging.Logger, logFactory logging.Factory) (bool, error) {
	log.Info("initializing node")
	node := node.Node{}
	restarter := &restarter{
		node:          &node,
		shouldRestart: &utils.AtomicBool{},
	}
	if err := node.Initialize(&Config, log, logFactory, restarter); err != nil {
		log.Error("error initializing node: %s", err)
		return restarter.shouldRestart.GetValue(), err
	}

	log.Debug("dispatching node handlers")
	err := node.Dispatch()
	if err != nil {
		log.Debug("node dispatch returned: %s", err)
	}
	return restarter.shouldRestart.GetValue(), nil
}

// restarter implements utils.Restarter
type restarter struct {
	node *node.Node
	// If true, node should restart after shutting down
	shouldRestart *utils.AtomicBool
}

// Restarter shuts down the node and marks that it should restart
// It is safe to call Restart multiple times
func (r *restarter) Restart() {
	r.shouldRestart.SetValue(true)
	// Shutdown is safe to call multiple times because it uses sync.Once
	r.node.Shutdown()
}
