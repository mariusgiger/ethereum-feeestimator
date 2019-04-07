package naive

import (
	"math/big"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestSorting(t *testing.T) {
	// arrange
	lowGP := big.NewInt(3)
	standardGP := big.NewInt(5)
	fastGP := big.NewInt(6)
	tx := types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, standardGP, nil)
	tx1 := types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, fastGP, nil)
	tx2 := types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, lowGP, nil)
	txs := []*types.Transaction{tx, tx1, tx2}

	// act
	sort.Sort(transactionsByGasPrice(txs))

	// assert
	require.Len(t, txs, 3)
	assert.Equal(t, txs[0].GasPrice(), lowGP)
	assert.Equal(t, txs[1].GasPrice(), standardGP)
	assert.Equal(t, txs[2].GasPrice(), fastGP)
}
