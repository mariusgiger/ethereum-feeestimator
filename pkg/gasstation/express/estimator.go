package express

import (
	"math"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/mariusgiger/ethereum-feeestimator/pkg/utils"

	. "github.com/ahmetb/go-linq"
	"go.uber.org/zap"
)

var (
	//RefreshInterval for fee estimation
	RefreshInterval = 10 * time.Second

	//InspectedBlocks sets the amount of blocks inspected
	InspectedBlocks = uint64(100)
)

// Estimator implements gas price estimation based on the ethereum gasstation express algorithm
type Estimator struct {
	cleanBlocks             map[string]*CleanBlock
	logger                  *zap.Logger
	rpcClient               *utils.CachedRPCClient
	lastObservedBlockNumber uint64

	mutex  *sync.Mutex
	scores *scores
}

// NewEstimator returns a new express estimator
func NewEstimator(logger *zap.Logger, rpcClient *utils.CachedRPCClient) *Estimator {
	return &Estimator{
		rpcClient:   rpcClient,
		logger:      logger,
		cleanBlocks: make(map[string]*CleanBlock),
		mutex:       &sync.Mutex{},
		scores:      newScores(rpcClient, logger),
	}
}

// Run starts the main event loop for estimating fees
func (e *Estimator) Run() error {
	ticker := time.NewTicker(RefreshInterval)
	defer ticker.Stop()

	quit := make(chan struct{})
	errorChannel := make(chan error)
	go func() {
		err := e.doWork() //since ticker only ticks after interval
		if err != nil {
			errorChannel <- err
			return
		}
		for {
			select {
			case <-ticker.C:
				err = e.doWork()
				if err != nil {
					errorChannel <- err
					return
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	return <-errorChannel
}

func (e *Estimator) doWork() error {
	e.mutex.Lock() //prevents duplicate loading if operation needs longer than tick
	defer e.mutex.Unlock()

	//get current block number
	latestBlock, err := e.rpcClient.GetLastestBlock()
	if err != nil {
		return err
	}
	blockNumber := latestBlock.Number.ToInt().Uint64()

	if blockNumber <= e.lastObservedBlockNumber {
		e.logger.Info("already predicted")
		return nil
	}

	//load last tx not in cache (max InspectedBlocks)
	if blockNumber > e.lastObservedBlockNumber {
		//TODO only consider mined blocks mined_block_num = block-3
		firstNew := blockNumber - InspectedBlocks
		if firstNew < e.lastObservedBlockNumber {
			firstNew = e.lastObservedBlockNumber + 1
		}

		e.logger.Info("getting blocks", zap.Uint64("from", firstNew), zap.Uint64("to", blockNumber))
		for currentBlock := firstNew; currentBlock <= blockNumber; currentBlock++ {
			cleanBlock, err := e.processBlockTxs(big.NewInt(int64(currentBlock)))
			if err != nil {
				return err
			}

			e.cleanBlocks[cleanBlock.BlockHash.String()] = cleanBlock
		}
	}
	e.lastObservedBlockNumber = blockNumber

	//clean cache
	//TODO truncate the cache to 200 blocks

	//estimate fees
	err = e.estimateFees()
	return err
}

func (e *Estimator) estimateFees() error {
	//get hashpower table from last 200 blocks
	hp, blockTime, err := e.analyzeLast200Blocks()
	if err != nil {
		return err
	}

	table := makePredictionTable(hp)
	predictions := getGaspriceRecs(table, e.lastObservedBlockNumber, blockTime)
	e.logger.Info("estimation complete: ", zap.Any("predictions", predictions), zap.Any("standardGwei", predictions.Standard/utils.GWei))

	e.scores.addPrediction(predictions.SafeLow, predictions.Standard, predictions.Fast, predictions.Fastest, int64(predictions.BlockNumber))
	return e.scores.predictScores()
}

func (e *Estimator) processBlockTxs(blockNumber *big.Int) (*CleanBlock, error) {
	//TODO this returns invalid blocks --> GP = 0 find out why
	block, err := e.rpcClient.GetBlockByNumber(blockNumber)
	if err != nil {
		return nil, err
	}

	sort.Sort(utils.TransactionsByGasPrice(block.Transactions))
	cleanBlock := newCleanBlock(block)
	for _, tx := range block.Transactions {
		if tx.GasPrice().Uint64() == 0 { //It sometimes happens that a whole block has gp = 0
			e.logger.Warn("gas price was 0", zap.Uint64("number", block.Number.ToInt().Uint64()))
			continue
		}

		cleanTx := newCleanTx(tx)
		cleanBlock.Transactions = append(cleanBlock.Transactions, cleanTx)
	}

	if len(cleanBlock.Transactions) > 0 {
		var gasPrices []*big.Int
		From(cleanBlock.Transactions).Select(
			func(block interface{}) interface{} {
				return block.(*CleanTx).GasPrice
			}).ToSlice(&gasPrices)

		cleanBlock.MinGasPrice = MinBig(gasPrices)
		cleanBlock.MinGasPrice10Gwei = roundGpTo10Gwei(cleanBlock.MinGasPrice)
	} else {
		cleanBlock.MinGasPrice = nil
	}

	return cleanBlock, nil
}

func (e *Estimator) analyzeLast200Blocks() (hashpower, int64, error) {
	blocks := make([]*CleanBlock, 0)
	for _, value := range e.cleanBlocks {
		blocks = append(blocks, value)
	}

	blocks = SortByBlockNumber(blocks)
	recentBlockIndex := 200
	if len(blocks) < 200 {
		recentBlockIndex = len(blocks)
	}

	recentBlocks := blocks[0 : recentBlockIndex-1]
	var groupedByMinGasPrice []*gasPriceGroup
	From(recentBlocks).GroupByT(func(block *CleanBlock) uint64 {
		if block.MinGasPrice10Gwei == nil {
			return 0 //TODO possibly ignore such blocks
		}

		return block.MinGasPrice10Gwei.Uint64()
	}, func(block *CleanBlock) *CleanBlock {
		return block
	}).SelectT(func(group Group) *gasPriceGroup {
		return &gasPriceGroup{GasPrice: group.Key.(uint64), Count: len(group.Group)}
	}).OrderByT(func(group *gasPriceGroup) uint64 {
		return group.GasPrice
	}).ToSlice(&groupedByMinGasPrice)

	var hp hashpower
	for _, group := range groupedByMinGasPrice {
		hp = append(hp, &hashpowerEntry{
			GasPrice: group.GasPrice,
			Count:    group.Count,
		})
	}

	var counts []int
	From(groupedByMinGasPrice).SelectT(
		func(group *gasPriceGroup) int {
			return group.Count
		}).ToSlice(&counts)

	cumBlocks := CumSum(counts)
	for i, cumBlock := range cumBlocks {
		hp[i].CumulativeBlock = cumBlock
	}

	totalBlocks := Sum(counts)
	hashpPcts := getHashpPct(cumBlocks, totalBlocks)
	for i, hashpPct := range hashpPcts {
		hp[i].HashpPct = hashpPct
	}

	//get avg blockinterval time
	blockinterval := DiffBlockNumber(blocks)
	sum := int64(0)
	avgCnt := int64(0)
	for i := 0; i < len(blockinterval); i++ {
		diff := blockinterval[i]
		if diff < 0 || diff > 1 {
			blockinterval[i] = math.MaxInt64
			diff = blockinterval[i]
		}

		if diff != math.MaxInt64 {
			sum += diff
			avgCnt++
		}
	}

	//TODO use timestamp?
	var avgTimemined int64
	if avgCnt != 0 {
		avgTimemined = sum / avgCnt
	} else {
		avgTimemined = int64(15)
	}

	return hp, avgTimemined, nil
}

func getHashpPct(cumBlocks []int, totalBlocks int) []float64 {
	hashpPcts := make([]float64, 0)
	for _, v := range cumBlocks {
		hashpPct := float64(v) / float64(totalBlocks) * 100.0
		hashpPcts = append(hashpPcts, hashpPct)
	}

	return hashpPcts
}

func emptyTable(start int, stop int, step int) []uint64 {
	table := make([]uint64, 0)
	for i := start; i <= stop; i = i + step {
		table = append(table, uint64(i))
	}

	return table
}

func makePredictionTable(hp hashpower) *predictionTable {
	table := emptyTable(10, 1010, 10)
	pTable2 := emptyTable(0, 10, 1)
	for _, val := range pTable2 {
		table = append(table, val)
	}

	sort.Slice(table, func(i, j int) bool {
		return table[i] < table[j]
	})

	predictions := make([]*pricePrediction, 0)
	for _, val := range table {
		hpa := getHashpowerAccepting(val, hp)
		prediction := &pricePrediction{
			HashpowerAccepting: hpa,
			GasPrice:           val,
		}
		predictions = append(predictions, prediction)
	}

	return &predictionTable{predictions}
}

//gets the hash power accpeting the gas price over last 200 blocks
func getHashpowerAccepting(price uint64, hp hashpower) uint64 {
	hpa := uint64(0)
	hpas := make([]uint64, 0)
	for i, group := range hp {
		if price >= group.GasPrice {
			hpas = append(hpas, uint64(hp[i].HashpPct))
		}
	}

	var prices []uint64
	From(hp).SelectT(
		func(hpEntry *hashpowerEntry) uint64 {
			return hpEntry.GasPrice
		}).ToSlice(&prices)

	if price > Max(prices) {
		hpa = 100
	} else if price < Min(prices) {
		hpa = 0
	} else {
		hpa = Max(hpas)
	}

	return hpa
}

func getGaspriceRecs(table *predictionTable, blockNumber uint64, blockTime int64) *gasPricePredictions {
	predictions := &gasPricePredictions{}

	var lowPrices []uint64
	From(table.predictions).WhereT(func(prediction *pricePrediction) bool {
		return prediction.HashpowerAccepting >= SafeLow
	}).SelectT(func(prediction *pricePrediction) uint64 {
		return prediction.GasPrice
	}).ToSlice(&lowPrices)
	predictions.SafeLow = (float64(Min(lowPrices)) / float64(10)) * utils.GWei //to wei

	var avgPrices []uint64
	From(table.predictions).WhereT(func(prediction *pricePrediction) bool {
		return prediction.HashpowerAccepting >= Standard
	}).SelectT(func(prediction *pricePrediction) uint64 {
		return prediction.GasPrice
	}).ToSlice(&avgPrices)
	predictions.Standard = (float64(Min(avgPrices)) / float64(10)) * utils.GWei //to wei

	var fastPrices []uint64
	From(table.predictions).WhereT(func(prediction *pricePrediction) bool {
		return prediction.HashpowerAccepting >= Fast
	}).SelectT(func(prediction *pricePrediction) uint64 {
		return prediction.GasPrice
	}).ToSlice(&fastPrices)
	predictions.Fast = (float64(Min(fastPrices)) / float64(10)) * utils.GWei //to wei

	var hashpowers []uint64
	From(table.predictions).SelectT(func(prediction *pricePrediction) uint64 {
		return prediction.HashpowerAccepting
	}).ToSlice(&hashpowers)

	var fastestPrices []uint64
	hpmax := Max(hashpowers)
	From(table.predictions).WhereT(func(prediction *pricePrediction) bool {
		return prediction.HashpowerAccepting == hpmax
	}).SelectT(func(prediction *pricePrediction) uint64 {
		return prediction.GasPrice
	}).ToSlice(&fastestPrices)
	predictions.Fastest = (float64(fastestPrices[0]) / float64(10)) * utils.GWei //to wei

	predictions.BlockNumber = blockNumber
	predictions.BlockTime = blockTime
	return predictions
}
