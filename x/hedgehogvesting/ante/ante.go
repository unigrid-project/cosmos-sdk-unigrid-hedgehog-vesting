package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// NewAnteHandler returns an AnteHandler that checks and increments sequence
// numbers, checks signatures & account numbers, and deducts fees from the first
// signer.
// Stripped from x/auth/ante/ante.go NewAnteHandler.
func NewAnteHandler(options ante.HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "account keeper is required for AnteHandler")
	}
	if options.BankKeeper == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "bank keeper is required for AnteHandler")
	}

	/*sigGasConsumer := options.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = ante.DefaultSigVerificationGasConsumer
	}*/

	anteDecorators := []sdk.AnteDecorator{
		//ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		//ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		NewValidateBasicDecorator(options.BankKeeper.(bankkeeper.Keeper)),
		//ante.NewTxTimeoutHeightDecorator(),
		//ante.NewValidateMemoDecorator(options.AccountKeeper),
		//ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		//ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
		// SetPubKeyDecorator must be called before all signature verification decorators
		//ante.NewSetPubKeyDecorator(options.AccountKeeper),
		//ante.NewValidateSigCountDecorator(options.AccountKeeper),
		//ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		//ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
