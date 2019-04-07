package express

import (
	"math/big"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mariusgiger/ethereum-feeestimator/pkg/utils"
)

type CleanTx struct {
	Hash         common.Hash
	GasPrice     *big.Int
	GasPriceGwei *big.Int
}

type CleanBlock struct {
	BlockNumber       *big.Int
	TimeMined         time.Time
	BlockHash         common.Hash
	MinGasPrice       *big.Int
	MinGasPrice10Gwei *big.Int
	Transactions      []*CleanTx
}

func SortByBlockNumber(blocks []*CleanBlock) []*CleanBlock {
	sort.Slice(blocks, func(i, j int) bool { return blocks[i].BlockNumber.Cmp(blocks[j].BlockNumber) == -1 })
	return blocks
}

func newCleanTx(tx *types.Transaction) *CleanTx {
	gpGwei := roundGpTo10Gwei(tx.GasPrice())
	return &CleanTx{
		Hash:         tx.Hash(),
		GasPrice:     tx.GasPrice(),
		GasPriceGwei: gpGwei,
	}
}

func newCleanBlock(block *utils.Block) *CleanBlock {
	var timeStampUTC time.Time
	if block.Time != nil {
		timeStampUnix := block.Time.ToInt().Int64()
		timeStampUTC = time.Unix(timeStampUnix, 0)
	}

	return &CleanBlock{
		BlockHash:    block.Hash,
		BlockNumber:  block.Number.ToInt(),
		Transactions: make([]*CleanTx, 0),
		TimeMined:    timeStampUTC,
	}
}

type gasPriceGroup struct {
	GasPrice uint64
	Count    int
}

type hashpower []*hashpowerEntry
type hashpowerEntry struct {
	GasPrice        uint64
	Count           int
	CumulativeBlock int
	HashpPct        float64
}

type pricePrediction struct {
	HashpowerAccepting uint64
	GasPrice           uint64
}

type predictionTable struct {
	predictions []*pricePrediction
}

type gasPricePredictions struct {
	SafeLow     float64
	Standard    float64
	Fast        float64
	Fastest     float64
	BlockTime   int64
	BlockNumber uint64
}
