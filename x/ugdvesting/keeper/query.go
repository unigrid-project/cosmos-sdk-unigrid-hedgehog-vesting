package keeper

import (
	"github.com/unigrid-project/cosmos-unigrid-hedgehog-vesting/x/ugdvesting/types"
)

var _ types.QueryServer = Keeper{}
