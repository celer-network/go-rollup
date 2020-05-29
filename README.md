# Go implementation of the Celer Optimistic Rollup

See the [contract](https://github.com/plasma-group/pigi/tree/master/packages/unipig)
repo for an overview of the Celer rollup architecture.

## Interacting with the sidechain testnet on Ropsten

1. Contact us to obtain some MOON tokens.
2. Deposit MOON into the DepositWithdrawManager contract on Ropsten
   ([Example](https://github.com/celer-network/go-rollup/blob/5ae956cadfb852163bd208d2a33156140c994461/test/manual/depositwithdraw/main.go#L161)). The rollup aggregator will relay the deposit and mint
   corresponding amount of tokens on the sidechain.
3. (Optional) Register on the [AccountRegistry](https://github.com/celer-network/rollup-contracts/blob/8a1d735cb4af3aa557d106701a73e65ff7a22f00/contracts/mainchain/AccountRegistry.sol#L12) to ensure
   rollup security for account-to-account transfers.
4. Use `http://54.186.171.203:8547` as the JSON-RPC endpoint for the sidechain.
5. Transfer to contracts and external accounts are similar to a regular ERC-20 transfer, with the
   exception that an additional signature needs to be supplied ([Example](https://github.com/celer-network/go-rollup/blob/5ae956cadfb852163bd208d2a33156140c994461/test/token_mapper.go#L153)).
6. The rollup aggregator will gather the transfers in rollup blocks and push to Ropsten.
