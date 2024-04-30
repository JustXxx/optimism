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

func TestSimple_Alphabet_ChallengerWins(t *testing.T) {
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

func TestSimple_Cannon_ChallengerWins(t *testing.T) {
	op_e2e.InitParallel(t)
	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 3, common.Hash{0x01, 0xaa})
	require.NotNil(t, game)
	claim := game.DisputeLastBlock(ctx)

	// Create the root of the cannon trace.
	//claim = claim.Attack(ctx, common.Hash{0x01})
	claim = claim.AttackAt(ctx, common.Hash{0x01}, 0)
	game.LogGameDataF(ctx, "AttackAt[0]")

	t.Logf("TestLog GameDuration: %v", game.GameDuration(ctx))
	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx) * 2)
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))
	game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
	game.LogGameDataF(ctx, "ChallengerWins")
}
