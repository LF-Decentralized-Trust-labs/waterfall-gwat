// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package light implements on-demand retrieval capable state and chain objects
// for the Ethereum Light Client.
package light

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/event"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
	valStore "gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
)

var (
	bodyCacheLimit  = 256
	blockCacheLimit = 256
)

// LightChain represents a canonical chain that by default only handles block
// headers, downloading block bodies and receipts on demand through an ODR
// interface. It only does header validation during chain insertion.
type LightChain struct {
	hc            *core.HeaderChain
	indexerConfig *IndexerConfig
	chainDb       ethdb.Database
	odr           OdrBackend
	chainFeed     event.Feed
	chainHeadFeed event.Feed
	scope         event.SubscriptionScope
	genesisBlock  *types.Block

	bodyCache    *lru.Cache // Cache for the most recent block bodies
	bodyRLPCache *lru.Cache // Cache for the most recent block bodies in RLP encoded format
	blockCache   *lru.Cache // Cache for the most recent entire blocks

	slotInfo         *types.SlotInfo // coordinator slot settings
	validatorStorage valStore.Storage
	eraInfo          *era.EraInfo

	chainmu sync.RWMutex // protects header inserts
	quit    chan struct{}
	wg      sync.WaitGroup

	// Atomic boolean switches:
	running          int32 // whether LightChain is running or stopped
	procInterrupt    int32 // interrupts chain insert
	disableCheckFreq int32 // disables header verification
}

func (lc *LightChain) GetLastCoordinatedCheckpoint() *types.Checkpoint {
	//TODO implement me
	panic("implement me")
}

func (lc *LightChain) EnterNextEra(u uint64, hash common.Hash) *era.Era {
	//TODO implement me
	panic("implement me")
}

func (lc *LightChain) StartTransitionPeriod(cp *types.Checkpoint, spineRoot common.Hash) {
	//TODO implement me
	panic("implement me")
}

func (lc *LightChain) EpochToEra(epoch uint64) *era.Era {
	//TODO implement me
	panic("implement me")
}

func (lc *LightChain) GetValidatorSyncData(InitTxHash common.Hash) *types.ValidatorSync {
	//TODO implement me
	panic("implement me")
}

func (lc *LightChain) GetTransaction(txHash common.Hash) (tx *types.Transaction, blHash common.Hash, index uint64) {
	//TODO implement me
	panic("implement me")
}

func (lc *LightChain) GetTransactionReceipt(txHash common.Hash) (rc *types.Receipt, blHash common.Hash, index uint64) {
	//TODO implement me
	panic("implement me")
}

func (lc *LightChain) GetEpoch(epoch uint64) common.Hash {
	//TODO implement me
	panic("implement me")
}

func (lc *LightChain) Database() ethdb.Database {
	return lc.chainDb
}

func (lc *LightChain) GetConfig() *params.ChainConfig {
	return lc.Config()
}

func (lc *LightChain) GetEraInfo() *era.EraInfo {
	return lc.eraInfo
}

func (lc *LightChain) IsSynced() bool {
	//TODO implement me
	panic("implement me")
}

func (lc *LightChain) Synchronising() bool {
	//TODO implement me
	panic("implement me")
}

func (lc *LightChain) FinSynchronising() bool {
	//TODO implement me
	panic("implement me")
}

func (lc *LightChain) DagSynchronising() bool {
	//TODO implement me
	panic("implement me")
}

func (lc *LightChain) IsRollbackActive() bool {
	//TODO implement me
	panic("implement me")
}

func (lc *LightChain) CurrentHeader() *types.Header {
	//TODO implement me
	panic("implement me")
}

// NewLightChain returns a fully initialised light chain using information
// available in the database. It initialises the default Ethereum header
// validator.
func NewLightChain(odr OdrBackend, config *params.ChainConfig, checkpoint *params.TrustedCheckpoint) (*LightChain, error) {
	bodyCache, _ := lru.New(bodyCacheLimit)
	bodyRLPCache, _ := lru.New(bodyCacheLimit)
	blockCache, _ := lru.New(blockCacheLimit)
	chainDB := odr.Database()

	bc := &LightChain{
		chainDb:       chainDB,
		indexerConfig: odr.IndexerConfig(),
		odr:           odr,
		quit:          make(chan struct{}),
		bodyCache:     bodyCache,
		bodyRLPCache:  bodyRLPCache,
		blockCache:    blockCache,
	}
	var err error
	bc.hc, err = core.NewHeaderChain(odr.Database(), config, bc.getProcInterrupt)
	if err != nil {
		return nil, err
	}
	bc.genesisBlock, _ = bc.GetBlockByNumber(NoOdr, 0)
	if bc.genesisBlock == nil {
		return nil, core.ErrNoGenesis
	}
	if checkpoint != nil {
		bc.AddTrustedCheckpoint(checkpoint)
	}
	if err := bc.loadLastState(); err != nil {
		return nil, err
	}

	bc.validatorStorage = valStore.NewStorage(config)

	bc.SetSlotInfo(&types.SlotInfo{
		GenesisTime:    bc.genesisBlock.Time(),
		SecondsPerSlot: config.SecondsPerSlot,
		SlotsPerEpoch:  config.SlotsPerEpoch,
	})

	return bc, nil
}

// AddTrustedCheckpoint adds a trusted checkpoint to the blockchain
func (lc *LightChain) AddTrustedCheckpoint(cp *params.TrustedCheckpoint) {
	if lc.odr.ChtIndexer() != nil {
		StoreChtRoot(lc.chainDb, cp.SectionIndex, cp.SectionHead, cp.CHTRoot)
		lc.odr.ChtIndexer().AddCheckpoint(cp.SectionIndex, cp.SectionHead)
	}
	if lc.odr.BloomTrieIndexer() != nil {
		StoreBloomTrieRoot(lc.chainDb, cp.SectionIndex, cp.SectionHead, cp.BloomRoot)
		lc.odr.BloomTrieIndexer().AddCheckpoint(cp.SectionIndex, cp.SectionHead)
	}
	if lc.odr.BloomIndexer() != nil {
		lc.odr.BloomIndexer().AddCheckpoint(cp.SectionIndex, cp.SectionHead)
	}
	log.Info("Added trusted checkpoint", "block", (cp.SectionIndex+1)*lc.indexerConfig.ChtSize-1, "hash", cp.SectionHead)
}

func (lc *LightChain) getProcInterrupt() bool {
	return atomic.LoadInt32(&lc.procInterrupt) == 1
}

// Odr returns the ODR backend of the chain
func (lc *LightChain) Odr() OdrBackend {
	return lc.odr
}

// HeaderChain returns the underlying header chain.
func (lc *LightChain) HeaderChain() *core.HeaderChain {
	return lc.hc
}

// loadLastState loads the last known chain state from the database. This method
// assumes that the chain manager mutex is held.
func (lc *LightChain) loadLastState() error {
	if currHash := rawdb.ReadLastFinalizedHash(lc.chainDb); currHash == (common.Hash{}) {
		// Corrupt or empty database, init from scratch
		lc.Reset()
	} else {
		header := lc.GetHeaderByHash(currHash)
		if header == nil {
			// Corrupt or empty database, init from scratch
			lc.Reset()
		} else {
			height := rawdb.ReadLastFinalizedNumber(lc.chainDb)
			lc.hc.SetLastFinalisedHeader(header, height)
		}
	}
	// Issue a status log and return
	header := lc.hc.GetLastFinalizedHeader()
	log.Info("Loaded most recent local header", "hash", header.Hash())
	return nil
}

// SetHead rewinds the local chain to a new head. Everything above the new
// head will be deleted and the new one set.
func (lc *LightChain) SetHead(head common.Hash) error {
	lc.chainmu.Lock()
	defer lc.chainmu.Unlock()

	lc.hc.SetHead(head, nil, nil)
	return lc.loadLastState()
}

// GasLimit returns the gas limit of the current HEAD block.
func (lc *LightChain) GasLimit() uint64 {
	return lc.hc.GetLastFinalizedHeader().GasLimit
}

// Reset purges the entire blockchain, restoring it to its genesis state.
func (lc *LightChain) Reset() {
	lc.ResetWithGenesisBlock(lc.genesisBlock)
}

// ResetWithGenesisBlock purges the entire blockchain, restoring it to the
// specified genesis state.
func (lc *LightChain) ResetWithGenesisBlock(genesis *types.Block) {
	// Dump the entire block chain and purge the caches
	lc.SetHead(lc.genesisBlock.Hash())

	lc.chainmu.Lock()
	defer lc.chainmu.Unlock()

	// Prepare the genesis block and reinitialise the chain
	batch := lc.chainDb.NewBatch()
	rawdb.WriteBlock(batch, genesis)
	rawdb.WriteLastFinalizedHash(batch, genesis.Hash())
	if err := batch.Write(); err != nil {
		log.Crit("Failed to reset genesis block", "err", err)
	}

	rawdb.AddSlotBlockHash(lc.chainDb, genesis.Slot(), genesis.Hash())
	lc.genesisBlock = genesis
	lc.hc.SetGenesis(lc.genesisBlock.Header())
	lc.hc.SetLastFinalisedHeader(lc.genesisBlock.Header(), uint64(0))
}

// Accessors

// Genesis returns the genesis block
func (lc *LightChain) Genesis() *types.Block {
	return lc.genesisBlock
}

func (lc *LightChain) StateCache() state.Database {
	panic("not implemented")
}

// GetBody retrieves a block body (transactions and uncles) from the database
// or ODR service by hash, caching it if found.
func (lc *LightChain) GetBody(ctx context.Context, hash common.Hash) (*types.Body, error) {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := lc.bodyCache.Get(hash); ok {
		body := cached.(*types.Body)
		return body, nil
	}
	body, err := GetBody(ctx, lc.odr, hash)
	if err != nil {
		return nil, err
	}
	// Cache the found body for next time and return
	lc.bodyCache.Add(hash, body)
	return body, nil
}

// GetBodyRLP retrieves a block body in RLP encoding from the database or
// ODR service by hash, caching it if found.
func (lc *LightChain) GetBodyRLP(ctx context.Context, hash common.Hash) (rlp.RawValue, error) {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := lc.bodyRLPCache.Get(hash); ok {
		return cached.(rlp.RawValue), nil
	}
	body, err := GetBodyRLP(ctx, lc.odr, hash)
	if err != nil {
		return nil, err
	}
	// Cache the found body for next time and return
	lc.bodyRLPCache.Add(hash, body)
	return body, nil
}

// HasBlock checks if a block is fully present in the database or not, caching
// it if present.
func (lc *LightChain) HasBlock(hash common.Hash) bool {
	ctx := context.Background()
	blk, _ := lc.GetBlock(ctx, hash)
	return blk != nil
}

// GetBlock retrieves a block from the database or ODR service by hash and number,
// caching it if found.
func (lc *LightChain) GetBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	// Short circuit if the block's already in the cache, retrieve otherwise
	if block, ok := lc.blockCache.Get(hash); ok {
		return block.(*types.Block), nil
	}
	block, err := GetBlock(ctx, lc.odr, hash)
	if err != nil {
		return nil, err
	}
	// Cache the found block for next time and return
	lc.blockCache.Add(block.Hash(), block)
	return block, nil
}

// GetBlockByHash retrieves a block from the database or ODR service by hash,
// caching it if found.
func (lc *LightChain) GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return lc.GetBlock(ctx, hash)
}

// GetBlockByNumber retrieves a block from the database or ODR service by
// number, caching it (associated with its hash) if found.
func (lc *LightChain) GetBlockByNumber(ctx context.Context, number uint64) (*types.Block, error) {
	hash, err := GetCanonicalHash(ctx, lc.odr, number)
	if hash == (common.Hash{}) || err != nil {
		return nil, err
	}
	return lc.GetBlock(ctx, hash)
}

// Stop stops the blockchain service. If any imports are currently in progress
// it will abort them using the procInterrupt.
func (lc *LightChain) Stop() {
	if !atomic.CompareAndSwapInt32(&lc.running, 0, 1) {
		return
	}
	close(lc.quit)
	lc.StopInsert()
	lc.wg.Wait()
	log.Info("Blockchain stopped")
}

// StopInsert interrupts all insertion methods, causing them to return
// errInsertionInterrupted as soon as possible. Insertion is permanently disabled after
// calling this method.
func (lc *LightChain) StopInsert() {
	atomic.StoreInt32(&lc.procInterrupt, 1)
}

// Rollback is designed to remove a chain of links from the database that aren't
// certain enough to be valid.
func (lc *LightChain) Rollback(chain []common.Hash) {
	lc.chainmu.Lock()
	defer lc.chainmu.Unlock()

	batch := lc.chainDb.NewBatch()
	for i := len(chain) - 1; i >= 0; i-- {
		hash := chain[i]

		// Degrade the chain markers if they are explicitly reverted.
		// In theory we should update all in-memory markers in the
		// last step, however the direction of rollback is from high
		// to low, so it's safe the update in-memory markers directly.
		if head := lc.hc.GetLastFinalizedHeader(); head.Hash() == hash {
			height := rawdb.ReadFinalizedNumberByHash(lc.chainDb, head.Hash())
			if height == nil || *height == uint64(0) {
				continue
			}
			prevHeight := *height - 1
			prevHash := rawdb.ReadFinalizedHashByNumber(lc.chainDb, prevHeight)
			rawdb.WriteLastFinalizedHash(batch, prevHash)
			prevHeader := lc.GetHeaderByHash(prevHash)
			lc.hc.SetLastFinalisedHeader(prevHeader, prevHeight)
		}
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to rollback light chain", "error", err)
	}
}

// postChainEvents iterates over the events generated by a chain insertion and
// posts them into the event feed.
func (lc *LightChain) postChainEvents(events []interface{}) {
	for _, event := range events {
		switch ev := event.(type) {
		case core.ChainEvent:
			if lc.GetLastFinalizedHeader().Hash() == ev.Hash {
				lc.chainHeadFeed.Send(core.ChainHeadEvent{Block: ev.Block})
			}
			lc.chainFeed.Send(ev)
		default:
			log.Warn("Unsupported event")
		}
	}
}

// InsertHeaderChain attempts to insert the given header chain in to the local
// chain, possibly creating a reorg. If an error is returned, it will return the
// index number of the failing header as well an error describing what went wrong.
//
// The verify parameter can be used to fine tune whether nonce verification
// should be done or not. The reason behind the optional check is because some
// of the header retrieval mechanisms already need to verfy nonces, as well as
// because nonces can be verified sparsely, not needing to check each.
//
// In the case of a light chain, InsertHeaderChain also creates and posts light
// chain events when necessary.
func (lc *LightChain) InsertHeaderChain(chain []*types.Header) (int, error) {
	start := time.Now()
	if i, err := lc.hc.ValidateHeaderChain(chain); err != nil {
		return i, err
	}

	// Make sure only one thread manipulates the chain at once
	lc.chainmu.Lock()
	defer lc.chainmu.Unlock()

	lc.wg.Add(1)
	defer lc.wg.Done()

	status, err := lc.hc.InsertHeaderChain(chain, start)
	if err != nil || len(chain) == 0 {
		return 0, err
	}

	// Create chain event for the new head block of this insertion.
	var (
		events     = make([]interface{}, 0, 1)
		lastHeader = chain[len(chain)-1]
		block      = types.NewBlockWithHeader(lastHeader)
	)
	switch status {
	case core.CanonStatTy:
		events = append(events, core.ChainEvent{Block: block, Hash: block.Hash()})
	case core.SideStatTy:
		events = append(events, core.ChainSideEvent{Block: block})
	}
	lc.postChainEvents(events)

	return 0, err
}

// GetLastFinalizedHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (lc *LightChain) GetLastFinalizedHeader() *types.Header {
	return lc.hc.GetLastFinalizedHeader()
}

func (lc *LightChain) ReadFinalizedNumberByHash(hash common.Hash) *uint64 {
	return lc.hc.GetBlockFinalizedNumber(hash)
}

// GetBlockFinalizedNumber retrieves a block finalized height
func (lc *LightChain) GetBlockFinalizedNumber(hash common.Hash) *uint64 {
	return lc.hc.GetBlockFinalizedNumber(hash)
}

// GetHeader retrieves a block header from the database by hash and number,
// caching it if found.
func (lc *LightChain) GetHeader(hash common.Hash) *types.Header {
	return lc.hc.GetHeader(hash)
}

// GetHeaderByHash retrieves a block header from the database by hash, caching it if
// found.
func (lc *LightChain) GetHeaderByHash(hash common.Hash) *types.Header {
	return lc.hc.GetHeaderByHash(hash)
}

// GetHeaderByHash retrieves a block header from the database by hash, caching it if
// found.
func (lc *LightChain) GetHeadersByHashes(hashes common.HashArray) types.HeaderMap {
	return lc.hc.GetHeadersByHashes(hashes)
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (lc *LightChain) HasHeader(hash common.Hash) bool {
	return lc.hc.HasHeader(hash)
}

// GetCanonicalHash returns the canonical hash for a given block number
func (bc *LightChain) GetCanonicalHash(number uint64) common.Hash {
	return bc.hc.GetCanonicalHash(number)
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (lc *LightChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {
	return lc.hc.GetBlockHashesFromHash(hash, max)
}

// GetAncestor retrieves the Nth ancestor of a given block. It assumes that either the given block or
// a close ancestor of it is canonical. maxNonCanonical points to a downwards counter limiting the
// number of blocks to be individually checked before we reach the canonical chain.
//
// Note: ancestor == 0 returns the same block, 1 returns its parent and so on.
func (lc *LightChain) GetAncestor(hash common.Hash, number, ancestor uint64, maxNonCanonical *uint64) (common.Hash, uint64) {
	return lc.hc.GetAncestor(hash, number, ancestor, maxNonCanonical)
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (lc *LightChain) GetHeaderByNumber(number uint64) *types.Header {
	return lc.hc.GetHeaderByNumber(number)
}

// GetHeaderByNumberOdr retrieves a block header from the database or network
// by number, caching it (associated with its hash) if found.
func (lc *LightChain) GetHeaderByNumberOdr(ctx context.Context, number uint64) (*types.Header, error) {
	if header := lc.hc.GetHeaderByNumber(number); header != nil {
		return header, nil
	}
	return GetHeaderByNumber(ctx, lc.odr, number)
}

// Config retrieves the header chain's chain configuration.
func (lc *LightChain) Config() *params.ChainConfig { return lc.hc.Config() }

// SyncCheckpoint fetches the checkpoint point block header according to
// the checkpoint provided by the remote peer.
//
// Note if we are running the clique, fetches the last epoch snapshot header
// which covered by checkpoint.
func (lc *LightChain) SyncCheckpoint(ctx context.Context, checkpoint *params.TrustedCheckpoint) bool {
	// Ensure the remote checkpoint head is ahead of us
	head := lc.GetLastFinalizedHeader().Nr()

	latest := (checkpoint.SectionIndex+1)*lc.indexerConfig.ChtSize - 1
	//if clique := lc.hc.Config().Clique; clique != nil {
	//	//latest -= latest % clique.Epoch // epoch snapshot for clique
	//}
	if head >= latest {
		return true
	}
	// Retrieve the latest useful header and update to it
	if header, err := GetHeaderByNumber(ctx, lc.odr, latest); header != nil && err == nil {
		lc.chainmu.Lock()
		defer lc.chainmu.Unlock()

		// Ensure the chain didn't move past the latest block while retrieving it
		if lc.hc.GetLastFinalizedHeader().Nr() < header.Nr() {
			log.Info("Updated latest header based on CHT", "number", header.Number, "hash", header.Hash(), "age", common.PrettyAge(time.Unix(int64(header.Time), 0)))
			rawdb.WriteLastFinalizedHash(lc.chainDb, header.Hash())
			lc.hc.SetLastFinalisedHeader(header, header.Nr())
		}
		return true
	}
	return false
}

// LockChain locks the chain mutex for reading so that multiple canonical hashes can be
// retrieved while it is guaranteed that they belong to the same version of the chain
func (lc *LightChain) LockChain() {
	lc.chainmu.RLock()
}

// UnlockChain unlocks the chain mutex
func (lc *LightChain) UnlockChain() {
	lc.chainmu.RUnlock()
}

// SubscribeChainEvent registers a subscription of ChainEvent.
func (lc *LightChain) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return lc.scope.Track(lc.chainFeed.Subscribe(ch))
}

// SubscribeChainHeadEvent registers a subscription of ChainHeadEvent.
func (lc *LightChain) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return lc.scope.Track(lc.chainHeadFeed.Subscribe(ch))
}

// SubscribeLogsEvent implements the interface of filters.Backend
// LightChain does not send logs events, so return an empty subscription.
func (lc *LightChain) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return lc.scope.Track(new(event.Feed).Subscribe(ch))
}

// SubscribeRemovedLogsEvent implements the interface of filters.Backend
// LightChain does not send core.RemovedLogsEvent, so return an empty subscription.
func (lc *LightChain) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return lc.scope.Track(new(event.Feed).Subscribe(ch))
}

// DisableCheckFreq disables header validation. This is used for ultralight mode.
func (lc *LightChain) DisableCheckFreq() {
	atomic.StoreInt32(&lc.disableCheckFreq, 1)
}

// EnableCheckFreq enables header validation.
func (lc *LightChain) EnableCheckFreq() {
	atomic.StoreInt32(&lc.disableCheckFreq, 0)
}

func (lc *LightChain) ValidatorStorage() valStore.Storage {
	return lc.validatorStorage
}

// SetSlotInfo set new slot info.
func (lc *LightChain) SetSlotInfo(si *types.SlotInfo) error {
	if si == nil {
		return core.ErrBadSlotInfo
	}
	lc.slotInfo = si.Copy()
	return nil
}

// GetSlotInfo get current slot info.
func (lc *LightChain) GetSlotInfo() *types.SlotInfo {
	return lc.slotInfo.Copy()
}

func (lc *LightChain) GetCoordinatedCheckpointEpoch(epoch uint64) uint64 {
	if epoch >= 2 {
		epoch = epoch - 2
	}

	return epoch
}

// StateAt returns a new mutable state based on a particular point in time.
func (lc *LightChain) StateAt(root common.Hash) (*state.StateDB, error) {
	//TODO implement me
	panic("implement me")
}
