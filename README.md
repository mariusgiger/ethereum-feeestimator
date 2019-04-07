# Ethereum fee estimator

Research on predicting Ethereum transaction fees.

## Context

Blockchain-technologies such as Ethereum and other crypto currencies are currently entering a more mature stage. However, the user experience is still lacking in several areas. One crucial point regarding payments is the optimization of transaction fees. In many cases the basic functionality for sending transactions is covered by the concrete wallet implementations which often make use of suboptimal naïve approaches. In the paper linked to this repository methods to estimate the current Ethereum gas price are challenged and improvements are discussed.

## Estimating Fees

Unlike Bitcoin, which applies a construct called “unspent transaction output” (UTXO) [1], Ethereum is based on accounts [2]. Every account is holding the information about the balance, representing the amount of Ether (the currency of Ethereum) available to a user. When a user wants to send Ether, she has to determine how much she is willing to pay for the transaction (transaction fee) to be processed contemporary. This is done by specifying a certain gas price and a gas limit.

_Gas_ is used to protect the Ethereum network from abuse. A real-world analogy would be fuel. Every computational step (e.g. steps to validate a transaction) uses a certain amount of gas.

The _gas limit_ determines the maximum amount of gas a user is willing to pay for a transaction. This ensures that for instance programming errors in smart contracts do not lead to infinite loops and therefore infinite transaction costs but rather the transaction gets cancelled preemptively once the limit is reached. All unused gas is refunded to the sender at the end of a transaction. A standard transaction has a gas limit of 21000.

The _gas price_ determines the amount of pay per unit of gas. By increasing this value, a user can reduce the confirmation time for a transaction. This is because miners pick unconfirmed transactions from a pool and prioritize the ones with high fees.

The total cost of a transaction is hence defined as follows:

> fee=gasLimit\*gasPrice

The estimation of the current gas price is in most cases implemented by the wallet. If this estimation is done right, the transaction fees can be reduced significantly, in particular for high-frequency wallets. However, the problem of choosing the right parameters is not trivial since there are multiple factors accounting for the current gas price, such as:

- The amount of unconfirmed transactions in the current mempool
- The gas price attached to the transactions in the mempool
- The miner prioritization algorithm for selecting the transactions for a block
- The difficulty of simulating the estimation due to the complexity of the system

Since Ethereum is a smart-contract enabling blockchain (an example are ERC20 based tokens ), the users are additionally competing for computational resources through an auction based on the gas price [3]. Thus, also the computational effort for a transaction has an influence on the prioritization. At this point it is unclear if the computational effort (number of operations) and the gas price correlate linearly.

## Goals

In the related paper a method for estimating the gas price based on go-ethereum’s SuggestGasPrice method is evaluated and compared to the prediction model of Ethereum gas station. Since these are the two most widely used methods for estimating gas prices (ref).
The following questions should be answered in the paper:

- Which factors contribute to the current gas price?
- How can the transaction fee for sending Ethereum be determined considering the current mempool of pending transactions?

## Run it

```bash
dep ensure
# Bugfix for missing libsecp256k1/include/secp256k1.h (because dep does not allow cpp files)
go get "github.com/ethereum/go-ethereum"
cp -r "${GOPATH}/src/github.com/ethereum/go-ethereum/crypto/secp256k1/libsecp256k1" "vendor/github.com/ethereum/go-ethereum/crypto/secp256k1/"

go build -o ./output/estimator . && ./output/estimator express
```

## Generate pseudo code

```bash
brew install pandoc
pandoc Pseudo.mdc -o Pseudo.pdf
```

## References

[1] S. Nakamoto, “Bitcoin: A peer-to-peer electronic cash system.,” 2008.

[2] V. Buterin, “A NEXT GENERATION SMART CONTRACT & DECENTRALIZED APPLICATION PLATFORM.”

[3] C. Hoffreumon, N. Van Zeebroeck, C. Hoffreumon, and N. Van Zeebroeck, “Forecasting short-term transaction fees on a smart contracts platform,” 2018.
