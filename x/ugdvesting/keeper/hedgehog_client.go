package keeper

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	ugdtypes "github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/x/ugdvesting/types"
)

type VestingData struct {
	Address   string `json:"address"`
	Amount    int64  `json:"amount"`
	Start     string `json:"start"`
	Duration  string `json:"duration"`
	Parts     int    `json:"parts"`
	Block     int64  `json:"block"`
	Percent   int    `json:"percent"`
	Processed bool
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

func (k Keeper) SetProcessedAddress(ctx sdk.Context, address sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	key := append(ugdtypes.VestingDataKey, address.Bytes()...)
	store.Set(key, []byte("processed"))
}

func (k Keeper) HasProcessedAddress(ctx sdk.Context, address sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	key := append(ugdtypes.VestingDataKey, address.Bytes()...)
	return store.Has(key)
}

func (k Keeper) ProcessPendingVesting(ctx sdk.Context) {
	k.mu.Lock()
	defer k.mu.Unlock() // Using defer to ensure the mutex is always unlocked
	currentHeight := ctx.BlockHeight()
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, ugdtypes.VestingDataKey)
	defer iterator.Close()
	fmt.Println("=====================================")
	fmt.Println("=Processing pending vesting accounts=")
	fmt.Println("=====================================")
	for ; iterator.Valid(); iterator.Next() {
		var data ugdtypes.VestingData
		err := proto.Unmarshal(iterator.Value(), &data)
		if err != nil {
			fmt.Println("Error unmarshalling data:", err)
			continue
		}

		addr, err := sdk.AccAddressFromBech32(data.Address)
		if err != nil {
			continue
		}

		// Check if the block height matches and the account hasn't been processed
		if data.Block == currentHeight && !data.Processed {
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

					// TODO: If we want to DELAY the vesting, we can set the startTime to a future time
					// currently it's set to the current block time when the vesting account is created
					startTime := ctx.BlockTime().Unix()

					// Calculate the TGE amount
					tgeAmount := sdk.Coins{}
					for _, coin := range currentBalances {
						amount := coin.Amount.Mul(sdk.NewInt(int64(data.Percent))).Quo(sdk.NewInt(100)) // 3% of total balance
						tgeAmount = append(tgeAmount, sdk.NewCoin(coin.Denom, amount))
					}

					// Calculate the regular vesting amount per period
					amountPerPeriod := sdk.Coins{}
					for _, coin := range currentBalances {
						// Subtract the TGE amount from the total balance before calculating the regular vesting amount
						remainingAmount := coin.Amount.Sub(tgeAmount.AmountOf(coin.Denom))
						fmt.Println("Remaining amount:", remainingAmount)
						amount := remainingAmount.Quo(sdk.NewInt(int64(data.Parts - data.Cliff)))
						fmt.Println("Amount minus TGE and Cliff:", amount)
						amountPerPeriod = append(amountPerPeriod, sdk.NewCoin(coin.Denom, amount))
						fmt.Println("Amount per period:", amountPerPeriod)
					}

					// Create the vesting periods
					periods := vestingtypes.Periods{}
					//const periodTime = 2592000 // This is roughly the number of seconds in a month
					const periodTime = 60 // TESTING

					// TGE Period
					periods = append(periods, vestingtypes.Period{
						Length: periodTime, // 30 days in seconds
						Amount: tgeAmount,
					})

					// Adjusted Length for the first vesting period after TGE to account for the cliff
					// Adjusted Length for the first vesting period after TGE to account for the cliff
					firstVestingLength := periodTime * (1 + int(data.Cliff))
					periods = append(periods, vestingtypes.Period{
						Length: int64(firstVestingLength), // Cast to int64
						Amount: amountPerPeriod,
					})

					// Regular Vesting Periods after the first adjusted period
					for i := 2; i < int(data.Parts); i++ {
						periods = append(periods, vestingtypes.Period{
							Length: periodTime, // 30 days in seconds
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

					vestingAcc := vestingtypes.NewPeriodicVestingAccount(baseAccount, currentBalances, startTime, periods)

					k.SetAccount(ctx, vestingAcc)
					k.SetProcessedAddress(ctx, addr)
					fmt.Println("Converted address to PeriodicVestingAccount:", addr)
					// Mark the data as processed
					data.Processed = true
					bz, err := proto.Marshal(&data)
					if err != nil {
						fmt.Println("Error marshalling data:", err)
						continue
					}

					store.Set(iterator.Key(), bz)
					fmt.Println("Processed vesting data:", periods)
				}
			}
		}
	}
}

func (k Keeper) ProcessVestingAccounts(ctx sdk.Context) {
	k.mu.Lock()
	defer k.mu.Unlock()
	base := "http://82.208.23.218:5000"
	hedgehogUrl := base + "/mockdata" // testing mock data endpoint
	// base := viper.GetString("hedgehog.hedgehog_url")
	// hedgehogUrl := base + "/gridspork/vesting-storage"
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	response, err := client.Get(hedgehogUrl)

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

	vestings := make(map[string]VestingData)
	for key, vesting := range res.Data.VestingAddresses {
		address := strings.TrimPrefix(key, "Address(wif=")
		address = strings.TrimSuffix(address, ")")

		// Convert ISO 8601 duration string to Go's time.Duration
		goDuration, err := convertISODurationToGoDuration(vesting.Duration)
		if err != nil {
			fmt.Println("Error converting ISO duration:", err)
			continue
		}

		layout := "2006-01-02T15:04:05Z" // This is the standard format for RFC3339
		startTime, err := time.Parse(layout, vesting.Start)
		if err != nil {
			fmt.Println("Error parsing start time:", err)
			continue
		}

		vestings[address] = VestingData{
			Address:  address,
			Amount:   vesting.Amount,
			Start:    startTime.Format(time.RFC3339),
			Duration: goDuration.String(),
			Parts:    vesting.Parts,
			Block:    vesting.Block,
			Percent:  vesting.Percent,
		}

	}

	for addrStr, vestingData := range vestings {
		addr, err := ConvertStringToAcc(addrStr)
		if err != nil {
			fmt.Println("Error converting address:", err)
			continue
		}

		if k.HasProcessedAddress(ctx, addr) {
			fmt.Println("Address already processed:", addr)
			continue
		}
		startTime, err := time.Parse(time.RFC3339, vestingData.Start)
		if err != nil {
			fmt.Println("Error parsing start time:", err)
			continue
		}
		startInt64 := startTime.Unix()
		duration, err := time.ParseDuration(vestingData.Duration)
		if err != nil {
			fmt.Println("Error parsing duration:", err)
			continue
		}
		durationInt64 := duration.Nanoseconds()
		ugdVestingData := &ugdtypes.VestingData{
			Address:   vestingData.Address,
			Amount:    vestingData.Amount,
			Start:     startInt64,
			Duration:  durationInt64,
			Parts:     int32(vestingData.Parts), // Assuming the Parts in ugdtypes.VestingData is int32
			Block:     vestingData.Block,
			Percent:   int32(vestingData.Percent), // Assuming the Percent in ugdtypes.VestingData is int32
			Processed: vestingData.Processed,
		}
		fmt.Println("vestingData set:", vestingData)
		// Store the vesting data for processing in ProcessPendingVesting
		k.SetVestingData(ctx, addr, ugdVestingData)
	}
}

func (k Keeper) SetVestingData(ctx sdk.Context, address sdk.AccAddress, data *ugdtypes.VestingData) {
	store := ctx.KVStore(k.storeKey)
	key := append(ugdtypes.VestingDataKey, address.Bytes()...) // Assuming VestingDataKey is a prefix for vesting data

	// Marshal data to bytes
	b, err := proto.Marshal(data)
	if err != nil {
		// Handle error, maybe log it or return
		fmt.Println("Error marshaling vesting data:", err)
		return
	}

	store.Set(key, b)
}

func (k Keeper) GetVestingData(ctx sdk.Context, address sdk.AccAddress) (VestingData, bool) {
	store := ctx.KVStore(k.storeKey)
	key := append(ugdtypes.VestingDataKey, address.Bytes()...)

	b := store.Get(key)
	if b == nil {
		return VestingData{}, false
	}

	var data VestingData
	err := json.Unmarshal(b, &data)
	if err != nil {
		// Handle error, maybe log it or return
		fmt.Println("Error unmarshaling vesting data:", err)
		return VestingData{}, false
	}

	return data, true
}

// USED FOR DEBUGGING TO CLEAR THE VESTING DATA STORE
// TODO: REMOVE FOR MAINNET
func (k Keeper) ClearVestingDataStore(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, ugdtypes.VestingDataKey)
	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}

func ConvertStringToAcc(address string) (sdk.AccAddress, error) {
	fmt.Println("Converting address:", address)
	return sdk.AccAddressFromBech32(address)
}

func convertISODurationToGoDuration(isoDuration string) (time.Duration, error) {
	re := regexp.MustCompile(`^PT(\d+H)?(\d+M)?(\d+S)?$`)
	matches := re.FindStringSubmatch(isoDuration)
	if matches == nil {
		return 0, fmt.Errorf("invalid ISO 8601 duration format")
	}

	var duration time.Duration
	if matches[1] != "" {
		hours, _ := strconv.Atoi(matches[1][:len(matches[1])-1])
		duration += time.Duration(hours) * time.Hour
	}
	if matches[2] != "" {
		minutes, _ := strconv.Atoi(matches[2][:len(matches[2])-1])
		duration += time.Duration(minutes) * time.Minute
	}
	if matches[3] != "" {
		seconds, _ := strconv.Atoi(matches[3][:len(matches[3])-1])
		duration += time.Duration(seconds) * time.Second
	}
	return duration, nil
}
