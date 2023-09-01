package ugdvesting

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/x/ugdvesting/keeper"
)

func (am AppModule) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {
	keeper.GetCache(ctx, am.keeper)
}
