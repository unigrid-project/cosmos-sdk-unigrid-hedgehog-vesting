package types_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/unigrid-project/cosmos-sdk-unigrid-hedgehog-vesting/x/ugdvesting/types"

	sdkmath "cosmossdk.io/math"
)

const (
	jsonString string = "{\"timeStamp\":\"2023-08-30T12:44:21.659Z\"," +
		"\"previousTimeStamp\":\"2023-08-30T12:43:31.055Z\"" +
		",\"flags\":0," +
		"\"type\":\"VESTING_STORAGE\"," +
		"\"data\":" +
		"{\"vestingAddresses\":" +
		"{\"Address(wif=unigrid1k3xsk7muy8738hteg94de6ynde0v0af9tgptx0)\":" +
		"{\"amount\":1000000,\"start\":\"2023-08-29T16:53:46Z\",\"duration\":\"PT3H\",\"parts\":5}," +
		"\"Address(wif=unigrid16vyxrtjarguun728vg0fuxs847sh5vls38cfss)\":{\"amount\":1000000,\"start\":\"2023-08-29T16:53:46Z\"," +
		"\"duration\":\"PT3H\",\"parts\":5}}}," +
		"\"previousData\":{\"vestingAddresses\":{\"Address(wif=unigrid1k3xsk7muy8738hteg94de6ynde0v0af9tgptx0)\"" +
		":{\"amount\":1000000,\"start\":\"2023-08-29T16:53:46Z\",\"duration\":\"PT3H\",\"parts\":5}}}," +
		"\"signature\":\"MIGIAkIBZxoWIkguoxE+VdrrkKzZ8W58bIz5WAy6uyYbRoz0qGwQw1ipTBObiOaUesNlYV34aVMo12F2XPlg1FFz0FryLX4CQgHg1szNaHDinSCQAgc3En9/msb94TfGyhh73nzKSPEpLE99fD8dxJO8LTuU3Ufl9X5zzg3mjJp0N4RHf3qwFAjLkg==\"}"
)

var (
	mux    *http.ServeMux
	server *httptest.Server
)

func serverSetup() func() {
	mux = http.NewServeMux()

	// priv, err := rsa.GenerateKey(rand.Reader, *rsaBits)
	priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	/*
	   hosts := strings.Split(*host, ",")
	   for _, h := range hosts {
	   	if ip := net.ParseIP(h); ip != nil {
	   		template.IPAddresses = append(template.IPAddresses, ip)
	   	} else {
	   		template.DNSNames = append(template.DNSNames, h)
	   	}
	   }
	   if *isCA {
	   	template.IsCA = true
	   	template.KeyUsage |= x509.KeyUsageCertSign
	   }
	*/

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)

	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	out := &bytes.Buffer{}
	out2 := &bytes.Buffer{}
	pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certKey := out.Bytes()
	fmt.Println(out.String())
	out.Reset()
	pem.Encode(out2, pemBlockForKey(priv))
	pubKey := out2.Bytes()
	fmt.Println(out2.String())

	cert, err := tls.X509KeyPair(certKey, pubKey)
	//tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Panic("bad server certs: ", err)
	}
	certs := []tls.Certificate{cert}

	server = httptest.NewUnstartedServer(mux)
	server.TLS = &tls.Config{Certificates: certs}
	//server.URL = "http://127.0.0.1:52884"
	//server. = "0.0.0.0:52884"
	server.StartTLS()

	return func() {
		server.Close()
	}
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

func TestParsVesting(t *testing.T) {
	teardown := serverSetup()
	//defer teardown()

	mux.HandleFunc("/gridspork/vesting-storage", func(w http.ResponseWriter, r *http.Request) {
		//r.RequestURI = "/gridspork/mint-storage"
		//r.Host = "https://127.0.0.1:52884"
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jsonString))
	})

	cache := types.NewCache()

	cache.UpdateVesting(server.URL)

	teardown()
}

func TestGetUnvestedAmount(t *testing.T) {
	// unvested should be 0, if vesting not started
	timeNow := time.Now()
	timeNow = timeNow.In(time.FixedZone("CET", 0))
	amount := int64(1000)
	duration := "PT10M"
	part := int64(10)
	timeStart := timeNow.Add(10 * time.Minute)
	formattedTimeStart := timeStart.UTC().Format("2006-01-02T15:04:05.999999Z")

	vesting := types.Vesting{
		Amount:   amount,
		Start:    formattedTimeStart,
		Duration: duration,
		Parts:    part,
	}
	unvested := types.GetUnvestedAmount(vesting)

	expected := big.NewInt(0)
	if unvested.BigInt().Cmp(expected) != 0 {
		t.Errorf("unvested = %v, expected = %v", unvested, expected)
	}

	// unvested should be 0, if vesting is done
	timeNow = time.Now()
	timeNow = timeNow.In(time.FixedZone("CET", 0))
	timeStart = timeNow.Add(-11 * time.Minute)
	formattedTimeStart = timeStart.UTC().Format("2006-01-02T15:04:05.999999Z")

	vesting = types.Vesting{
		Amount:   amount,
		Start:    formattedTimeStart,
		Duration: duration,
		Parts:    part,
	}
	unvested = types.GetUnvestedAmount(vesting)

	expected = big.NewInt(0)
	if unvested.BigInt().Cmp(expected) != 0 {
		t.Errorf("unvested = %v, expected = %v", unvested, expected)
	}

	// unvested should be 600, if vesting progress is 4/10
	timeNow = time.Now()
	timeNow = timeNow.In(time.FixedZone("CET", 0))
	timeStart = timeNow.Add(-4 * time.Minute)
	formattedTimeStart = timeStart.UTC().Format("2006-01-02T15:04:05.999999Z")

	vesting = types.Vesting{
		Amount:   amount,
		Start:    formattedTimeStart,
		Duration: duration,
		Parts:    part,
	}
	unvested = types.GetUnvestedAmount(vesting)

	expected = big.NewInt(600)
	if unvested.BigInt().Cmp(expected) != 0 {
		t.Errorf("unvested = %v, expected = %v", unvested, expected)
	}
}

func TestSdkIntToFloat(t *testing.T) {
	var expected big.Float
	var bigInt big.Int

	expected.SetPrec(256)
	expected.SetString("7000.707070707070707070")

	bigInt.SetString("7000707070707070707070", 10)
	result := types.SdkIntToFloat(sdkmath.NewIntFromBigInt(&bigInt), 256, math.Pow10(18))

	if result.Cmp(&expected) != 0 {
		t.Errorf("Unexpected response. Expected %+v, but got %+v", expected, result)
	}
}

func TestSdkIntToString(t *testing.T) {
	expected := "7000.707070707070707070"

	var bigInt big.Int
	bigInt.SetString("7000707070707070707070", 10)
	result := types.SdkIntToString(sdkmath.NewIntFromBigInt(&bigInt), 256, math.Pow10(18), 18)

	if result != expected {
		t.Errorf("Unexpected response. Expected %+v, but got %+v", expected, result)
	}

}

/*func TestUnmarshalJSON(t *testing.T) {
	var vesting types.Vesting
	var bigInt big.Int

	jsonStr := `{"amount":"7000.707070707070707070","start":"2023-03-14T18:41:20Z","duration":"PT168H29M58S","parts":7}`

	err := json.Unmarshal([]byte(jsonStr), &vesting)
	if err != nil {
		panic(err)
	}

	bigInt.SetString("7000707070707070707070", 10)

	expected := types.Vesting{
		Amount:   sdkmath.NewIntFromBigInt(&bigInt),
		Start:    "2023-03-14T18:41:20Z",
		Duration: "PT168H29M58S",
		Parts:    7,
	}

	if vesting.Amount.Neg().Equal(expected.Amount) &&
		vesting.Start != expected.Start &&
		vesting.Duration != expected.Duration &&
		vesting.Parts != expected.Parts {
		t.Errorf("Unexpected response. Expected %+v, but got %+v", expected, vesting)
	}
}*/
