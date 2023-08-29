package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	DefaultCoinPower      uint32 = 18
	DefaultCoinPowerValue uint64 = 1000000000000000000
	DefaultPrecision      uint32 = 256
	Denom                 string = "ugd"
)

// NewParams creates a new Params instance
func NewParams(coinPower uint32, coinPowerValue uint64, precision uint32, denom string) Params {
	return Params{
		CoinPower:      coinPower,
		CoinPowerValue: coinPowerValue,
		Precision:      precision,
		Denom:          denom,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultCoinPower,
		DefaultCoinPowerValue,
		DefaultPrecision,
		Denom,
	)
}

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

// Validate validates the set of params
func (p Params) Validate() error {
	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}
