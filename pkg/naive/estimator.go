package naive

import (
	"errors"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/mariusgiger/ethereum-feeestimator/pkg/utils"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/gasprice"

	"go.uber.org/zap"
)

var (
	//RefreshInterval for fee estimation
	RefreshInterval = 10 * time.Second
)

// Estimator implements a naive gas price estimation
type Estimator struct {
	logger   *zap.Logger
	config   gasprice.Config
	maxEmpty int

	lastObserved *big.Int
	mutex        *sync.Mutex
	scores       *scores
	rpcClient    *utils.CachedRPCClient
}

// NewEstimator creates a new estimation.Estimator
func NewEstimator(logger *zap.Logger, config gasprice.Config, rpcClient *utils.CachedRPCClient) *Estimator {
	return &Estimator{
		logger:       logger,
		config:       config,
		maxEmpty:     config.Blocks / 2,
		mutex:        &sync.Mutex{},
		lastObserved: big.NewInt(-1),
		rpcClient:    rpcClient,
		scores:       newScores(rpcClient, logger),
	}
}

// Run starts the main event loop for estimating fees
func (e *Estimator) Run() error {
	ticker := time.NewTicker(RefreshInterval)
	defer ticker.Stop()

	errorChannel := make(chan error)
	go func() {
		err := e.estimateFees() //since ticker only ticks after interval
		if err != nil {
			errorChannel <- err
		}
		for {
			select {
			case <-ticker.C:
				err = e.estimateFees()
				if err != nil {
					errorChannel <- err
				}
			}
		}
	}()

	return <-errorChannel
}

func (e *Estimator) estimateFees() error {
	e.mutex.Lock() //prevents duplicate loading if operation needs longer than tick
	defer e.mutex.Unlock()

	latest, err := e.rpcClient.GetLastestBlock()
	if err != nil {
		return err
	}

	if e.lastObserved.Cmp(latest.Number.ToInt()) >= 0 {
		e.logger.Info("already predicted")
		return nil
	}

	prediction, err := e.SuggestGasPrice()
	if err != nil {
		e.logger.Error("an error occurred while suggesting gas price", zap.Error(err))
		return err
	}
	e.logger.Info("estimation complete: ", zap.Any("gasPriceGwei", prediction.Price.Uint64()/utils.GWei), zap.Any("prediction", prediction))
	e.scores.addPrediction(prediction.Price.Int64(), latest.Number.ToInt().Int64())
	return e.scores.predictScores()
}

// SuggestGasPrice suggests a gas price in gwei
func (e *Estimator) SuggestGasPrice() (*GasPricePrediction, error) {
	header, err := e.rpcClient.GetLastestBlock()
	if err != nil {
		return nil, err
	}

	currentBlockNumber := header.Number.ToInt()
	checkBlocks := e.config.Blocks
	ch := make(chan getBlockPricesResult, checkBlocks)
	sent := 0
	exp := 0

	var blockPrices []*big.Int
	blockNum := currentBlockNumber.Uint64()
	signer := types.NewEIP155Signer(nil)

	for sent < checkBlocks && blockNum > 0 {
		big := hexutil.Big(*big.NewInt(int64(blockNum)))
		go e.getBlockPrices(signer, &big, ch)
		sent++
		exp++
		blockNum-- //TODO possibly skip a block
	}

	maxEmpty := e.maxEmpty
	for exp > 0 {
		res := <-ch
		if res.err != nil {
			return nil, res.err
		}
		exp--
		if res.price != nil {
			blockPrices = append(blockPrices, res.price)
			continue
		}
		if maxEmpty > 0 {
			maxEmpty--
			continue
		}

		//TODO handle failed --> possibly reload or ignore as it is in gasPriceOracle
	}

	prediction := &GasPricePrediction{BlockNumber: currentBlockNumber}
	if len(blockPrices) > 0 {
		sort.Sort(bigIntArray(blockPrices))
		prediction.Price = blockPrices[(len(blockPrices)-1)*e.config.Percentile/100]

		if prediction.Price.Cmp(utils.MaxPrice) > 0 {
			prediction.Price = new(big.Int).Set(utils.MaxPrice)
		}

		e.lastObserved = currentBlockNumber
		return prediction, err
	}

	return nil, errors.New("not enough blocks")
}

// getBlockPrices calculates the lowest transaction gas price in a given block
// and sends it to the result channel. If the block is empty price is nil.
// If the block is incomplete or an error occurred the error is sent to the channel.
func (e *Estimator) getBlockPrices(signer types.Signer, blockNum *hexutil.Big, ch chan getBlockPricesResult) {
	block, err := e.rpcClient.GetBlockByNumber(blockNum.ToInt())
	if err != nil {
		ch <- getBlockPricesResult{nil, nil, err}
		return
	}

	sort.Sort(utils.TransactionsByGasPrice(block.Transactions))

	for _, tx := range block.Transactions {
		sender, err := types.Sender(signer, tx)
		if err == nil && sender != block.Miner {
			ch <- getBlockPricesResult{tx.GasPrice(), blockNum.ToInt(), nil}
			return
		}
	}
	ch <- getBlockPricesResult{nil, nil, nil}
}
