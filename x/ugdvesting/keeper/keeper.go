package keeper

import (
	"fmt"
	"sync"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	//paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/unigrid-project/cosmos-unigrid-hedgehog-vesting/x/ugdvesting/types"
)

type (
	Keeper struct {
		cdc                 codec.BinaryCodec
		storeService        store.KVStoreService
		logger              log.Logger
		authKeeper          types.AccountKeeper
		bankKeeper          types.BankKeeper
		mu                  sync.Mutex
		InMemoryVestingData InMemoryVestingData
		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	bk types.BankKeeper,
	ak types.AccountKeeper,

) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:                 cdc,
		storeService:        storeService,
		authority:           authority,
		logger:              logger,
		authKeeper:          ak,
		bankKeeper:          bk,
		InMemoryVestingData: InMemoryVestingData{VestingAccounts: make(map[string]VestingData)},
	}
}

// GetAuthority returns the module's authority.
func (k *Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k *Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k *Keeper) GetAccount(ctx sdk.Context, addr sdk.AccAddress) sdk.AccountI {
	if k.authKeeper == nil {
		fmt.Println("authKeeper is nil")
		return nil
	}
	return k.authKeeper.GetAccount(ctx, addr)
}

func (k *Keeper) SetAccount(ctx sdk.Context, acc sdk.AccountI) {
	k.authKeeper.SetAccount(ctx, acc)
}

func (k *Keeper) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	if k.bankKeeper == nil {
		fmt.Println("bankKeeper is nil")
		return sdk.Coins{}
	}
	return k.bankKeeper.GetAllBalances(ctx, addr)
}
