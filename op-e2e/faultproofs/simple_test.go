package faultproofs

import (
	"context"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSimple_ChallengerWins(t *testing.T) {
	op_e2e.InitParallel(t)
	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputAlphabetGame(ctx, "sequencer", 3, common.Hash{0xff})
	game.LogGameData(ctx)

	// The dispute game should have a zero balance
	balance := game.WethBalance(ctx, game.Addr())
	require.Zero(t, balance.Uint64())

	//alice := sys.Cfg.Secrets.Addresses().Alice

	// Grab the root claim
	claim := game.RootClaim(ctx)
	opts := challenger.WithPrivKey(sys.Cfg.Secrets.Alice)
	game.StartChallenger(ctx, "sequencer", "Challenger", opts)
	game.LogGameData(ctx)

	// Perform a few moves
	claim = claim.WaitForCounterClaim(ctx)
	game.LogGameData(ctx)

	claim = claim.Attack(ctx, common.Hash{})
	claim = claim.WaitForCounterClaim(ctx)
	game.LogGameData(ctx)

	/*
	 *claim = claim.Attack(ctx, common.Hash{})
	 *game.LogGameData(ctx)
	 *_ = claim.WaitForCounterClaim(ctx)
	 */

	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))
	game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
	game.LogGameData(ctx)
}
