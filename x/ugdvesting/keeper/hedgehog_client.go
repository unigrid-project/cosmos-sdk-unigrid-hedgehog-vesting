package keeper

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	ugdtypes "github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/x/ugdvesting/types"
)

type VestingData struct {
	Address   string        `json:"address"`
	Amount    int64         `json:"amount"`
	Start     time.Time     `json:"start"`
	Duration  time.Duration `json:"duration"`
	Parts     int           `json:"parts"`
	Block     int64         `json:"block"`
	Percent   int           `json:"percent"`
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
	key := append(ugdtypes.VestingKey, address.Bytes()...)
	store.Set(key, []byte("processed"))
}

func (k Keeper) HasProcessedAddress(ctx sdk.Context, address sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	key := append(ugdtypes.VestingKey, address.Bytes()...)
	return store.Has(key)
}

func (k Keeper) ProcessPendingVesting(ctx sdk.Context) {
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
						return
					}

					startTime := ctx.BlockTime().Unix()
					amountPerPeriod := sdk.Coins{}
					for _, coin := range currentBalances {
						amount := coin.Amount.Quo(sdk.NewInt(int64(data.Parts))) // Use the parts from data
						amountPerPeriod = append(amountPerPeriod, sdk.NewCoin(coin.Denom, amount))
					}

					periods := vestingtypes.Periods{}
					for i := 0; i < int(data.Parts); i++ { // Cast data.Parts to int
						period := vestingtypes.Period{
							Length: 60, // Adjust this if needed
							Amount: amountPerPeriod,
						}
						periods = append(periods, period)
					}

					var pubKeyAny *codectypes.Any
					if baseAcc.GetPubKey() != nil {
						var err error
						pubKeyAny, err = codectypes.NewAnyWithValue(baseAcc.GetPubKey())
						if err != nil {
							fmt.Println("Error packing public key into Any:", err)
							return
						}
					}

					baseAccount := &authtypes.BaseAccount{
						Address:       baseAcc.GetAddress().String(),
						PubKey:        pubKeyAny,
						AccountNumber: baseAcc.GetAccountNumber(),
						Sequence:      baseAcc.GetSequence(),
					}

					vestingAcc := vestingtypes.NewPeriodicVestingAccount(baseAccount, currentBalances, startTime, periods)

					k.mu.Lock()
					defer k.mu.Unlock() // Using defer to ensure the mutex is always unlocked

					k.SetAccount(ctx, vestingAcc)
					k.SetProcessedAddress(ctx, addr)
					fmt.Println("Converted address to PeriodicVestingAccount:", addr)
					// Mark the data as processed
					data.Processed = true
					bz, err := proto.Marshal(&data)
					if err != nil {
						fmt.Println("Error marshalling data:", err)
						return // or continue, depending on whether you want to skip the rest of the loop or exit the function
					}

					store.Set(iterator.Key(), bz)
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
		vestings[address] = VestingData{
			Address:  address,
			Amount:   vesting.Amount,
			Start:    vesting.Start,
			Duration: vesting.Duration,
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

		account := k.GetAccount(ctx, addr)
		if account == nil {
			fmt.Println("Account not found:", addr)
			continue
		}
		fmt.Println("vestingData set:", vestingData)
		// Store the vesting data for processing in ProcessPendingVesting
		k.SetVestingData(ctx, addr, vestingData)
	}
}

func (k Keeper) SetVestingData(ctx sdk.Context, address sdk.AccAddress, data VestingData) {
	store := ctx.KVStore(k.storeKey)
	key := append(ugdtypes.VestingDataKey, address.Bytes()...) // Assuming VestingDataKey is a prefix for vesting data

	// Marshal data to bytes
	b, err := json.Marshal(data)
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

func ConvertStringToAcc(address string) (sdk.AccAddress, error) {
	fmt.Println("Converting address:", address)
	return sdk.AccAddressFromBech32(address)
}
