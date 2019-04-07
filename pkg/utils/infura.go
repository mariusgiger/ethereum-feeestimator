package utils

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ybbus/jsonrpc"
)

// GetGasPriceFromInfura gets the current gas price from infura
func GetGasPriceFromInfura() (uint64, error) {
	rpcClient := jsonrpc.NewClient("https://mainnet.infura.io/")

	price := new(hexutil.Big)
	err := rpcClient.CallFor(price, "eth_gasPrice")
	if err != nil {
		return 0, err
	}

	return price.ToInt().Uint64(), nil
}
