package ugdvesting

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/viper"
	"github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/x/ugdvesting/keeper"
	"github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/x/ugdvesting/types"
)

func (am AppModule) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {
	// Check if block height is a multiple of 10
	if ctx.BlockHeight()%10 == 0 {
		store := ctx.KVStore(am.keeper.GetStoreKey())
		iterator := sdk.KVStorePrefixIterator(store, types.VestingKey)
		for ; iterator.Valid(); iterator.Next() {
			key := iterator.Key()
			value := iterator.Value()
			fmt.Println("Key:", string(key), "Value:", string(value))
		}
		iterator.Close()
		hedgehogUrl := viper.GetString("hedgehog.hedgehog_url")
		fmt.Println("hedgehogUrl in vesting:", hedgehogUrl)
		vc := keeper.GetCache()
		vc.CallHedgehog(hedgehogUrl+"/gridspork/vesting-storage", ctx, am.keeper)
	}
}
