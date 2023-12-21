package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/testutil/keeper"
	"github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/x/ugdvesting/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.UgdvestingKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
