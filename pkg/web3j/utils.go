package web3j

import "math/big"

func Min(a, b *big.Int) *big.Int {
	if a.Cmp(b) < 0 {
		return a
	}
	return b
}

func MinInt64(nums []int64) int64 {
	min := int64(nums[0])
	for _, num := range nums {
		if num < min {
			min = num
		}
	}

	return min
}

func percentile(sortedValues []int64, percentile int) int64 {
	rank := int64(len(sortedValues) * percentile / 100)
	index := int64(-1)
	if rank > 0 {
		index := rank - 1
		if index < 0 {
			return sortedValues[0]
		}
	} else {
		index = rank
	}

	if index%1 == 0 {
		return sortedValues[index]
	}

	fractional := int64(index % 1)
	integer := int(index - fractional)
	lower := sortedValues[integer]
	higher := sortedValues[integer+1]
	return lower + fractional*(higher-lower)
}
