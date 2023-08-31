package ante

import (
	"fmt"

	"github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/x/ugdvesting/types"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ugdvestingmodulekeeper "github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/x/ugdvesting/keeper"
)

// ValidateBasicDecorator will call tx.ValidateBasic and return any non-nil error.
// If ValidateBasic passes, decorator calls next AnteHandler in chain. Note,
// ValidateBasicDecorator decorator will not get executed on ReCheckTx since it
// is not dependent on application state.
type ValidateBasicDecorator struct {
	bankKeeper       bankkeeper.Keeper
	ugdVestingKeeper ugdvestingmodulekeeper.Keeper
}

func NewValidateBasicDecorator(bk bankkeeper.Keeper, uk ugdvestingmodulekeeper.Keeper) ValidateBasicDecorator {
	return ValidateBasicDecorator{
		bankKeeper:       bk,
		ugdVestingKeeper: uk,
	}
}

func (vbd ValidateBasicDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// no need to validate basic on recheck tx, call next antehandler
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}

	if err := ValidateTransaction(ctx, vbd.bankKeeper, vbd.ugdVestingKeeper, tx.GetMsgs()); err != nil {
		return ctx, err
	}

	if err := tx.ValidateBasic(); err != nil {
		return ctx, err
	}

	return next(ctx, tx, simulate)
}

func ValidateTransaction(ctx sdk.Context, bk bankkeeper.Keeper, uk ugdvestingmodulekeeper.Keeper, msgs []sdk.Msg) error {
	allowTransaction := true

	// res, found := bk.GetDenomMetaData(ctx, "ugd")
	// if !found {
	// 	return &types.MyError{Message: "Denomination has not been found in bank keeper"}
	// }

	params := uk.GetParams(ctx)
	fmt.Println("PARAMS ", params)
	denom := params.Denom
	coinPowerValue := params.CoinPowerValue
	precision := params.Precision

	for _, msg := range msgs {
		if msgBank, ok := msg.(*banktypes.MsgSend); ok {
			addr, err := sdk.AccAddressFromBech32(msgBank.FromAddress)
			account := bk.GetBalance(ctx, addr, denom)
			if account.Denom != denom {
				return nil
			}

			if err != nil {
				return err
			}

			vesting := types.HegdehogRequestGetVestingByAddr(addr.String())
			if vesting == nil {
				return nil
			}

			isInMintingList := types.HegdehogCheckIfInMintingList(addr.String())
			if isInMintingList {
				return &types.MyError{
					Message: fmt.Sprintf("Address: %s should not be in minting and vesting list", addr.String()),
				}
			}

			unvestedAmount := types.GetUnvestedAmount(*vesting)
			messageAmount := msgBank.Amount.AmountOf(denom)
			accountAmount := account.Amount

			// check if transaction is allowed based on unvested, transaction and account balance
			accountRequiredBalance := unvestedAmount.Add(messageAmount)
			allowTransaction = unvestedAmount.LTE(sdkmath.NewInt(0)) || accountAmount.GTE(accountRequiredBalance)

			if !allowTransaction {
				err := &types.MyError{
					Message: fmt.Sprintf(
						"%v with %.18f%v unvested, need least %.18f%v, when doing transaction of %.18f%v, but account has %.18f%v.",
						addr.String(),
						types.SdkIntToFloat(unvestedAmount, uint(precision), float64(coinPowerValue)),
						denom,
						types.SdkIntToFloat(accountRequiredBalance, uint(precision), float64(coinPowerValue)),
						denom,
						types.SdkIntToFloat(messageAmount, uint(precision), float64(coinPowerValue)),
						denom,
						types.SdkIntToFloat(accountAmount, uint(precision), float64(coinPowerValue)),
						denom,
					),
				}

				return err
			}
		}
	}

	return nil
}
