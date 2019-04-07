package express

import (
	"math"
	"math/big"

	"github.com/mariusgiger/ethereum-feeestimator/pkg/utils"
)

func roundGpTo10Gwei(gasPrice *big.Int) *big.Int {
	//TODO use big int calc
	//TODO why is 10 gwei?
	gp := float64(gasPrice.Int64()) / utils.TenGWei
	if gp >= 1 && gp < 10 {
		gp = math.Floor(gp)
	} else if gp >= 10 {
		gp = gp / 10
		gp = math.Floor(gp)
		gp = gp * 10
	} else {
		gp = 0 //TODO this should not happen --> set it to a safe default instead
	}

	return big.NewInt(int64(gp))
}

func Min(nums []uint64) uint64 {
	min := uint64(nums[0])
	for _, num := range nums {
		if num < min {
			min = num
		}
	}

	return min
}

func Max(nums []uint64) uint64 {
	max := uint64(0)
	for _, num := range nums {
		if num > max {
			max = num
		}
	}

	return max
}

func MinBig(nums []*big.Int) *big.Int {
	min := big.NewInt(nums[0].Int64())
	for _, num := range nums {
		if num.Cmp(min) < 0 {
			min = num
		}
	}

	return min
}

func CumSum(s []int) []int {
	receiver := make([]int, len(s))
	if len(s) == 0 {
		return receiver[:0]
	}

	receiver[0] = s[0]
	for i := 1; i < len(s); i++ {
		receiver[i] += receiver[i-1] + s[i]
	}

	return receiver
}

func Sum(nums []int) int {
	sum := int(0)
	for _, num := range nums {
		sum += num
	}

	return sum
}

func DiffBlockNumber(blocks []*CleanBlock) []int64 {
	diffs := make([]int64, 0)
	for i := 0; i < len(blocks); i++ {
		if i == 0 {
			diffs = append(diffs, math.MaxInt64)
		} else {
			diffs = append(diffs, blocks[i].BlockNumber.Int64()-blocks[i-1].BlockNumber.Int64())
		}
	}

	return diffs
}
