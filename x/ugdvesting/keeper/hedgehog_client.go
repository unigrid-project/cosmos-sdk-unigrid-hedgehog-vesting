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
	"cosmossdk.io/math"
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

					startTime, err := time.Parse(time.RFC3339, data.Start)
					if err != nil {
						fmt.Println("Error parsing start time:", err)
						continue
					}

					startTimeUnix := startTime.Unix()

					// Calculate TGE amount based on data.Amount
					tgeAmount := sdk.Coins{}
					if data.Percent == 0 {
						fmt.Println("TGE percent is zero, adjusting start time")
						vestingDuration, err := parseISO8601Duration(data.Duration)
						if err != nil {
							fmt.Println("Error parsing vesting duration:", err)
							continue
						}
						startTimeUnix += int64(vestingDuration)
					} else {
						for _, coin := range currentBalances {
							if coin.Denom == "uugd" {
								amount := coin.Amount.Mul(math.NewInt(int64(data.Percent))).Quo(math.NewInt(100))
								tgeAmount = append(tgeAmount, sdk.NewCoin(coin.Denom, amount))
							}
						}
					}

					// Calculate remaining amount after TGE based on data.Amount
					remainingAmount := sdk.Coins{}
					for _, coin := range currentBalances {
						if coin.Denom == "uugd" {
							remaining := coin.Amount.Sub(tgeAmount.AmountOf(coin.Denom))
							remainingAmount = append(remainingAmount, sdk.NewCoin(coin.Denom, remaining))
						}
					}

					// Create vesting periods
					periods := vestingtypes.Periods{}
					vestingDuration, err := parseISO8601Duration(data.Duration)
					if err != nil {
						fmt.Println("Error parsing vesting duration:", err)
						continue
					}
					goDurationStr := strconv.FormatInt(vestingDuration, 10) + "s"
					periodTime, _ := time.ParseDuration(goDurationStr)

					// Add TGE period
					if data.Percent > 0 {
						periods = append(periods, vestingtypes.Period{
							Length: int64(periodTime.Seconds()),
							Amount: tgeAmount,
						})
					}

					if data.Parts == 0 {
						fmt.Println("Parts cannot be zero")
						continue
					}

					// Calculate the amount per part
					amountPerPart := sdk.Coins{}
					for _, coin := range remainingAmount {
						amountPerPart = append(amountPerPart, sdk.NewCoin(coin.Denom, coin.Amount.Quo(math.NewInt(int64(data.Parts)))))
					}

					// If cliff is zero, add all parts directly
					if data.Cliff == 0 {
						totalDistributed := sdk.NewCoins()
						for i := 0; i < int(data.Parts); i++ {
							periods = append(periods, vestingtypes.Period{
								Length: int64(periodTime.Seconds()),
								Amount: amountPerPart,
							})
							totalDistributed = totalDistributed.Add(amountPerPart...)
						}

						// Calculate the remaining difference to match the original balance
						difference := sdk.NewCoins()
						for _, coin := range remainingAmount {
							totalDistributedCoin := totalDistributed.AmountOf(coin.Denom)
							remaining := coin.Amount.Sub(totalDistributedCoin)
							difference = difference.Add(sdk.NewCoin(coin.Denom, remaining))
						}

						// Add the remaining difference to the final vesting period
						if len(periods) > 0 {
							periods[len(periods)-1].Amount = periods[len(periods)-1].Amount.Add(difference...)
						} else {
							periods = append(periods, vestingtypes.Period{
								Length: int64(periodTime.Seconds()),
								Amount: difference,
							})
						}
					} else {
						// Calculate the amount per cliff period if cliff is not zero
						rampUpAmountPerCliffPeriod := sdk.Coins{}
						for _, coin := range amountPerPart {
							rampUpAmountPerCliffPeriod = append(rampUpAmountPerCliffPeriod, sdk.NewCoin(coin.Denom, coin.Amount.Quo(math.NewInt(int64(data.Cliff)))))
						}

						// Add cliff periods
						for i := 0; i < int(data.Cliff); i++ {
							periods = append(periods, vestingtypes.Period{
								Length: int64(periodTime.Seconds()),
								Amount: rampUpAmountPerCliffPeriod,
							})
						}

						// Subtract one part from remainingAmount to account for cliff periods
						for i, coin := range remainingAmount {
							remainingAmount[i].Amount = coin.Amount.Sub(amountPerPart.AmountOf(coin.Denom))
						}

						// Add remaining vesting periods
						totalDistributed := sdk.NewCoins()
						for i := 0; i < int(data.Parts-1); i++ {
							periods = append(periods, vestingtypes.Period{
								Length: int64(periodTime.Seconds()),
								Amount: amountPerPart,
							})
							totalDistributed = totalDistributed.Add(amountPerPart...)
						}

						// Calculate the remaining difference to match the original balance
						difference := sdk.NewCoins()
						for _, coin := range remainingAmount {
							totalDistributedCoin := totalDistributed.AmountOf(coin.Denom)
							remaining := coin.Amount.Sub(totalDistributedCoin)
							difference = difference.Add(sdk.NewCoin(coin.Denom, remaining))
						}

						// Add the remaining difference to the final vesting period
						if len(periods) > 0 {
							periods[len(periods)-1].Amount = periods[len(periods)-1].Amount.Add(difference...)
						} else {
							periods = append(periods, vestingtypes.Period{
								Length: int64(periodTime.Seconds()),
								Amount: difference,
							})
						}
					}

					// Adjust the final period to handle any remaining small discrepancies
					totalAmount := sdk.NewCoins()
					for _, period := range periods {
						totalAmount = totalAmount.Add(period.Amount...)
					}

					finalPeriodIndex := len(periods) - 1
					for _, coin := range currentBalances {
						if coin.Denom == "uugd" {
							remaining := coin.Amount.Sub(totalAmount.AmountOf(coin.Denom))
							if !remaining.IsZero() {
								periods[finalPeriodIndex].Amount = periods[finalPeriodIndex].Amount.Add(sdk.NewCoin(coin.Denom, remaining))
							}
						}
					}

					// Log intermediate values for debugging
					fmt.Println("TGE Amount:", tgeAmount)
					fmt.Println("Remaining Amount after TGE:", remainingAmount)
					fmt.Println("Amount Per Part:", amountPerPart)

					// Calculate the sum of all periods
					totalAmount = sdk.NewCoins()
					for _, period := range periods {
						totalAmount = totalAmount.Add(period.Amount...)
					}

					// Compare the sum with currentBalances
					if !coinsEqual(totalAmount, currentBalances) {
						fmt.Printf("Mismatch! Original: %s, Calculated: %s\n", currentBalances, totalAmount)
						continue
					} else {
						fmt.Printf("Match! Original: %s, Calculated: %s\n", currentBalances, totalAmount)
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

					vestingAcc, err := vestingtypes.NewPeriodicVestingAccount(baseAccount, currentBalances, startTimeUnix, periods)
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

// coinsEqual compares two sets of sdk.Coins for equality
func coinsEqual(a, b sdk.Coins) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].IsEqual(b[i]) {
			return false
		}
	}
	return true
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
