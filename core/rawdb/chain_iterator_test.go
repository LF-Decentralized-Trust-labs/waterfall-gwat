// Copyright 2019 The go-ethereum Authors
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

package rawdb

import (
	"math/big"
	"reflect"
	"sort"
	"sync"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
)

func TestChainIterator(t *testing.T) {
	// Construct test chain db
	chainDb := NewMemoryDatabase()

	var block *types.Block
	var txs []*types.Transaction
	to := common.BytesToAddress([]byte{0x11})
	block = types.NewBlock(&types.Header{Height: uint64(0)}, nil, nil, newHasher()) // Empty genesis block
	WriteBlock(chainDb, block)
	WriteFinalizedHashNumber(chainDb, block.Hash(), block.Height())
	for i := uint64(1); i <= 10; i++ {
		var tx *types.Transaction
		if i%2 == 0 {
			tx = types.NewTx(&types.LegacyTx{
				Nonce:    i,
				GasPrice: big.NewInt(11111),
				Gas:      1111,
				To:       &to,
				Value:    big.NewInt(111),
				Data:     []byte{0x11, 0x11, 0x11},
			})
		} else {
			tx = types.NewTx(&types.AccessListTx{
				ChainID:  big.NewInt(1337),
				Nonce:    i,
				GasPrice: big.NewInt(11111),
				Gas:      1111,
				To:       &to,
				Value:    big.NewInt(111),
				Data:     []byte{0x11, 0x11, 0x11},
			})
		}
		txs = append(txs, tx)
		block = types.NewBlock(&types.Header{Height: i}, []*types.Transaction{tx}, nil, newHasher())
		WriteBlock(chainDb, block)
		WriteFinalizedHashNumber(chainDb, block.Hash(), block.Height())
	}

	var cases = []struct {
		from, to uint64
		reverse  bool
		expect   []int
	}{
		{0, 11, true, []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}},
		{0, 0, true, nil},
		{0, 5, true, []int{4, 3, 2, 1, 0}},
		{10, 11, true, []int{10}},
		{0, 11, false, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{0, 0, false, nil},
		{10, 11, false, []int{10}},
	}
	for i, c := range cases {
		var numbers []int
		hashCh := iterateTransactions(chainDb, c.from, c.to, c.reverse, nil)
		if hashCh != nil {
			for h := range hashCh {
				numbers = append(numbers, int(h.number))
				if len(h.hashes) > 0 {
					if got, exp := h.hashes[0], txs[h.number-1].Hash(); got != exp {
						t.Fatalf("block %d: hash wrong, got %x exp %x", h.number, got, exp)
					}
				}
			}
		}
		if !c.reverse {
			sort.Ints(numbers)
		} else {
			sort.Sort(sort.Reverse(sort.IntSlice(numbers)))
		}
		if !reflect.DeepEqual(numbers, c.expect) {
			t.Fatalf("Case %d failed, visit element mismatch, want %v, got %v", i, c.expect, numbers)
		}
	}
}

func TestIndexTransactions(t *testing.T) {
	// Construct test chain db
	chainDb := NewMemoryDatabase()

	var block *types.Block
	var txs []*types.Transaction
	to := common.BytesToAddress([]byte{0x11})

	// Write empty genesis block
	block = types.NewBlock(&types.Header{Height: uint64(0)}, nil, nil, newHasher())
	WriteBlock(chainDb, block)
	WriteFinalizedHashNumber(chainDb, block.Hash(), block.Height())

	for i := uint64(1); i <= 10; i++ {
		var tx *types.Transaction
		if i%2 == 0 {
			tx = types.NewTx(&types.LegacyTx{
				Nonce:    i,
				GasPrice: big.NewInt(11111),
				Gas:      1111,
				To:       &to,
				Value:    big.NewInt(111),
				Data:     []byte{0x11, 0x11, 0x11},
			})
		} else {
			tx = types.NewTx(&types.AccessListTx{
				ChainID:  big.NewInt(1337),
				Nonce:    i,
				GasPrice: big.NewInt(11111),
				Gas:      1111,
				To:       &to,
				Value:    big.NewInt(111),
				Data:     []byte{0x11, 0x11, 0x11},
			})
		}
		txs = append(txs, tx)
		block = types.NewBlock(&types.Header{Height: i}, []*types.Transaction{tx}, nil, newHasher())
		block.SetNumber(&i)
		WriteBlock(chainDb, block)
		WriteTxLookupEntry(chainDb, tx.Hash(), block.Hash())
	}
	// verify checks whether the tx indices in the range [from, to)
	// is expected.
	verify := func(from, to int, exist bool, tail uint64) {
		for i := from; i < to; i++ {
			if i == 0 {
				continue
			}
			hash := ReadTxLookupEntry(chainDb, txs[i-1].Hash())
			if exist && hash == (common.Hash{}) {
				t.Fatalf("Transaction index %d missing", i)
			}
		}
		number := ReadTxIndexTail(chainDb)
		if number == nil || *number != tail {
			t.Fatalf("Transaction tail mismatch index from %+v, to %+v, number %+v, tail %+v", from, to, *number, tail)
		}
	}
	IndexTransactions(chainDb, 5, 11, nil)
	verify(5, 11, true, 11)
	verify(0, 5, false, 11)

	IndexTransactions(chainDb, 0, 6, nil)
	verify(0, 11, true, 6)

	UnindexTransactions(chainDb, 0, 5, nil)
	verify(5, 11, true, 1)
	verify(0, 5, false, 1)

	UnindexTransactions(chainDb, 5, 2, nil)
	verify(0, 11, false, 1)

	// Testing corner cases
	signal := make(chan struct{})
	var once sync.Once
	indexTransactionsForTesting(chainDb, 5, 11, signal, func(n uint64) bool {
		if n <= 8 {
			once.Do(func() {
				close(signal)
			})
			return false
		}
		return true
	})
	verify(9, 11, true, 11)
	verify(0, 9, false, 11)
	IndexTransactions(chainDb, 0, 9, nil)

	signal = make(chan struct{})
	var once2 sync.Once
	unindexTransactionsForTesting(chainDb, 0, 11, signal, func(n uint64) bool {
		if n >= 8 {
			once2.Do(func() {
				close(signal)
			})
			return false
		}
		return true
	})
	verify(8, 11, true, 1)
	verify(0, 8, false, 1)
}
