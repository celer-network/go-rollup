# Go implementation of the Celer Optimistic Rollup

See the [contract](https://github.com/celer-network/rollup-contracts)
repo for an overview of the Celer rollup architecture.

## Examples for interacting with the sidechain testnet on Ropsten

1. Make sure you have `go`, `geth` and `make` installed.
2. Prepare an Ethereum keystore file with an **empty password**. Eg. run:

```shellscript
geth account new --lightkdf --keystore <path-to-keystore-folder>
```

3. We have mapped an ERC-20 token named MOON from Ropsten to the sidechain. Join our [Discord](https://discord.gg/uGx4fjQ)
   server. Ping us to obtain some MOON tokens and sidechain ETH for gas. You should also obtain some Ropsten ETH from places
   like the MetaMask [faucet](https://faucet.metamask.io).
4. Clone the repository and install the demo binary:

```shellscript
git clone https://github.com/celer-network/go-rollup
cd go-rollup
make install-demo
```

5. Deposit into the sidechain:

```shellscript
rollupdemo deposit --keystore <path-to-keystore-file> --amount <amount>
```

This deposits `<amount>` MOON tokens into the sidechain.

6. For an example of account-to-account transfer, run:

```shellscript
rollupdemo transfer-to-account --keystore <path-to-keystore-file> --recipient <recipient-address> --amount <amount>
```

This transfers `<amount>` MOON tokens from the sender to the recipient.

6. For an example of contract interaction, run:

```shellscript
rollupdemo transfer-to-contract --keystore <path-to-keystore-file> --amount <amount>
```

This deploys a dummy dApp
[contract](https://github.com/celer-network/rollup-contracts/blob/8a1d735cb4af3aa557d106701a73e65ff7a22f00/contracts/sidechain/DummyApp.sol)
on the sidechain and sends `<amount>` MOON tokens to it.

## A few more details

1. Deposit is sent to [DepositWithdrawManager](https://github.com/celer-network/rollup-contracts/blob/8a1d735cb4af3aa557d106701a73e65ff7a22f00/contracts/mainchain/DepositWithdrawManager.sol) contract on Ropsten. The rollup aggregator will
   relay the deposit and mint corresponding amount of tokens on the sidechain.
2. Optional rollup security for account-to-account transfers is ensured by registering on the [AccountRegistry](https://github.com/celer-network/rollup-contracts/blob/8a1d735cb4af3aa557d106701a73e65ff7a22f00/contracts/mainchain/AccountRegistry.sol#L12).
3. Transfer to contracts and external accounts are similar to a regular ERC-20 transfer, with the
   exception that an additional signature needs to be supplied ([Example](https://github.com/celer-network/go-rollup/blob/5ae956cadfb852163bd208d2a33156140c994461/test/token_mapper.go#L153)).
4. The rollup aggregator will pack the transfers into rollup blocks and commit to Ropsten.
