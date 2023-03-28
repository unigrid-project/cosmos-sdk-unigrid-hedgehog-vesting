package types

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	sdkmath "cosmossdk.io/math"
	durationLib "github.com/sosodev/duration"
)

const (
	Denom          = "ugd"
	CoinPower      = 18
	CoinPowerValue = 1000000000000000000
	precision      = 256
)

type Vesting struct {
	Amount   sdkmath.Int `json:"amount"`
	Start    string      `json:"start"`
	Duration string      `json:"duration"`
	Parts    int64       `json:"parts"`
}

func GetUnvestedAmount(vesting Vesting) sdkmath.Int {
	timeStart, _ := time.Parse(time.RFC3339, vesting.Start)
	timeNow := time.Now()
	timePassed := timeNow.Sub(timeStart)

	vestingDurationLib, err := durationLib.Parse(vesting.Duration)
	if err != nil {
		panic(err)
	}

	vestingDuration := vestingDurationLib.ToTimeDuration()
	timeEnd := timeStart.Add(vestingDuration)

	// if vesting has started and not done
	if timePassed.Seconds() > 0 && timeEnd.After(timeNow) {
		partDuration := vestingDuration.Seconds() / float64(vesting.Parts)
		partAmount := vesting.Amount.Quo(sdkmath.NewInt(vesting.Parts))
		// round down, to get current part
		partNow := int64(timePassed.Seconds() / partDuration)
		vested := partAmount.Mul(sdkmath.NewInt(partNow))
		unvested := vesting.Amount.Sub(vested)

		return unvested
	}

	return sdkmath.NewInt(0)
}

func (v *Vesting) AmountBigFloat() *big.Float {
	var float big.Float
	float.SetPrec(precision)
	float.SetInt(v.Amount.BigInt())
	result := float.Quo(&float, big.NewFloat(CoinPowerValue))
	return result
}

func SdkIntToFloat(amount sdkmath.Int) *big.Float {
	var float big.Float
	float.SetPrec(precision)
	float.SetInt(amount.BigInt())
	result := float.Quo(&float, big.NewFloat(CoinPowerValue))
	return result
}

func SdkIntToString(amount sdkmath.Int) string {
	float := SdkIntToFloat(amount)
	return float.Text('f', CoinPower)
}

func (v *Vesting) UnmarshalJSON(data []byte) error {
	// define an alias to avoid infinite recursion
	type vestingAlias Vesting

	// define a struct to handle the raw JSON data
	aux := struct {
		Amount   string `json:"amount"`
		Start    string `json:"start"`
		Duration string `json:"duration"`
		Parts    int64  `json:"parts"`
		*vestingAlias
	}{
		vestingAlias: (*vestingAlias)(v),
	}

	// unmarshal the raw JSON data into the auxiliary struct
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// convert the "amount" string to a big.Float
	amountFloat, _, err := big.ParseFloat(aux.Amount, 10, precision, big.ToNearestEven)
	if err != nil {
		return fmt.Errorf("invalid amount value: %s", aux.Amount)
	}

	// Multiply the float by 10^18 to shift the decimal point 18 places to the right
	amountFloatMul := new(big.Float).Mul(amountFloat, big.NewFloat(CoinPowerValue))

	// Convert the scaled float to a big.Int
	amountInt := new(big.Int)
	amountFloatMul.Int(amountInt)
	v.Amount = sdkmath.NewIntFromBigInt(amountInt)
	v.Duration = aux.Duration
	v.Start = aux.Start
	v.Parts = aux.Parts

	return nil
}
