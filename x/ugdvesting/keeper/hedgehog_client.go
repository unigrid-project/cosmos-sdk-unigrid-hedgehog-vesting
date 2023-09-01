package keeper

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/x/ugdvesting/types"
)

type VestingData struct {
	Address  string
	Amount   int
	Start    time.Time
	Duration time.Duration
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

type VestingCache struct {
	stop     chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
	vestings map[string]VestingData
	first    bool
}

var (
	c    = &VestingCache{}
	once sync.Once
)

func GetCache() *VestingCache {
	fmt.Println("Getting vesting cache")
	once.Do(func() {
		c = NewCache()
	})
	return c
}

func NewCache() *VestingCache {
	vc := &VestingCache{
		vestings: make(map[string]VestingData),
		stop:     make(chan struct{}),
		first:    true,
	}

	return vc
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

func (vc *VestingCache) CallHedgehog(serverUrl string, ctx sdk.Context, k Keeper) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	response, err := client.Get(serverUrl)

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
	for key, vesting := range res.Data.VestingAddresses {
		address := strings.TrimPrefix(key, "Address(wif=")
		address = strings.TrimSuffix(address, ")")
		vc.vestings[address] = VestingData{
			Address:  address,
			Amount:   vesting.Amount,
			Start:    vesting.Start,    // Directly assign
			Duration: vesting.Duration, // Directly assign
			Parts:    vesting.Parts,
		}
	}

	// Loop through all addresses in the vesting data
	for addrStr, _ := range vc.vestings {
		addr, err := ConvertStringToAcc(addrStr)
		if err != nil {
			fmt.Println("Error converting address:", err)
			continue
		}

		// Check if the address has been processed
		if k.HasProcessedAddress(ctx, addr) {
			continue
		}

		account := k.GetAccount(ctx, addr)
		if account == nil {
			continue
		}

		// Check if the account is already a PeriodicVestingAccount
		if _, ok := account.(*vestingtypes.PeriodicVestingAccount); !ok {
			// Ensure the account exists and has a balance
			currentBalances := k.GetAllBalances(ctx, addr)
			if currentBalances.IsZero() {
				continue
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

			pubKeyAny, err := codectypes.NewAnyWithValue(account.GetPubKey())
			if err != nil {
				// Handle the error
				fmt.Println("Error packing public key into Any:", err)
				return
			}

			baseAccount := &authtypes.BaseAccount{
				Address:       account.GetAddress().String(),
				PubKey:        pubKeyAny,
				AccountNumber: account.GetAccountNumber(),
				Sequence:      account.GetSequence(),
			}

			// Create the PeriodicVestingAccount
			vestingAcc := vestingtypes.NewPeriodicVestingAccount(baseAccount, currentBalances, startTime, periods)
			k.SetAccount(ctx, vestingAcc)
		}

		// Mark the address as processed
		k.SetProcessedAddress(ctx, addr)
	}

	// Print vesting data for debugging
	for _, v := range vc.vestings {
		fmt.Println(v)
	}
}

func ConvertStringToAcc(address string) (sdk.AccAddress, error) {
	//sdk.GetConfig().SetBech32PrefixForAccount("unigrid", "unigrid")
	//s := strings.TrimPrefix(address, "unigrid")
	return sdk.AccAddressFromBech32(address)
}