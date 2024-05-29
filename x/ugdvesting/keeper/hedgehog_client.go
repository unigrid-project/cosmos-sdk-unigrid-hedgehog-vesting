package keeper

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/log"
	math "cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	durationLib "github.com/sosodev/duration"
	"github.com/spf13/viper"
	"github.com/unigrid-project/cosmos-common/common/httpclient"
	//ugdtypes "github.com/unigrid-project/cosmos-unigrid-hedgehog-vesting/x/ugdvesting/types"
)

type VestingData struct {
	Address   string `json:"address"`
	Amount    int64  `json:"amount"`
	Start     string `json:"start"`
	Duration  string `json:"duration"`
	Parts     int    `json:"parts"`
	Block     int64  `json:"block"`
	Percent   int    `json:"percent"`
	Cliff     int    `json:"cliff"`
	Processed bool
}

// InMemoryVestingData holds vesting data in memory.
type InMemoryVestingData struct {
	VestingAccounts map[string]VestingData
}

type HedgehogData struct {
	Timestamp         string `json:"timestamp"`
	PreviousTimeStamp string `json:"previousTimeStamp"`
	Flags             int    `json:"flags"`
	Hedgehogtype      string `json:"type"`
	Data              struct {
		VestingAddresses map[string]VestingData `json:"vestingAddresses"`
	} `json:"data"`
	Signature string `json:"signature"`
}

func (k *Keeper) ProcessPendingVesting(ctx sdk.Context) {
	k.mu.Lock()
	defer k.mu.Unlock()

	currentHeight := ctx.BlockHeight()

	for address, data := range k.InMemoryVestingData.VestingAccounts {
		if data.Block == currentHeight && !data.Processed {
			addr, err := sdk.AccAddressFromBech32(address)
			if err != nil {
				fmt.Println("Error parsing address:", err)
				continue
			}

			account := k.GetAccount(ctx, addr)
			if account == nil {
				fmt.Println("Account not found:", addr)
				continue
			}

			// Convert to PeriodicVestingAccount if it's not already one
			if _, ok := account.(*vestingtypes.PeriodicVestingAccount); !ok {
				if baseAcc, ok := account.(*vestingtypes.DelayedVestingAccount); ok {
					currentBalances := k.GetAllBalances(ctx, addr)
					if currentBalances.IsZero() {
						fmt.Println("No balances found for address:", addr)
						continue
					}

					startTime := ctx.BlockTime().Unix()

					tgeAmount := sdk.Coins{}
					for _, coin := range currentBalances {
						amount := coin.Amount.Mul(math.NewInt(int64(data.Percent))).Quo(math.NewInt(100))
						tgeAmount = append(tgeAmount, sdk.NewCoin(coin.Denom, amount))
					}

					remainingAmount := sdk.Coins{}
					for _, coin := range currentBalances {
						remainingAmount = append(remainingAmount, sdk.NewCoin(coin.Denom, coin.Amount.Sub(tgeAmount.AmountOf(coin.Denom))))
					}

					totalVestingParts := data.Parts - 1

					vestingAmountPerPeriod := sdk.Coins{}
					for _, coin := range remainingAmount {
						vestingAmountPerPeriod = append(vestingAmountPerPeriod, sdk.NewCoin(coin.Denom, coin.Amount.Quo(math.NewInt(int64(totalVestingParts)))))
					}

					rampUpAmountPerPeriod := sdk.Coins{}
					for _, coin := range vestingAmountPerPeriod {
						rampUpAmountPerPeriod = append(rampUpAmountPerPeriod, sdk.NewCoin(coin.Denom, coin.Amount.Quo(math.NewInt(int64(data.Cliff)))))
					}

					periods := vestingtypes.Periods{}
					vestingDuration, err := parseISO8601Duration(data.Duration)
					if err != nil {
						fmt.Println("Error parsing vesting duration:", err)
						continue
					}
					goDurationStr := strconv.FormatInt(vestingDuration, 10) + "s"
					periodTime, _ := time.ParseDuration(goDurationStr)

					// Add the TGE period
					periods = append(periods, vestingtypes.Period{
						Length: int64(periodTime.Seconds()),
						Amount: tgeAmount,
					})

					// Add the cliff periods
					for i := 0; i < int(data.Cliff); i++ {
						periods = append(periods, vestingtypes.Period{
							Length: int64(periodTime.Seconds()),
							Amount: rampUpAmountPerPeriod,
						})
					}

					// Add the remaining vesting periods
					for i := 0; i < int(totalVestingParts); i++ {
						periods = append(periods, vestingtypes.Period{
							Length: int64(periodTime.Seconds()),
							Amount: vestingAmountPerPeriod,
						})
					}

					var pubKeyAny *codectypes.Any
					if baseAcc.GetPubKey() != nil {
						pubKeyAny, err = codectypes.NewAnyWithValue(baseAcc.GetPubKey())
						if err != nil {
							fmt.Println("Error packing public key into Any:", err)
							continue
						}
					}

					baseAccount := &authtypes.BaseAccount{
						Address:       baseAcc.GetAddress().String(),
						PubKey:        pubKeyAny,
						AccountNumber: baseAcc.GetAccountNumber(),
						Sequence:      baseAcc.GetSequence(),
					}

					vestingAcc, err := vestingtypes.NewPeriodicVestingAccount(baseAccount, currentBalances, startTime, periods)
					if err != nil {
						logger := log.NewLogger(os.Stderr)
						logger.Error("Error creating new periodic vesting account", "err", err)
						continue
					}

					k.SetAccount(ctx, vestingAcc)
					data.Processed = true
					k.InMemoryVestingData.VestingAccounts[address] = data
					fmt.Println("Processed vesting data for address:", address)
				}
			}
		}
	}
}

func (k *Keeper) ProcessVestingAccounts(ctx sdk.Context) {
	k.mu.Lock()
	defer func() {
		//fmt.Println("Unlocking k.mu")
		k.mu.Unlock()
	}()

	base := viper.GetString("hedgehog.hedgehog_url")
	hedgehogUrl := base + "/gridspork/vesting-storage"

	response, err := httpclient.Client.Get(hedgehogUrl)
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
		fmt.Println("Received empty response from hedgehog server.")
		return
	}

	var res HedgehogData
	body, err1 := io.ReadAll(response.Body)
	if err1 != nil {
		fmt.Println(err1.Error())
		return
	}

	e := json.Unmarshal(body, &res)
	if e != nil {
		fmt.Println(e.Error())
		return
	}

	// fmt.Println("Received vesting data from Hedgehog:")
	// fmt.Printf("Timestamp: %s\n", res.Timestamp)
	// fmt.Printf("PreviousTimeStamp: %s\n", res.PreviousTimeStamp)
	// fmt.Printf("Flags: %d\n", res.Flags)
	// fmt.Printf("Hedgehogtype: %s\n", res.Hedgehogtype)
	for key, vesting := range res.Data.VestingAddresses {
		address := strings.TrimPrefix(key, "Address(wif=")
		address = strings.TrimSuffix(address, ")")
		addr, err := ConvertStringToAcc(address)
		if err != nil {
			fmt.Println("Error converting address:", err)
			continue
		}

		if k.HasProcessedAddress(ctx, addr) {
			fmt.Printf("Address already processed: %s\n", addr)
			continue
		}

		// Store the parsed data in memory
		vestingData := VestingData{
			Address:   key,
			Amount:    vesting.Amount,
			Start:     vesting.Start,
			Duration:  vesting.Duration,
			Parts:     vesting.Parts,
			Block:     vesting.Block,
			Percent:   vesting.Percent,
			Cliff:     vesting.Cliff,
			Processed: false,
		}

		k.SetVestingDataInMemory(address, vestingData)
		//fmt.Printf("In-memory vesting data set for address: %s\n", address)
	}
}

// parseISO8601Duration parses an ISO 8601 duration string and returns the duration in seconds.
func parseISO8601Duration(durationStr string) (int64, error) {
	// Example implementation - you'll need a proper parser for ISO 8601 durations
	duration, err := durationLib.Parse(durationStr)
	if err != nil {
		return 0, err
	}
	return int64(duration.ToTimeDuration().Seconds()), nil
}

func (k *Keeper) SetVestingDataInMemory(address string, data VestingData) {
	// fmt.Printf("Setting VestingData in memory for address: %s\n", address)
	// fmt.Printf("Data: %+v\n", data)
	k.InMemoryVestingData.VestingAccounts[address] = data
}

func (k *Keeper) GetVestingDataInMemory(address string) (VestingData, bool) {
	//fmt.Println("Locking k.mu")
	k.mu.Lock()
	defer func() {
		//fmt.Println("Unlocking k.mu")
		k.mu.Unlock()
	}()

	data, found := k.InMemoryVestingData.VestingAccounts[address]
	return data, found
}

func (k *Keeper) LogInMemoryVestingData() {
	fmt.Println("Logging InMemoryVestingData:")
	for key, data := range k.InMemoryVestingData.VestingAccounts {
		fmt.Printf("Key: %s, Value: %+v\n", key, data)
	}
}

func (k *Keeper) HasProcessedAddress(ctx sdk.Context, address sdk.AccAddress) bool {

	//k.LogInMemoryVestingData()

	// if len(k.InMemoryVestingData.VestingAccounts) == 0 {
	// 	fmt.Println("InMemoryVestingData is empty")
	// } else {
	// 	fmt.Println("InMemoryVestingData is not empty")
	// }
	data, found := k.InMemoryVestingData.VestingAccounts[address.String()]
	//fmt.Println("After Checking if address has been processed:\n", address)

	return found && data.Processed
}

func (k *Keeper) DeleteVestingDataInMemory(address string) {
	//fmt.Println("Locking k.mu")
	k.mu.Lock()
	defer func() {
		//fmt.Println("Unlocking k.mu")
		k.mu.Unlock()
	}()
	delete(k.InMemoryVestingData.VestingAccounts, address)
}

func ConvertStringToAcc(address string) (sdk.AccAddress, error) {
	//fmt.Println("Converting address:", address)
	return sdk.AccAddressFromBech32(address)
}

// USED FOR DEBUGGING TO CLEAR THE VESTING DATA STORE
// TODO: REMOVE FOR MAINNET
// func (k Keeper) ClearVestingDataStore(ctx sdk.Context) {
// 	store := ctx.KVStore(k.storeKey)
// 	iterator := sdk.KVStorePrefixIterator(store, ugdtypes.VestingDataKey)
// 	for ; iterator.Valid(); iterator.Next() {
// 		store.Delete(iterator.Key())
// 	}
// }
