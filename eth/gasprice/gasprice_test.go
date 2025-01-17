// Copyright 2020 The go-ethereum Authors
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

package gasprice

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/event"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rpc"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	valStore "gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
)

const testHead = 0

type testBackend struct {
	chain   *core.BlockChain
	pending bool // pending block available
}

func (b *testBackend) ValidatorsStorage() valStore.Storage {
	//TODO implement me
	panic("implement me")
}

func (b *testBackend) Genesis() *types.Block {
	//TODO implement me
	panic("implement me")
}

func (b *testBackend) BlockChain() *core.BlockChain {
	//TODO implement me
	panic("implement me")
}

func (b *testBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	if number > testHead {
		return nil, nil
	}
	if number == rpc.LatestBlockNumber {
		number = testHead
	}
	if number == rpc.PendingBlockNumber {
		if b.pending {
			number = testHead + 1
		} else {
			return nil, nil
		}
	}
	return b.chain.GetHeaderByNumber(uint64(number)), nil
}

func (b *testBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	if number > testHead {
		return nil, nil
	}
	if number == rpc.LatestBlockNumber {
		number = testHead
	}
	if number == rpc.PendingBlockNumber {
		if b.pending {
			number = testHead + 1
		} else {
			return nil, nil
		}
	}
	return b.chain.GetBlockByNumber(uint64(number)), nil
}

func (b *testBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.chain.GetReceiptsByHash(hash), nil
}

func (b *testBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	if b.pending {
		block := b.chain.GetBlockByNumber(testHead + 1)
		return block, b.chain.GetReceiptsByHash(block.Hash())
	}
	return nil, nil
}

func (b *testBackend) ChainConfig() *params.ChainConfig {
	return b.chain.Config()
}

func (b *testBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return nil
}

func newTestBackend(t *testing.T, londonBlock *big.Int, pending bool) *testBackend {
	depositData := make(core.DepositData, 0)
	for i := 0; i < 64; i++ {
		valData := &core.ValidatorData{
			Pubkey:            common.BytesToBlsPubKey(testutils.RandomData(96)).String(),
			CreatorAddress:    common.BytesToAddress(testutils.RandomData(20)).String(),
			WithdrawalAddress: common.BytesToAddress(testutils.RandomData(20)).String(),
			Amount:            3200,
		}

		depositData = append(depositData, valData)
	}

	var (
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		config = *params.TestChainConfig // needs copy because it is modified below
		gspec  = &core.Genesis{
			Config:     &config,
			Alloc:      core.GenesisAlloc{addr: {Balance: big.NewInt(math.MaxInt64)}},
			Validators: depositData,
			GasLimit:   30000,
			BaseFee:    big.NewInt(2),
		}
		signer = types.LatestSigner(gspec.Config)
	)
	//config.LondonBlock = londonBlock
	db := rawdb.NewMemoryDatabase()
	genesis, _ := gspec.Commit(db)

	genesisCp := &types.Checkpoint{
		Epoch:    0,
		FinEpoch: 0,
		Root:     common.Hash{},
		Spine:    genesis.Hash(),
	}
	rawdb.WriteLastCoordinatedCheckpoint(db, genesisCp)
	rawdb.WriteCoordinatedCheckpoint(db, genesisCp)
	rawdb.WriteEpoch(db, 0, genesisCp.Spine)

	bc, err := core.NewBlockChain(db, &core.CacheConfig{TrieCleanNoPrefetch: true}, &config, vm.Config{}, nil)
	if err != nil {
		t.Fatalf("Failed to create local chain, %v", err)
	}

	blocks, _ := core.GenerateChain(gspec.Config, genesis, db, testHead+1, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{1})

		var txdata types.TxData
		if londonBlock != nil && *b.Number() >= londonBlock.Uint64() {
			txdata = &types.DynamicFeeTx{
				ChainID:   gspec.Config.ChainID,
				Nonce:     b.TxNonce(addr),
				To:        &common.Address{},
				Gas:       21000,
				GasFeeCap: big.NewInt(9000000000000),
				GasTipCap: big.NewInt(int64(i + 1)),
				Data:      []byte{},
			}
		} else {
			txdata = &types.LegacyTx{
				Nonce:    b.TxNonce(addr),
				To:       &common.Address{},
				Gas:      21000,
				GasPrice: big.NewInt(int64(53518000000)),
				Value:    big.NewInt(10),
				Data:     []byte{},
			}
		}
		b.AddTxWithChain(bc, types.MustSignNewTx(key, signer, txdata))
		//b.AddTx(types.MustSignNewTx(key, signer, txdata))
	})

	// Construct testing chain
	//diskdb := rawdb.NewMemoryDatabase()
	//gspec.Commit(diskdb)
	//chain, err := core.NewBlockChain(diskdb, &core.CacheConfig{TrieCleanNoPrefetch: true}, &config, vm.Config{}, nil)
	//if err != nil {
	//	t.Fatalf("Failed to create local chain, %v", err)
	//}
	for i, bl := range blocks {
		nr := big.NewInt(int64(i)).Uint64()
		bl.SetNumber(&nr)
		fmt.Println("Nr", bl.Header().Nr())
		bc.SetLastFinalisedHeader(bl.Header(), bl.Header().Nr())
	}

	bc.InsertChain(blocks)
	return &testBackend{chain: bc, pending: pending}
}

func (b *testBackend) CurrentHeader() *types.Header {
	return b.chain.GetLastFinalizedHeader()
}

func (b *testBackend) GetBlockByNumber(number uint64) *types.Block {
	return b.chain.GetBlockByNumber(number)
}

func TestSuggestTipCap(t *testing.T) {
	config := Config{
		Blocks:     3,
		Percentile: 60,
		Default:    big.NewInt(params.GWei),
	}
	var cases = []struct {
		fork   *big.Int // London fork number
		expect *big.Int // Expected gasprice suggestion
	}{
		{nil, big.NewInt(int64(1000000000))},
		{big.NewInt(0), big.NewInt(int64(1000000000))},  // Fork point in genesis
		{big.NewInt(1), big.NewInt(int64(1000000000))},  // Fork point in first block
		{big.NewInt(32), big.NewInt(int64(1000000000))}, // Fork point in last block
		{big.NewInt(33), big.NewInt(int64(1000000000))}, // Fork point in the future
	}
	for _, c := range cases {
		backend := newTestBackend(t, c.fork, false)
		oracle := NewOracle(backend, config)

		// The gas price sampled is: 32G, 31G, 30G, 29G, 28G, 27G
		got, err := oracle.SuggestTipCap(context.Background())
		if err != nil {
			t.Fatalf("Failed to retrieve recommended gas price: %v", err)
		}
		if got.Cmp(c.expect) != 0 {
			t.Fatalf("Gas price mismatch, want %d, got %d", c.expect, got)
		}
	}
	//30000000000
	//1000000000
}
