package utils

import (
	"errors"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ybbus/jsonrpc"
	"go.uber.org/zap"
)

var (
	DefaultExpiration = 5 * time.Hour
	ErrBlockNotFound  = errors.New("block was not found")
)

type cacheItem struct {
	block      Block
	expiration int64
}

type CachedRPCClient struct {
	rpcClient  jsonrpc.RPCClient
	blockCache map[string]*cacheItem
	janitor    *janitor
	logger     *zap.Logger

	numberToHash map[int64]string //used to allow both loading by number and hash to be cached
	//TODO numberToHash should also be cleaned up

	mu sync.RWMutex
}

func NewCachedRPCClient(logger *zap.Logger) *CachedRPCClient {
	rpcClient := jsonrpc.NewClient("http://13.80.132.186:8645")
	C := &CachedRPCClient{
		rpcClient:    rpcClient,
		blockCache:   make(map[string]*cacheItem),
		mu:           sync.RWMutex{},
		logger:       logger,
		numberToHash: make(map[int64]string),
	}

	runJanitor(C, time.Minute*5)
	runtime.SetFinalizer(C, stopJanitor)

	return C
}

func (c *CachedRPCClient) getByNumber(number *big.Int) (*Block, bool) {
	c.mu.RLock()
	hash, ok := c.numberToHash[number.Int64()]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}

	return c.get(hash)
}

func (c *CachedRPCClient) get(hash string) (*Block, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, found := c.blockCache[hash]

	if found {
		return &item.block, found
	}
	return nil, found
}

func (c *CachedRPCClient) set(block *Block) {
	c.mu.Lock()
	c.numberToHash[block.Number.ToInt().Int64()] = block.Hash.String()
	expiration := time.Now().Add(DefaultExpiration).UnixNano()
	c.blockCache[block.Hash.String()] = &cacheItem{block: *block, expiration: expiration}
	c.mu.Unlock()
}

func (c *CachedRPCClient) GetBlockByHash(hash common.Hash) (*Block, error) {
	block, found := c.get(hash.String())
	if !found {
		block = new(Block)
		err := c.rpcClient.CallFor(block, "eth_getBlockByHash", hash, true)
		if err != nil {
			return nil, err
		}
		c.set(block)
	}

	return block, nil
}

func (cache *CachedRPCClient) GetLastestBlock() (*Block, error) {
	block := new(Block)
	err := cache.rpcClient.CallFor(block, "eth_getBlockByNumber", "latest", true)
	if err != nil {
		return nil, err
	}

	return block, nil
}

func (cache *CachedRPCClient) GetBlockHeaderByNumber(blockNumber *big.Int) (*BlockHeader, error) {
	header := new(BlockHeader)
	err := cache.rpcClient.CallFor(header, "eth_getBlockByNumber", hexutil.Big(*blockNumber), false)
	if err != nil {
		return nil, err
	}

	return header, nil
}

func (c *CachedRPCClient) GetBlockByNumber(blockNumber *big.Int) (*Block, error) {
	block, found := c.getByNumber(blockNumber)
	if !found {
		block = new(Block)
		err := c.rpcClient.CallFor(block, "eth_getBlockByNumber", hexutil.Big(*blockNumber), true)
		if err != nil {
			return nil, err
		}

		if block.Number == nil {
			return nil, ErrBlockNotFound
		}

		c.set(block)
	}

	return block, nil
}

// deleteExpired all expired items from the cache.
func (c *CachedRPCClient) deleteExpired() {
	c.logger.Info("deleting expired items")
	now := time.Now().UnixNano()
	c.mu.Lock()
	for k, v := range c.blockCache {
		// "Inlining" of expired
		if v.expiration > 0 && now > v.expiration {
			c.logger.Info("deleted item", zap.String("key", k))
			delete(c.blockCache, k)
		}
	}
	c.mu.Unlock()
}

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *janitor) Run(c *CachedRPCClient) {
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func stopJanitor(c *CachedRPCClient) {
	c.janitor.stop <- true
}

func runJanitor(c *CachedRPCClient, ci time.Duration) {
	j := &janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	c.janitor = j
	go j.Run(c)
}
