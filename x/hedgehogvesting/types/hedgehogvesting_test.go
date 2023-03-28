package types_test

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/timnhanta/ugdvesting/x/hedgehogvesting/types"
)

func TestGetUnvestedAmount(t *testing.T) {
	// unvested should be 0, if vesting not started
	timeNow := time.Now()
	timeNow = timeNow.In(time.FixedZone("CET", 0))
	amount := sdkmath.NewInt(1000)
	duration := "PT10M"
	part := int64(10)
	timeStart := timeNow.Add(10 * time.Minute)
	formattedTimeStart := timeStart.UTC().Format("2006-01-02T15:04:05.999999Z")

	vesting := types.Vesting{
		Amount:   amount,
		Start:    formattedTimeStart,
		Duration: duration,
		Parts:    part,
	}
	unvested := types.GetUnvestedAmount(vesting)

	expected := big.NewInt(0)
	if unvested.BigInt().Cmp(expected) != 0 {
		t.Errorf("unvested = %v, expected = %v", unvested, expected)
	}

	// unvested should be 0, if vesting is done
	timeNow = time.Now()
	timeNow = timeNow.In(time.FixedZone("CET", 0))
	timeStart = timeNow.Add(-11 * time.Minute)
	formattedTimeStart = timeStart.UTC().Format("2006-01-02T15:04:05.999999Z")

	vesting = types.Vesting{
		Amount:   amount,
		Start:    formattedTimeStart,
		Duration: duration,
		Parts:    part,
	}
	unvested = types.GetUnvestedAmount(vesting)

	expected = big.NewInt(0)
	if unvested.BigInt().Cmp(expected) != 0 {
		t.Errorf("unvested = %v, expected = %v", unvested, expected)
	}

	// unvested should be 600, if vesting progress is 4/10
	timeNow = time.Now()
	timeNow = timeNow.In(time.FixedZone("CET", 0))
	timeStart = timeNow.Add(-4 * time.Minute)
	formattedTimeStart = timeStart.UTC().Format("2006-01-02T15:04:05.999999Z")

	vesting = types.Vesting{
		Amount:   amount,
		Start:    formattedTimeStart,
		Duration: duration,
		Parts:    part,
	}
	unvested = types.GetUnvestedAmount(vesting)

	expected = big.NewInt(600)
	if unvested.BigInt().Cmp(expected) != 0 {
		t.Errorf("unvested = %v, expected = %v", unvested, expected)
	}
}

func TestAmountBigFloat(t *testing.T) {
	var expected big.Float
	var bigInt big.Int

	expected.SetPrec(256)
	expected.SetString("7000.707070707070707070")

	bigInt.SetString("7000707070707070707070", 10)

	vesting := types.Vesting{
		Amount:   sdkmath.NewIntFromBigInt(&bigInt),
		Start:    "2023-03-14T18:41:20Z",
		Duration: "PT168H29M58S",
		Parts:    7,
	}

	if vesting.AmountBigFloat().Cmp(&expected) != 0 {
		t.Errorf("Unexpected response. Expected %+v, but got %+v", expected, vesting.AmountBigFloat())
	}
}

func TestSdkIntToFloat(t *testing.T) {
	var expected big.Float
	var bigInt big.Int

	expected.SetPrec(256)
	expected.SetString("7000.707070707070707070")

	bigInt.SetString("7000707070707070707070", 10)
	result := types.SdkIntToFloat(sdkmath.NewIntFromBigInt(&bigInt))

	if result.Cmp(&expected) != 0 {
		t.Errorf("Unexpected response. Expected %+v, but got %+v", expected, result)
	}
}

func TestSdkIntToString(t *testing.T) {
	expected := "7000.707070707070707070"

	var bigInt big.Int
	bigInt.SetString("7000707070707070707070", 10)
	result := types.SdkIntToString(sdkmath.NewIntFromBigInt(&bigInt))

	if result != expected {
		t.Errorf("Unexpected response. Expected %+v, but got %+v", expected, result)
	}

}

func TestUnmarshalJSON(t *testing.T) {
	var vesting types.Vesting
	var bigInt big.Int

	jsonStr := `{"amount":"7000.707070707070707070","start":"2023-03-14T18:41:20Z","duration":"PT168H29M58S","parts":7}`

	err := json.Unmarshal([]byte(jsonStr), &vesting)
	if err != nil {
		panic(err)
	}

	bigInt.SetString("7000707070707070707070", 10)

	expected := types.Vesting{
		Amount:   sdkmath.NewIntFromBigInt(&bigInt),
		Start:    "2023-03-14T18:41:20Z",
		Duration: "PT168H29M58S",
		Parts:    7,
	}

	if vesting.Amount.Neg().Equal(expected.Amount) &&
		vesting.Start != expected.Start &&
		vesting.Duration != expected.Duration &&
		vesting.Parts != expected.Parts {
		t.Errorf("Unexpected response. Expected %+v, but got %+v", expected, vesting)
	}
}
