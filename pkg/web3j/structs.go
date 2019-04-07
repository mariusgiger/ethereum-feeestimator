package web3j

import (
	"math/big"
)

type Tx struct {
	Miner    string
	Hash     string
	GasPrice *big.Int
}

type Probability struct {
	GasPrice    int64
	Probability float64
}

type minerData struct {
	Miner                 string
	Blocks                int
	MinGasPrice           int64
	LowPercentileGasPrice int64
}
