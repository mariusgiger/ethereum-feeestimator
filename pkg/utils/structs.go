package utils

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// BlockHeader represents a simplified block header in the Ethereum blockchain.
type BlockHeader struct {
	Hash         common.Hash  `json:"hash"`
	Number       *hexutil.Big `json:"number"`
	GasLimit     *hexutil.Big `json:"gasLimit"`
	GasUsed      *hexutil.Big `json:"gasUsed"`
	Time         *hexutil.Big `json:"timestamp"`
	Transactions []string     `json:"transactions"`
}

// Block represents a block in the Ethereum blockchain.
type Block struct {
	ParentHash   common.Hash    `json:"parentHash"`
	Hash         common.Hash    `json:"hash"`
	Miner        common.Address `json:"miner"`
	Difficulty   *hexutil.Big   `json:"difficulty"`
	Number       *hexutil.Big   `json:"number"`
	GasLimit     *hexutil.Big   `json:"gasLimit"`
	GasUsed      *hexutil.Big   `json:"gasUsed"`
	Time         *hexutil.Big   `json:"timestamp"`
	Transactions Transactions   `json:"transactions"`
}

// Transactions is a Transaction slice type for basic sorting.
type Transactions []*types.Transaction

// Len returns the length of s
func (s Transactions) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s.
func (s Transactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type TransactionsByGasPrice Transactions

func (t TransactionsByGasPrice) Len() int      { return len(t) }
func (t TransactionsByGasPrice) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t TransactionsByGasPrice) Less(i, j int) bool {
	return t[i].GasPrice().Cmp(t[j].GasPrice()) < 0
}
