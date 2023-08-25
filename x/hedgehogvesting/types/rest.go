package types

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/spf13/viper"
)

func HegdehogRequestGetVestingByAddr(addr string) *Vesting {
	hedgehogUrl := viper.GetString("hedgehog.hedgehog_url") + "/gridspork/vesting-storage/"
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	resp, err := client.Get(hedgehogUrl + addr)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
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
