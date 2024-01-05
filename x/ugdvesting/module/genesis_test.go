package ugdvesting_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/unigrid-project/cosmos-unigrid-hedgehog-vesting/testutil/keeper"
	"github.com/unigrid-project/cosmos-unigrid-hedgehog-vesting/testutil/nullify"
	ugdvesting "github.com/unigrid-project/cosmos-unigrid-hedgehog-vesting/x/ugdvesting/module"
	"github.com/unigrid-project/cosmos-unigrid-hedgehog-vesting/x/ugdvesting/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.UgdvestingKeeper(t)
	ugdvesting.InitGenesis(ctx, k, genesisState)
	got := ugdvesting.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
