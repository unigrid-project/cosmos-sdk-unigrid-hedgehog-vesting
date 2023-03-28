package app

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/timnhanta/ugdvesting/x/hedgehogvesting/types"
)

// ValidateBasicDecorator will call tx.ValidateBasic and return any non-nil error.
// If ValidateBasic passes, decorator calls next AnteHandler in chain. Note,
// ValidateBasicDecorator decorator will not get executed on ReCheckTx since it
// is not dependent on application state.
type ValidateBasicDecorator struct {
	bankKeeper bankkeeper.Keeper
}

func NewValidateBasicDecorator(bk bankkeeper.Keeper) ValidateBasicDecorator {
	return ValidateBasicDecorator{
		bankKeeper: bk,
	}
}

func (vbd ValidateBasicDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// no need to validate basic on recheck tx, call next antehandler
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}

	if err := ValidateTransaction(ctx, vbd.bankKeeper, tx.GetMsgs()); err != nil {
		return ctx, err
	}

	if err := tx.ValidateBasic(); err != nil {
		return ctx, err
	}

	return next(ctx, tx, simulate)
}

func ValidateTransaction(ctx sdk.Context, bk bankkeeper.Keeper, msgs []sdk.Msg) error {
	allowTransaction := true

	for _, msg := range msgs {
		if msgBank, ok := msg.(*banktypes.MsgSend); ok {
			addr, err := sdk.AccAddressFromBech32(msgBank.FromAddress)
			account := bk.GetBalance(ctx, addr, types.Denom)
			if account.Denom != types.Denom {
				return nil
			}
			if err != nil {
				return err
			}

			var vesting *types.Vesting
			vesting = types.HegdehogRequestGetVestingByAddr(addr.String())
			if vesting == nil {
				return nil
			}

			unvestedAmount := types.GetUnvestedAmount(*vesting)
			messageAmount := msgBank.Amount.AmountOf(types.Denom)
			accountAmount := account.Amount

			// check if transaction is allowed based on unvested, transaction and account balance
			accountRequiredBalance := unvestedAmount.Add(messageAmount)
			allowTransaction = unvestedAmount.LTE(sdkmath.NewInt(0)) || accountAmount.GTE(accountRequiredBalance)

			if !allowTransaction {
				err := &types.MyError{
					Message: fmt.Sprintf(
						"%v with %.18f%v unvested, need least %.18f%v, when doing transaction of %.18f%v, but account has %.18f%v.",
						addr.String(),
						types.SdkIntToFloat(unvestedAmount),
						types.Denom,
						types.SdkIntToFloat(accountRequiredBalance),
						types.Denom,
						types.SdkIntToFloat(messageAmount),
						types.Denom,
						types.SdkIntToFloat(accountAmount),
						types.Denom,
					),
				}

				return err
			}
		}
	}

	return nil
}
