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

func (k *Keeper) SetProcessedAddress(ctx sdk.Context, address sdk.AccAddress) {
	k.mu.Lock()
	defer k.mu.Unlock()
	// Assuming you have a field to mark processed in VestingData
	if data, found := k.InMemoryVestingData.VestingAccounts[address.String()]; found {
		data.Processed = true
		k.InMemoryVestingData.VestingAccounts[address.String()] = data
	}
}

func (k *Keeper) HasProcessedAddress(ctx sdk.Context, address sdk.AccAddress) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	data, found := k.InMemoryVestingData.VestingAccounts[address.String()]
	return found && data.Processed
}

func (k *Keeper) ProcessPendingVesting(ctx sdk.Context) {
	k.mu.Lock()
	defer k.mu.Unlock() // Ensure the mutex is always unlocked

	currentHeight := ctx.BlockHeight()
	fmt.Println("=====================================")
	fmt.Println("=Processing pending vesting accounts=")
	fmt.Println("=====================================")

	for address, data := range k.InMemoryVestingData.VestingAccounts {
		// Check if the block height matches and the account hasn't been processed
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

					amountPerPeriod := sdk.Coins{}
					for _, coin := range currentBalances {
						remainingAmount := coin.Amount.Sub(tgeAmount.AmountOf(coin.Denom))
						vestingPeriods := int(data.Parts) - int(data.Cliff) - 1
						amount := remainingAmount.Quo(math.NewInt(int64(vestingPeriods)))
						amountPerPeriod = append(amountPerPeriod, sdk.NewCoin(coin.Denom, amount))
					}

					periods := vestingtypes.Periods{}
					// Parse Duration from ISO 8601 format to seconds
					vestingDuration, err := parseISO8601Duration(data.Duration)
					if err != nil {
						fmt.Println("Error parsing vesting duration:", err)
						continue
					}
					// Convert the duration in seconds to a duration string
					goDurationStr := strconv.FormatInt(vestingDuration, 10) + "s"

					periodTime, _ := time.ParseDuration(goDurationStr) // Convert string to duration

					periods = append(periods, vestingtypes.Period{
						Length: int64(periodTime.Seconds()), // Convert duration to seconds
						Amount: tgeAmount,
					})

					zeroAmount := sdk.NewCoin("ugd", math.NewInt(0))
					for i := 1; i <= int(data.Cliff); i++ {
						periods = append(periods, vestingtypes.Period{
							Length: int64(periodTime.Seconds()),
							Amount: sdk.Coins{zeroAmount},
						})
					}

					for i := int(data.Cliff) + 1; i < int(data.Parts); i++ {
						periods = append(periods, vestingtypes.Period{
							Length: int64(periodTime.Seconds()),
							Amount: amountPerPeriod,
						})
					}

					var pubKeyAny *codectypes.Any
					if baseAcc.GetPubKey() != nil {
						var err error
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
					}

					k.SetAccount(ctx, vestingAcc)
					// Mark the data as processed
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
	defer k.mu.Unlock()

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

	for key, vesting := range res.Data.VestingAddresses {
		address := strings.TrimPrefix(key, "Address(wif=")
		address = strings.TrimSuffix(address, ")")

		addr, err := ConvertStringToAcc(address)
		if err != nil {
			fmt.Println("Error converting address:", err)
			continue
		}

		if k.HasProcessedAddress(ctx, addr) {
			fmt.Println("Address already processed:", addr)
			continue
		}

		// Store the parsed data in memory
		k.InMemoryVestingData.VestingAccounts[key] = VestingData{
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
		fmt.Println("In-memory vesting data set for address:", address)
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
	k.mu.Lock()
	defer k.mu.Unlock()
	k.InMemoryVestingData.VestingAccounts[address] = data
}

func (k *Keeper) GetVestingDataInMemory(address string) (VestingData, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	data, found := k.InMemoryVestingData.VestingAccounts[address]
	return data, found
}

func (k *Keeper) DeleteVestingDataInMemory(address string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.InMemoryVestingData.VestingAccounts, address)
}

func ConvertStringToAcc(address string) (sdk.AccAddress, error) {
	fmt.Println("Converting address:", address)
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
