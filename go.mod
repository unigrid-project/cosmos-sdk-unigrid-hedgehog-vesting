module github.com/timnhanta/ugdvesting

go 1.19

require (
	cosmossdk.io/math v1.0.0
	cosmossdk.io/simapp v0.0.0-20230426205644-8f6a94cd1f9f
	github.com/cometbft/cometbft v0.37.1
	github.com/cometbft/cometbft-db v0.7.0
	github.com/cosmos/cosmos-sdk v0.47.2
	github.com/cosmos/ibc-go/v7 v7.0.0-20230427100746-a25f0d421c32
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.15.2
	github.com/spf13/cast v1.5.0
	github.com/spf13/cobra v1.7.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.2
)

require (
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.1.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/regen-network/cosmos-proto v0.3.1 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230223222841-637eb2293923 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/confio/ics23/go => github.com/cosmos/cosmos-sdk/ics23/go v0.8.0
