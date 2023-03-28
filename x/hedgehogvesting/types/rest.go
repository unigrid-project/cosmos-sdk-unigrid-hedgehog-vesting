package types

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

const (
	HedgehogBaseUrlTest = "https://localhost:52884/gridspork/vesting-storage/"
)

func HegdehogRequestGetVestingByAddr(addr string) *Vesting {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	resp, err := client.Get(HedgehogBaseUrlTest + addr)
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
