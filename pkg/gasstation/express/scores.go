package express

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
	Slow          float64
	Standard      float64
	Fast          float64
	Fastest       float64
	ScoreSlow     float64
	ScoreStandard float64
	ScoreFast     float64
	ScoreFastest  float64
	NumberOfTxs   int
}

type prediction struct {
	scores      map[int64]*score //blocknum -> score
	predictedAt int64            //blocknum of prediction
	Slow        float64
	Standard    float64
	Fast        float64
	Fastest     float64
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

func (s *scores) addPrediction(slow float64, standard float64, fast float64, fastest float64, at int64) {
	_, ok := s.predictions[at]
	if !ok {
		s.predictions[at] = &prediction{
			scores:      make(map[int64]*score),
			predictedAt: at,
			Slow:        slow,
			Standard:    standard,
			Fast:        fast,
			Fastest:     fastest,
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
			scoreSlow := s.getPercentageOfTxsWithBiggerGP(block, predict.Slow)
			scoreStandard := s.getPercentageOfTxsWithBiggerGP(block, predict.Standard)
			scoreFast := s.getPercentageOfTxsWithBiggerGP(block, predict.Fast)
			scoreFastest := s.getPercentageOfTxsWithBiggerGP(block, predict.Fastest)

			predict.scores[i] = &score{
				Slow:          predict.Slow,
				Standard:      predict.Standard,
				Fast:          predict.Fast,
				Fastest:       predict.Fastest,
				ScoreSlow:     scoreSlow,
				ScoreStandard: scoreStandard,
				ScoreFast:     scoreFast,
				ScoreFastest:  scoreFastest,
				NumberOfTxs:   len(block.Transactions),
			}
		}
	}

	return nil
}

func (s *scores) getPercentageOfTxsWithBiggerGP(block *utils.Block, prediction float64) float64 {
	for idx, tx := range block.Transactions {
		//TODO ignore coinbase txs
		// signer := types.NewEIP155Signer(big.NewInt(1)) //mainnet
		// sender, err := types.Sender(signer, tx)
		// if err != nil {
		// 	s.logger.Error("an error occurred", zap.Error(err))
		// 	panic("err")
		// }
		// if sender == block.Miner { //ignore coinbase txs
		// 	s.logger.Info("coinbase", zap.Any("tx", sender), zap.Any("block", block.Miner))
		// 	continue
		// }

		if tx.GasPrice().Int64() > int64(prediction) {
			percentage := (1.0 - (float64(idx) / float64(len(block.Transactions)))) * 100.0 //(1-idx/txs)*100
			return percentage
		}
	}

	return 0
}

func (s *scores) flush() error {
	fileName := fmt.Sprintf("expressscores%v.csv", time.Now().Format(time.RFC3339))
	f, err := os.OpenFile("./output/"+fileName, os.O_CREATE|os.O_RDWR, 0660)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	err = w.Write([]string{
		"block_number",
		"priceSlow",
		"priceStandard",
		"priceFast",
		"priceFastest",
		"scoreSlowPlus1",
		"scoreStandardPlus1",
		"scoreFastPlus1",
		"scoreFastestPlus1",
		"scoreSlowPlus2",
		"scoreStandardPlus2",
		"scoreFastPlus2",
		"scoreFastestPlus2",
		"scoreSlowPlus3",
		"scoreStandardPlus3",
		"scoreFastPlus3",
		"scoreFastestPlus3",
		"scoreSlowPlus4",
		"scoreStandardPlus4",
		"scoreFastPlus4",
		"scoreFastestPlus4",
		"scoreSlowPlus5",
		"scoreStandardPlus5",
		"scoreFastPlus5",
		"scoreFastestPlus5",
		"scoreSlowPlus6",
		"scoreStandardPlus6",
		"scoreFastPlus6",
		"scoreFastestPlus6",
		"scoreSlowPlus7",
		"scoreStandardPlus7",
		"scoreFastPlus7",
		"scoreFastestPlus7",
		"scoreSlowPlus8",
		"scoreStandardPlus8",
		"scoreFastPlus8",
		"scoreFastestPlus8",
		"scoreSlowPlus9",
		"scoreStandardPlus9",
		"scoreFastPlus9",
		"scoreFastestPlus9",
		"scoreSlowPlus10",
		"scoreStandardPlus10",
		"scoreFastPlus10",
		"scoreFastestPlus10",
	})

	if err != nil {
		return err
	}

	var records [][]string
	for blockNum, prediction := range s.predictions {
		record := []string{
			strconv.FormatInt(blockNum, 10),
			strconv.FormatFloat(prediction.Slow, 'f', 3, 64),
			strconv.FormatFloat(prediction.Standard, 'f', 3, 64),
			strconv.FormatFloat(prediction.Fast, 'f', 3, 64),
			strconv.FormatFloat(prediction.Fastest, 'f', 3, 64),
		}
		for i := blockNum + 1; i < blockNum+11; i++ {
			score, ok := prediction.scores[i]
			if !ok {
				record = append(record, strconv.Itoa(-1))
				record = append(record, strconv.Itoa(-1))
				record = append(record, strconv.Itoa(-1))
				record = append(record, strconv.Itoa(-1))
			} else {
				record = append(record, strconv.FormatFloat(score.ScoreSlow, 'f', 3, 64))
				record = append(record, strconv.FormatFloat(score.ScoreStandard, 'f', 3, 64))
				record = append(record, strconv.FormatFloat(score.ScoreFast, 'f', 3, 64))
				record = append(record, strconv.FormatFloat(score.ScoreFastest, 'f', 3, 64))
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
