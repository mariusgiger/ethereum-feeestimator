package naive

import (
	"encoding/csv"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/mariusgiger/ethereum-feeestimator/pkg/utils"
	"go.uber.org/zap"
)

type score struct {
	Standard      int64
	ScoreStandard float64
	NumberOfTxs   int
}

type prediction struct {
	scores      map[int64]*score //blocknum -> score
	predictedAt int64            //blocknum of prediction
	Standard    int64
}

type scores struct {
	predictions map[int64]*prediction
	rpcClient   *utils.CachedRPCClient
	logger      *zap.Logger
}

func newScores(rpcClient *utils.CachedRPCClient, logger *zap.Logger) *scores {
	return &scores{
		rpcClient:   rpcClient,
		logger:      logger,
		predictions: make(map[int64]*prediction),
	}
}

func (s *scores) addPrediction(standard int64, at int64) {
	_, ok := s.predictions[at]
	if !ok {
		s.predictions[at] = &prediction{
			scores:      make(map[int64]*score),
			predictedAt: at,
			Standard:    standard,
		}
	}
}

func (s *scores) predictScores() error {
	for num, pred := range s.predictions {
		err := s.comparePredictionToNext10Blocks(num, pred)
		if err != nil {
			return err
		}
	}

	return s.flush()
}

func (s *scores) comparePredictionToNext10Blocks(blockNumber int64, predict *prediction) error {
	for i := blockNumber + 1; i < blockNumber+11; i++ {
		_, ok := predict.scores[i]
		if !ok {
			//load transactions of block i
			block, err := s.rpcClient.GetBlockByNumber(big.NewInt(i))
			if err == utils.ErrBlockNotFound {
				return nil //block does not yet exist
			}

			if err != nil {
				return err
			}

			sort.Sort(utils.TransactionsByGasPrice(block.Transactions))
			scoreStandard := s.getPercentageOfTxsWithBiggerGP(block, predict.Standard)

			predict.scores[i] = &score{
				Standard:      predict.Standard,
				ScoreStandard: scoreStandard,
				NumberOfTxs:   len(block.Transactions),
			}
		}
	}

	return nil
}

func (s *scores) getPercentageOfTxsWithBiggerGP(block *utils.Block, prediction int64) float64 {
	for idx, tx := range block.Transactions {
		if tx.GasPrice().Int64() > prediction {
			percentage := (1.0 - (float64(idx) / float64(len(block.Transactions)))) * 100.0 //(1-idx/txs)*100
			return percentage
		}
	}

	return 0
}

func (s *scores) flush() error {
	fileName := fmt.Sprintf("naivescores%v.csv", time.Now().Format(time.RFC3339))
	f, err := os.OpenFile("./output/"+fileName, os.O_CREATE|os.O_RDWR, 0660)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	err = w.Write([]string{
		"block_number",
		"priceStandard",
		"scoreStandardPlus1",
		"scoreStandardPlus2",
		"scoreStandardPlus3",
		"scoreStandardPlus4",
		"scoreStandardPlus5",
		"scoreStandardPlus6",
		"scoreStandardPlus7",
		"scoreStandardPlus8",
		"scoreStandardPlus9",
		"scoreStandardPlus10",
	})

	if err != nil {
		return err
	}

	var records [][]string
	for blockNum, prediction := range s.predictions {
		record := []string{
			strconv.FormatInt(blockNum, 10),
			strconv.FormatInt(prediction.Standard, 10),
		}
		for i := blockNum + 1; i < blockNum+11; i++ {
			score, ok := prediction.scores[i]
			if !ok {
				record = append(record, strconv.Itoa(-1))
			} else {
				record = append(record, strconv.FormatFloat(score.ScoreStandard, 'f', 3, 64))
			}
		}

		records = append(records, record)
	}

	err = w.WriteAll(records)
	if err != nil {
		return err
	}

	return nil
}
