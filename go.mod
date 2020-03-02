module github.com/celer-network/sidechain-rollup-aggregator

go 1.13

require (
	github.com/aergoio/aergo-lib v1.0.1
	github.com/celer-network/cChannel-eth-go v0.12.6
	github.com/celer-network/goCeler v0.16.15
	github.com/celer-network/goutils v0.1.3
	github.com/celer-network/sidechain-contracts v0.0.0-20200227115541-c115f8123ca0
	github.com/ethereum/go-ethereum v1.9.11
	github.com/minio/sha256-simd v0.1.1
	github.com/rs/zerolog v1.18.0
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/viper v1.5.0
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

replace github.com/celer-network/sidechain-contracts => ../sidechain-contracts
