package keeper

import (
	"github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/x/ugdvesting/types"
)

var _ types.QueryServer = Keeper{}
