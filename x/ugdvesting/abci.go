package ugdvesting

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const startBlockHeight = 420 // example block height for testing

func (am AppModule) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {
	k := am.keeper
	if ctx.BlockHeight() >= startBlockHeight {
		k.ProcessPendingVesting(ctx)
	}
	if ctx.BlockHeight()%10 == 0 {
		// Call the function to process the vesting accounts
		k.ProcessVestingAccounts(ctx)
	}
}
