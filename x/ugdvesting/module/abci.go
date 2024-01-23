package ugdvesting

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const startBlockHeight = 50 // example block height for testing vestsing start

func (am AppModule) BeginBlock(goCtx context.Context) error {
	k := am.keeper
	ctx := sdk.UnwrapSDKContext(goCtx)
	if ctx.BlockHeight() >= startBlockHeight {
		k.ProcessPendingVesting(ctx)
	}
	if ctx.BlockHeight()%10 == 0 {
		// Call the function to process the vesting accounts
		k.ProcessVestingAccounts(ctx)
	}
	// FORE TESTING ONLY TODO: REMOVE OR DISABLE IN PRODUCTION
	// if ctx.BlockHeight() == 9 {
	// 	k.ClearVestingDataStore(ctx)
	// }
	return nil
}
