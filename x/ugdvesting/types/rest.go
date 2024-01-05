package types

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/spf13/viper"
	"github.com/unigrid-project/cosmos-common/common/httpclient"
)

func HegdehogRequestGetVestingByAddr(addr string) *Vesting {
	hedgehogUrl := viper.GetString("hedgehog.hedgehog_url") + "/gridspork/vesting-storage/"

	resp, err := httpclient.Client.Get(hedgehogUrl + addr)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if len(body) == 0 {
		return nil
	}

	var vesting *Vesting

	err = json.Unmarshal([]byte(body), &vesting)
	if err != nil {
		panic(err)
	}

	return vesting
}

type Mints struct {
	Mints map[string]int
}

type HedgehogData struct {
	Data         Mints `json:"data"`
	PreviousData Mints `json:"previousData"`
}

func HegdehogCheckIfInMintingList(addr string) bool {
	hedgehogUrl := viper.GetString("hedgehog.hedgehog_url") + "/gridspork/mint-storage/"

	resp, err := httpclient.Client.Get(hedgehogUrl + addr)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if len(body) == 0 {
		return false
	}

	var data *HedgehogData

	err = json.Unmarshal([]byte(body), &data)
	if err != nil {
		panic(err)
	}

	for key := range data.Data.Mints {
		if strings.Contains(key, addr) {
			return true
		}
	}

	return false
}
