package types

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"

	//"math"
	"math/big"
	"net/http"
	"sync"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	durationLib "github.com/sosodev/duration"
	"github.com/spf13/viper"
)

const (
	cacheUpdateInterval = 15 * time.Second
)

type Inter interface {
	getMap() map[string]Vesting
}

type Vesting struct {
	Amount   int64  `json:"amount"`
	Start    string `json:"start"`
	Duration string `json:"duration"`
	Parts    int64  `json:"parts"`
}

type VestingAddresses struct {
	VestingAddresses map[string]Vesting `json:"vestingAddresses"`
}

type VestingStorage struct {
	Timestamp         string           `json:"timestamp"`
	PreviousTimeStamp string           `json:"previousTimeStamp"`
	Flags             int              `json:"flags"`
	Hedgehogtype      string           `json:"type"`
	Data              VestingAddresses `json:"data"`
	PreviousData      VestingAddresses `json:"previousData"`
	Signature         string           `json:"signature"`
}

type VestingCache struct {
	stop chan struct{}

	wg              sync.WaitGroup
	mu              sync.RWMutex
	vestedAccounts  map[string]Vesting
	vestingChachMap map[string]Vesting
	lastChanged     string
}

type GridSpork struct {
	MintStorageEntries struct {
		Amount      string `json:"amount"`
		LastChanged string `json:"lastchanged"`
		MintSupply  struct {
			Amount      string `json:"amount"`
			LastChanged string `json:"lastchanged"`
		}
		VestingStorageEntries struct {
			Amount      string `json:"amount"`
			LastChanged string `json:"lastchanged"`
		}
	}
}

func getMap(m map[string]Vesting) map[string]Vesting {
	return m
}

func (vc *VestingCache) cleanupCache() {
	t := time.NewTicker(cacheUpdateInterval)

	defer t.Stop()

	for {
		select {
		case <-vc.stop:
			return
		case <-t.C:
			vc.mu.Lock()
			hedgehogUrl := viper.GetString("hedgehog.hedgehog_url")
			if vc.LatestUpdate(hedgehogUrl) {
				vc.UpdateVesting(hedgehogUrl)
			}
			vc.mu.Unlock()
		}
	}
}

func (vc *VestingCache) LatestUpdate(endpoint string) bool {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	response, err := client.Get(endpoint + "/gridspork")

	if err != nil {
		if err == io.EOF {
			fmt.Println("Received empty response from hedgehog server.")
		} else {
			fmt.Println("Error accessing hedgehog:", err.Error())
		}
		return false
	}

	defer response.Body.Close()

	if response.ContentLength == 0 {
		fmt.Println("Empty response from hedgehog")
		return false
	}

	var res GridSpork
	body, errBody := io.ReadAll(response.Body)

	if errBody != nil {
		fmt.Println(errBody.Error())
		return false
	}

	errUnmar := json.Unmarshal(body, &res)

	if errUnmar != nil {
		fmt.Println(errUnmar.Error())
		return false
	}

	if res.MintStorageEntries.VestingStorageEntries.LastChanged == "never" {
		return false
	}

	latestTime, _ := time.Parse(time.RFC3339, vc.lastChanged)
	hedgehogTime, _ := time.Parse(time.RFC3339, res.MintStorageEntries.VestingStorageEntries.LastChanged)

	if latestTime.Unix() < hedgehogTime.Unix() {
		vc.lastChanged = res.MintStorageEntries.VestingStorageEntries.LastChanged
		return true
	}
	return false
}

func (vc *VestingCache) UpdateVesting(endpoint string) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	response, err := client.Get(endpoint + "/gridspork/vesting-storage")

	if err != nil {
		if err == io.EOF {
			fmt.Println("Received empty response from hedgehog server.")
		} else {
			fmt.Println("Error accessing hedgehog:", err.Error())
		}
		return
	}

	defer response.Body.Close()

	if response.ContentLength == 0 {
		fmt.Println("Empty response from hedgehog")
		return
	}

	fmt.Println(response.StatusCode)

	var res VestingStorage
	body, errBody := io.ReadAll(response.Body)

	if errBody != nil {
		fmt.Println("body error: " + errBody.Error())
		return
	}

	errUnmar := json.Unmarshal(body, &res)

	if errUnmar != nil {
		fmt.Println("Unmarshall error: " + errUnmar.Error())
		return
	}

	var temp map[string]Vesting

	for k, v := range res.Data.VestingAddresses {
		lc, ok := vc.vestingChachMap[k]
		if !ok {
			temp[k] = v
			break
		}

		if v.Start != lc.Start || v.Amount != lc.Amount || v.Parts != lc.Parts || v.Duration != lc.Duration {
			temp[k] = v
		}
	}

	vc.vestingChachMap = temp

}

func updateVestingAcc(address string, v Vesting) {
	s := strings.TrimPrefix(address, "Address(wif=")
	acc, err := sdk.AccAddressFromBech32(s)
	if err != nil {
		fmt.Println("address error: " + err.Error())
		return
	}
	baseAcc := authtypes.NewBaseAccountWithAddress(acc)
	coins := sdk.NewCoins(sdk.NewCoin("ugd", sdk.NewInt(int64(v.Amount)*int64(math.Pow10(8)))))
	timeStart, _ := time.Parse(time.RFC3339, v.Start)

	vestingDurationLib, err := durationLib.Parse(v.Duration)
	vestingDuration := vestingDurationLib.ToTimeDuration()

	period := vestingtypes.Period{
		Length: int64(vestingDuration.Seconds()) / v.Parts,
		Amount: sdk.NewCoins(sdk.NewCoin("ugd", sdk.NewInt((int64(v.Amount) * int64(math.Pow10(8)/float64(v.Parts)))))),
	}
	periods := vestingtypes.Periods{
		period,
	}

	vestingAcc := vestingtypes.NewPeriodicVestingAccount(baseAcc, coins, timeStart.Unix(), periods)

}

func NewCache() *VestingCache {
	vc := &VestingCache{
		stop:            make(chan struct{}),
		vestingChachMap: make(map[string]Vesting),
		vestedAccounts:  make(map[string]Vesting),
		lastChanged:     "never",
	}
	vc.wg.Add(1)
	go func() {
		defer vc.wg.Done()
		vc.cleanupCache()
	}()
	return vc
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
		partAmount := vesting.Amount / vesting.Parts

		// round down, to get current part
		partNow := int64(timePassed.Seconds() / partDuration)
		vested := partAmount * partNow
		unvested := vesting.Amount - vested

		return sdkmath.NewInt(unvested)
	}

	return sdkmath.NewInt(0)
}

func SdkIntToFloat(amount sdkmath.Int, precision uint, coinPowerValue float64) *big.Float {
	var float big.Float
	float.SetPrec(precision)
	float.SetInt(amount.BigInt())
	result := float.Quo(&float, big.NewFloat(coinPowerValue))
	return result
}

func SdkIntToString(amount sdkmath.Int, precision uint, coinPowerValue float64, coinPower int) string {
	float := SdkIntToFloat(amount, precision, coinPowerValue)
	return float.Text('f', coinPower)
}

/*func (v *Vesting) UnmarshalJSON(data []byte) error {
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
	amountFloat, _, err := big.ParseFloat(aux.Amount, 10, 256, big.ToNearestEven)
	if err != nil {
		return fmt.Errorf("invalid amount value: %s", aux.Amount)
	}

	// Multiply the float by 10^18 to shift the decimal point 18 places to the right
	amountFloatMul := new(big.Float).Mul(amountFloat, big.NewFloat(math.Pow10(8)))

	// Convert the scaled float to a big.Int
	amountInt := new(big.Int)
	amountFloatMul.Int(amountInt)
	v.Amount = sdkmath.NewIntFromBigInt(amountInt)
	v.Duration = aux.Duration
	v.Start = aux.Start
	v.Parts = aux.Parts

	return nil
}*/
