package utils

import "math/big"

// These are the multipliers for ether denominations.
// Example: To get the wei value of an amount in 'gwei', use
//
//    new(big.Int).Mul(value, big.NewInt(params.GWei))
//
const (
	Wei     = 1
	GWei    = 1e9
	TenGWei = 1e8
	Ether   = 1e18
)

var MaxPrice = big.NewInt(500 * GWei)
