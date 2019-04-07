package web3j

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
	Slow          int64
	Standard      int64
	Fast          int64
	Glacial       int64
	ScoreSlow     float64
	ScoreStandard float64
	ScoreFast     float64
	ScoreGlacial  float64
	NumberOfTxs   int
}

type prediction struct {
	scores      map[int64]*score //blocknum -> score
	predictedAt int64            //blocknum of prediction
	Slow        int64
	Standard    int64
	Fast        int64
	Glacial     int64
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

func (s *scores) addPrediction(slow int64, standard int64, fast int64, glacial int64, at int64) {
	_, ok := s.predictions[at]
	if !ok {
		s.predictions[at] = &prediction{
			scores:      make(map[int64]*score),
			predictedAt: at,
			Slow:        slow,
			Standard:    standard,
			Fast:        fast,
			Glacial:     glacial,
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
			scoreGlacial := s.getPercentageOfTxsWithBiggerGP(block, predict.Glacial)

			predict.scores[i] = &score{
				Slow:          predict.Slow,
				Standard:      predict.Standard,
				Fast:          predict.Fast,
				Glacial:       predict.Glacial,
				ScoreSlow:     scoreSlow,
				ScoreStandard: scoreStandard,
				ScoreFast:     scoreFast,
				ScoreGlacial:  scoreGlacial,
				NumberOfTxs:   len(block.Transactions),
			}
		}
	}

	return nil
}

func (s *scores) getPercentageOfTxsWithBiggerGP(block *utils.Block, prediction int64) float64 {
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

		if tx.GasPrice().Int64() > prediction {
			percentage := (1.0 - (float64(idx) / float64(len(block.Transactions)))) * 100.0 //(1-idx/txs)*100
			return percentage
		}
	}

	return 0
}

func (s *scores) flush() error {
	fileName := fmt.Sprintf("web3jscores%v.csv", time.Now().Format(time.RFC3339))
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
		"priceGlacial",
		"scoreSlowPlus1",
		"scoreStandardPlus1",
		"scoreFastPlus1",
		"scoreGlacialPlus1",
		"scoreSlowPlus2",
		"scoreStandardPlus2",
		"scoreFastPlus2",
		"scoreGlacialPlus2",
		"scoreSlowPlus3",
		"scoreStandardPlus3",
		"scoreFastPlus3",
		"scoreGlacialPlus3",
		"scoreSlowPlus4",
		"scoreStandardPlus4",
		"scoreFastPlus4",
		"scoreGlacialPlus4",
		"scoreSlowPlus5",
		"scoreStandardPlus5",
		"scoreFastPlus5",
		"scoreGlacialPlus5",
		"scoreSlowPlus6",
		"scoreStandardPlus6",
		"scoreFastPlus6",
		"scoreGlacialPlus6",
		"scoreSlowPlus7",
		"scoreStandardPlus7",
		"scoreFastPlus7",
		"scoreGlacialPlus7",
		"scoreSlowPlus8",
		"scoreStandardPlus8",
		"scoreFastPlus8",
		"scoreGlacialPlus8",
		"scoreSlowPlus9",
		"scoreStandardPlus9",
		"scoreFastPlus9",
		"scoreGlacialPlus9",
		"scoreSlowPlus10",
		"scoreStandardPlus10",
		"scoreFastPlus10",
		"scoreGlacialPlus10",
	})

	if err != nil {
		return err
	}

	var records [][]string
	for blockNum, prediction := range s.predictions {
		record := []string{
			strconv.FormatInt(blockNum, 10),
			strconv.FormatInt(prediction.Slow, 10),
			strconv.FormatInt(prediction.Standard, 10),
			strconv.FormatInt(prediction.Fast, 10),
			strconv.FormatInt(prediction.Glacial, 10),
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
				record = append(record, strconv.FormatFloat(score.ScoreGlacial, 'f', 3, 64))
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
