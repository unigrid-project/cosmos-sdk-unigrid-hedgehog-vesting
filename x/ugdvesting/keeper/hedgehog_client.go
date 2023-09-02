package keeper

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/spf13/viper"
	"github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/x/ugdvesting/types"
)

type VestingData struct {
	Address  string
	Amount   int
	Start    time.Time
	Duration string
	Parts    int
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
	key := append(types.VestingKey, address.Bytes()...)
	store.Set(key, []byte("processed"))
}

func (k Keeper) HasProcessedAddress(ctx sdk.Context, address sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	key := append(types.VestingKey, address.Bytes()...)
	return store.Has(key)
}

func ProcessVestingAccounts(ctx sdk.Context, k Keeper) {
	hedgehogUrl := viper.GetString("hedgehog.hedgehog_url")

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

	// Check if the response is empty
	if response.ContentLength == 0 {
		fmt.Println("Received empty response from hedgehog server.")
		return
	}

	var res HedgehogData
	body, err1 := io.ReadAll(response.Body)

	if err1 != nil {
		fmt.Println(err1.Error())
		//report response error in log
		return
	}

	e := json.Unmarshal(body, &res)
	if e != nil {
		fmt.Println(e.Error())
		//report response error in log
		return
	}

	// Handle vesting data
	vestings := make(map[string]VestingData) // Local variable to store vesting data
	for key, vesting := range res.Data.VestingAddresses {
		address := strings.TrimPrefix(key, "Address(wif=")
		address = strings.TrimSuffix(address, ")")
		fmt.Println("Address from hedgehog vesting:", address)
		vestings[address] = VestingData{
			Address:  address,
			Amount:   vesting.Amount,
			Start:    vesting.Start,    // Directly assign
			Duration: vesting.Duration, // Directly assign
			Parts:    vesting.Parts,
		}
	}

	// Loop through all addresses in the vesting data
	for addrStr, _ := range vestings {
		addr, err := ConvertStringToAcc(addrStr)
		if err != nil {
			fmt.Println("Error converting address:", err)
			continue
		}

		// Check if the address has been processed
		if k.HasProcessedAddress(ctx, addr) {
			fmt.Println("Address already processed:", addr)
			continue
		}

		account := k.GetAccount(ctx, addr)
		if account == nil {
			fmt.Println("Account not found:", addr)
			continue
		}
		fmt.Println("Account found:", account)
		// Check if the account is already a PeriodicVestingAccount
		if _, ok := account.(*vestingtypes.PeriodicVestingAccount); !ok {
			if baseAcc, ok := account.(*vestingtypes.DelayedVestingAccount); ok {
				// Ensure the account exists and has a balance
				currentBalances := k.GetAllBalances(ctx, addr)
				if currentBalances.IsZero() {

					return
				}

				startTime := ctx.BlockTime().Unix() // Current block time as start time

				// Calculate the amount for each vesting period for each coin in currentBalances
				amountPerPeriod := sdk.Coins{}
				for _, coin := range currentBalances {
					amount := coin.Amount.Quo(sdk.NewInt(10))
					amountPerPeriod = append(amountPerPeriod, sdk.NewCoin(coin.Denom, amount))
				}

				// Create 10 vesting periods, each 1 minute apart
				periods := vestingtypes.Periods{}
				for i := 0; i < 10; i++ {
					period := vestingtypes.Period{
						Length: 60, // 60 seconds = 1 minute
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

				// Create the PeriodicVestingAccount
				vestingAcc := vestingtypes.NewPeriodicVestingAccount(baseAccount, currentBalances, startTime, periods)
				fmt.Println("Created PeriodicVestingAccount:", vestingAcc)
				k.SetAccount(ctx, vestingAcc)
			}
		}

		// Mark the address as processed
		k.SetProcessedAddress(ctx, addr)
	}

	// Print vesting data for debugging
	for _, v := range vestings {
		fmt.Println(v)
	}
}

func ConvertStringToAcc(address string) (sdk.AccAddress, error) {
	fmt.Println("Converting address:", address)
	return sdk.AccAddressFromBech32(address)
}
