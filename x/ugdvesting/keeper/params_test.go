package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/testutil/keeper"
	"github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/x/ugdvesting/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.UgdvestingKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
