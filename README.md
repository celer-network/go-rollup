# Go implementation of the Celer Optimistic Rollup

See the [contract](https://github.com/plasma-group/pigi/tree/master/packages/unipig)
repo for an overview of the Celer rollup architecture.

## Interacting with the sidechain testnet on Ropsten

1. Make sure you have `go` and `geth` installed.
2. Prepare an Ethereum keystore file with an empty password. Eg. running `geth account new`.
3. Join our Discord and ping us to obtain some Ropsten ETH and MOON tokens.
4. `git clone https://github.com/celer-network/go-rollup`.
5. `go run manual/transfertoken/main.go -ks <path-to-keystore-file> -recipient <recipient-address>`

## A few more details

1. Deposit MOON into the DepositWithdrawManager contract on Ropsten
   ([Example](https://github.com/celer-network/go-rollup/blob/5ae956cadfb852163bd208d2a33156140c994461/test/manual/depositwithdraw/main.go#L161)). The rollup aggregator will relay the deposit and mint
   corresponding amount of tokens on the sidechain.
2. (Optional) Register on the [AccountRegistry](https://github.com/celer-network/rollup-contracts/blob/8a1d735cb4af3aa557d106701a73e65ff7a22f00/contracts/mainchain/AccountRegistry.sol#L12) to ensure
   rollup security for account-to-account transfers.
3. Use `http://54.186.171.203:8547` as the JSON-RPC endpoint for the sidechain.
4. Transfer to contracts and external accounts are similar to a regular ERC-20 transfer, with the
   exception that an additional signature needs to be supplied ([Example](https://github.com/celer-network/go-rollup/blob/5ae956cadfb852163bd208d2a33156140c994461/test/token_mapper.go#L153)).
5. The rollup aggregator will gather the transfers in rollup blocks and push to Ropsten.
