package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	cldr "github.com/ethereum-optimism/optimism/op-program/client/driver"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum-optimism/optimism/op-program/client/l2"
)

// Main executes the client program in a detached context and exits the current process.
// The client runtime environment must be preset before calling this function.
func Main(logger log.Logger) {
	log.Info("Starting fault proof program client")

	if err := RunProgramWithDefault(logger); errors.Is(err, cldr.ErrClaimNotValid) {
		log.Error("Claim is invalid", "err", err)
		os.Exit(1)
	} else if err != nil {
		log.Error("Program failed", "err", err)
		os.Exit(2)
	} else {
		log.Info("Claim successfully verified")
		os.Exit(0)
	}
}

// RunProgramWithDefault executes the Program, while attached to an IO based pre-image oracle, to be served by a host.
func RunProgramWithDefault(logger log.Logger) error {
	println("runing wasm program=======>")

	pClient, hClient := NewOracleClientAndHintWriter()
	l1PreimageOracle := l1.NewCachingOracle(l1.NewPreimageOracle(pClient, hClient))
	l2PreimageOracle := l2.NewCachingOracle(l2.NewPreimageOracle(pClient, hClient))

	bootInfo := NewBootstrapClient(pClient).BootInfo()
	logger.Info("Program Bootstrapped", "bootInfo", bootInfo)
	return runDerivation(
		logger,
		bootInfo.RollupConfig,
		bootInfo.L2ChainConfig,
		bootInfo.L1Head,
		bootInfo.L2Head,
		bootInfo.L2Claim,
		bootInfo.L2ClaimBlockNumber,
		l1PreimageOracle,
		l2PreimageOracle,
	)
}

// runDerivation executes the L2 state transition, given a minimal interface to retrieve data.
func runDerivation(logger log.Logger, cfg *rollup.Config, l2Cfg *params.ChainConfig, l1Head common.Hash, l2Head common.Hash, l2Claim common.Hash, l2ClaimBlockNum uint64, l1Oracle l1.Oracle, l2Oracle l2.Oracle) error {
	l1Source := l1.NewOracleL1Client(logger, l1Oracle, l1Head)
	engineBackend, err := l2.NewOracleBackedL2Chain(logger, l2Oracle, l2Cfg, l2Head)
	if err != nil {
		return fmt.Errorf("failed to create oracle-backed L2 chain: %w", err)
	}
	l2Source := l2.NewOracleEngine(cfg, logger, engineBackend)

	logger.Info("Starting derivation")
	d := cldr.NewDriver(logger, cfg, l1Source, l2Source, l2ClaimBlockNum)
	for {
		if err = d.Step(context.Background()); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}
	}
	return d.ValidateClaim(eth.Bytes32(l2Claim))
}
