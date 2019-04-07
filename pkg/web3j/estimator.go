package web3j

import (
	"errors"
	"math"
	"math/big"
	"sort"
	"sync"
	"time"

	. "github.com/ahmetb/go-linq"
	"github.com/mariusgiger/ethereum-feeestimator/pkg/utils"
	"go.uber.org/zap"
)

var (
	//RefreshInterval for fee estimation
	RefreshInterval = 10 * time.Second
)

// Estimator implements a naive gas price estimation
type Estimator struct {
	logger *zap.Logger

	rpcClient    *utils.CachedRPCClient
	mutex        *sync.Mutex
	scores       *scores
	lastObserved int64
}

// NewEstimator creates a new estimation.Estimator
func NewEstimator(logger *zap.Logger, rpcClient *utils.CachedRPCClient) *Estimator {
	return &Estimator{
		rpcClient: rpcClient,
		logger:    logger,
		mutex:     &sync.Mutex{},
		scores:    newScores(rpcClient, logger),
	}
}

// Run starts the main event loop for estimating fees
func (e *Estimator) Run() error {
	ticker := time.NewTicker(RefreshInterval)
	defer ticker.Stop()

	defer func() {
		// panic?
		p := recover()
		if p == nil {
			return // done, no panic
		}

		// error?
		err, ok := p.(error)
		if !ok {
			panic(p) // not error, re-raise
		}

		e.logger.Error("unhandled panic!", zap.Error(err))
	}()

	errorChannel := make(chan error)
	go func() {
		err := e.estimateFees() //since ticker only ticks after interval
		//TODO handle errors
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
				//TODO handle errors
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
	latestNum := latest.Number.ToInt().Int64()
	if e.lastObserved >= latestNum {
		e.logger.Info("already predicted")
		return nil
	}

	//fast: mine within 1 minute
	fastGasPrice, err := func() (int64, error) {
		maxWaitSeconds := int64(60)
		sampleSize := int64(120)
		probability := int(98)
		return e.constructTimeBasedStrategy(maxWaitSeconds, sampleSize, probability)()
	}()
	if err != nil {
		e.logger.Error("error while predicting fast price", zap.Error(err))
		return err
	}

	//medium: mine within 10 minutes
	mediumGasPrice, err := func() (int64, error) {
		maxWaitSeconds := int64(600)
		sampleSize := int64(120)
		probability := int(98)
		return e.constructTimeBasedStrategy(maxWaitSeconds, sampleSize, probability)()
	}()
	if err != nil {
		e.logger.Error("error while predicting medium price", zap.Error(err))
		return err
	}

	//slow: mine within 1 hour (60 minutes)
	slowGasPrice, err := func() (int64, error) {
		maxWaitSeconds := int64(60 * 60)
		sampleSize := int64(120)
		probability := int(98)
		return e.constructTimeBasedStrategy(maxWaitSeconds, sampleSize, probability)()
	}()
	if err != nil {
		e.logger.Error("error while predicting slow price", zap.Error(err))
		return err
	}

	//glacial: mine within the next 24 hours.
	glacialGasPrice, err := func() (int64, error) {
		maxWaitSeconds := int64(60 * 60 * 24)
		sampleSize := int64(720)
		probability := int(98)
		return e.constructTimeBasedStrategy(maxWaitSeconds, sampleSize, probability)()
	}()
	if err != nil {
		e.logger.Error("error while predicting glacial price", zap.Error(err))
		return err
	}

	e.lastObserved = latestNum
	e.logger.Info(
		"predictions",
		zap.Int64("fast", fastGasPrice),
		zap.Int64("medium", mediumGasPrice),
		zap.Int64("slow", slowGasPrice),
		zap.Int64("glacial", glacialGasPrice),
		zap.Float64("fastGwei", float64(fastGasPrice)/utils.GWei),
		zap.Float64("mediumGwei", float64(mediumGasPrice)/utils.GWei),
		zap.Float64("slowGwei", float64(slowGasPrice)/utils.GWei),
		zap.Float64("glacialGwei", float64(glacialGasPrice)/utils.GWei),
	)
	e.scores.addPrediction(slowGasPrice, mediumGasPrice, fastGasPrice, glacialGasPrice, latestNum)
	return e.scores.predictScores()
}

// A gas pricing strategy that uses recently mined block data to derive a gas
// price for which a transaction is likely to be mined within X seconds with
// probability P.
// maxWaitSeconds: The desired maxiumum number of seconds the
//     transaction should take to mine.
// sampleSize: The number of recent blocks to sample
// probability: An integer representation of the desired probability
//     that the transaction will be mined within ``max_wait_seconds``.  0 means 0%
//     and 100 means 100%.
func (e *Estimator) constructTimeBasedStrategy(maxWaitSeconds int64, sampleSize int64, probability int) func() (int64, error) {
	return func() (int64, error) {
		avgBlockTime, err := e.getAvgBlockTime(sampleSize)
		if err != nil {
			return 0, err
		}
		e.logger.Info("avg block time", zap.Any("time", avgBlockTime))

		maxWaitSecondsFloat := new(big.Float).SetInt64(maxWaitSeconds)
		waitBlocks, _ := maxWaitSecondsFloat.Quo(maxWaitSecondsFloat, avgBlockTime).Float64()
		waitBlocks = math.Ceil(waitBlocks)

		rawMinerData, err := e.getRawMinerData(sampleSize)
		if err != nil {
			return 0, err
		}
		minerData := e.aggregateMinerData(rawMinerData)
		probabilities := e.computeProbabilities(minerData, waitBlocks, sampleSize)
		gasPrice := e.computeGasPrice(probabilities, float64(probability)/100)
		return gasPrice, nil
	}
}

func (e *Estimator) getAvgBlockTime(sampleSize int64) (*big.Float, error) {
	header, err := e.rpcClient.GetLastestBlock()
	if err != nil {
		return nil, err
	}

	latestBlockNumber := header.Number.ToInt()
	constrainedSampleSize := Min(latestBlockNumber, big.NewInt(sampleSize))
	if constrainedSampleSize.Cmp(big.NewInt(0)) == 0 {
		return nil, errors.New("Constrained sample size is 0")
	}

	oldestBlockNumber := latestBlockNumber.Sub(latestBlockNumber, constrainedSampleSize)
	oldest, err := e.rpcClient.GetBlockHeaderByNumber(oldestBlockNumber)
	if err != nil {
		return nil, err
	}

	latestTime := header.Time.ToInt()
	diff := latestTime.Sub(latestTime, oldest.Time.ToInt())

	avgBlockTime := new(big.Float).SetInt(diff)
	constrainedSampleSizeFloat := new(big.Float).SetInt(constrainedSampleSize)
	return avgBlockTime.Quo(avgBlockTime, constrainedSampleSizeFloat), nil
}

func (e *Estimator) getRawMinerData(sampleSize int64) ([]*Tx, error) {
	latest, err := e.rpcClient.GetLastestBlock()
	if err != nil {
		return nil, err
	}

	txs := make([]*Tx, len(latest.Transactions))
	for i, tx := range latest.Transactions {
		cleanTx := &Tx{
			Miner:    latest.Miner.String(),
			Hash:     latest.Hash.String(),
			GasPrice: tx.GasPrice(),
		}

		txs[i] = cleanTx
	}

	block := latest
	for i := int64(0); i < sampleSize-1; i++ {
		blockNumber := block.Number.ToInt()
		if blockNumber.Cmp(big.NewInt(0)) == 0 {
			break
		}

		//we intentionally trace backwards using parent hashes rather than
		//block numbers to make caching the data easier to implement.
		loadedBlock, err := e.rpcClient.GetBlockByHash(block.ParentHash)
		if err != nil {
			return nil, err
		}

		for _, tx := range loadedBlock.Transactions {
			cleanTx := &Tx{
				Miner:    loadedBlock.Miner.String(),
				Hash:     loadedBlock.Hash.String(),
				GasPrice: tx.GasPrice(),
			}

			txs = append(txs, cleanTx)
		}

		block = loadedBlock
	}

	return txs, nil
}

func (e *Estimator) aggregateMinerData(txs []*Tx) []*minerData {
	var query []*minerData
	From(txs).GroupByT(func(tx *Tx) string {
		return tx.Miner
	}, func(tx *Tx) *Tx {
		return tx
	}).SelectT(func(group Group) *minerData {
		miner := group.Key.(string)
		var gasPrices []int64
		From(group.Group).SelectT(func(tx *Tx) int64 {
			return tx.GasPrice.Int64()
		}).OrderByT(func(gasPrice int64) int64 {
			return gasPrice
		}).ToSlice(&gasPrices)

		blocks := From(group.Group).GroupByT(func(tx *Tx) string {
			return tx.Hash //group by block hash
		}, func(tx *Tx) *Tx {
			return tx
		}).Count()

		pricePercentile := gasPrices[(len(gasPrices)-1)*20/100]
		//pricePercentile := percentile(gasPrices, 20)

		return &minerData{
			Miner:                 miner,
			Blocks:                blocks,
			MinGasPrice:           MinInt64(gasPrices),
			LowPercentileGasPrice: pricePercentile,
		}
	}).ToSlice(&query)

	return query
}

// Computes the probabilities that a txn will be accepted at each of the gas
// prices accepted by the miners.
func (e *Estimator) computeProbabilities(miners []*minerData, waitBlocks float64, sampleSize int64) []*Probability {
	sort.Slice(miners, func(i, j int) bool {
		return miners[i].LowPercentileGasPrice > miners[j].LowPercentileGasPrice
	})

	var probabilties []*Probability
	for idx, miner := range miners {
		lowPercentileGasPrice := miner.LowPercentileGasPrice
		numBlocksAcceptingGasPrice := From(miners[idx:]).SelectT(func(data *minerData) int { return data.Blocks }).SumInts()
		invProbPerBlock := float64(sampleSize-numBlocksAcceptingGasPrice) / float64(sampleSize)
		probabilityAccepted := 1 - math.Pow(invProbPerBlock, waitBlocks)
		probabilties = append(probabilties, &Probability{GasPrice: lowPercentileGasPrice, Probability: probabilityAccepted})
	}

	return probabilties
}

//  Given a sorted range of ``Probability`` named-tuples returns a gas price
//  computed based on where the ``desired_probability`` would fall within the
//  range.
//  	probabilities: An iterable of `Probability` named-tuples sorted in reverse order.
//  	desired_probability: An floating point representation of the desired
//      probability. (e.g. ``85% -> 0.85``)
func (e *Estimator) computeGasPrice(probabilities []*Probability, desiredProbability float64) int64 {
	first := probabilities[0]
	last := probabilities[len(probabilities)-1]

	if desiredProbability >= first.Probability {
		return first.GasPrice
	} else if desiredProbability <= last.Probability {
		return last.GasPrice
	}

	for i := 0; i < len(probabilities)-1; i = i + 1 {
		left := probabilities[i]
		right := probabilities[i+1]
		if desiredProbability < right.Probability {
			continue
		} else if desiredProbability > left.Probability {
			// This code block should never be reachable as it would indicate
			// that we already passed by the probability window in which our
			// `desired_probability` is located.
			panic("Invariant")
		}

		adjProb := desiredProbability - right.Probability
		windowSize := left.Probability - right.Probability
		position := adjProb / windowSize
		gasWindowSize := float64(left.GasPrice - right.GasPrice)
		gasPrice := int64(math.Ceil(float64(right.GasPrice) + gasWindowSize*position))
		return gasPrice
	}

	// The initial `if/else` clause in this function handles the case where
	// the `desired_probability` is either above or below the min/max
	// probability found in the `probabilities`.

	// With these two cases handled, the only way this code block should be
	// reachable would be if the `probabilities` were not sorted correctly.
	// Otherwise, the `desired_probability` **must** fall between two of the
	// values in the `probabilities``.
	panic("Invariant - probabilities were not sorted correctly")
}
