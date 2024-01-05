package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/unigrid-project/cosmos-unigrid-hedgehog-vesting/testutil/keeper"
	"github.com/unigrid-project/cosmos-unigrid-hedgehog-vesting/x/ugdvesting/keeper"
	"github.com/unigrid-project/cosmos-unigrid-hedgehog-vesting/x/ugdvesting/types"
)

func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, context.Context) {
	k, ctx := keepertest.UgdvestingKeeper(t)
	return k, keeper.NewMsgServerImpl(k), ctx
}

func TestMsgServer(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}
