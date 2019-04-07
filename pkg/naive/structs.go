package naive

import (
	"math/big"
)

type getBlockPricesResult struct {
	price       *big.Int
	blockNumber *big.Int
	err         error
}

type bigIntArray []*big.Int

func (s bigIntArray) Len() int           { return len(s) }
func (s bigIntArray) Less(i, j int) bool { return s[i].Cmp(s[j]) < 0 }
func (s bigIntArray) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type GasPricePrediction struct {
	Price       *big.Int
	BlockNumber *big.Int
}
