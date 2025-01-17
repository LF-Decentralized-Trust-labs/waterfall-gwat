// Copyright 2021 The go-ethereum Authors
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

package eth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/token"
	"gitlab.waterfall.network/waterfall/protocol/gwat/trie"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator"
)

// Parameters:
//   - block: The block for which we want the state (== state at the stateRoot of the parent)
//   - reexec: The maximum number of blocks to reprocess trying to obtain the desired state
//   - base: If the caller is tracing multiple blocks, the caller can provide the parent state
//     continuously from the callsite.
//   - checklive: if true, then the live 'blockchain' state database is used. If the caller want to
//     perform Commit or other 'save-to-disk' changes, this should be set to false to avoid
//     storing trash persistently
//   - preferDisk: this arg can be used by the caller to signal that even though the 'base' is provided,
//     it would be preferrable to start from a fresh state, if we have it on disk.
func (eth *Ethereum) stateAtBlock(block *types.Block, reexec uint64, base *state.StateDB, checkLive bool, preferDisk bool) (statedb *state.StateDB, err error) {
	if block.Number() == nil {
		return statedb, nil
	}

	var (
		current  *types.Block
		database state.Database
		report   = true
		origin   = block.Nr()
	)
	// Check the live database first if we have the state fully available, use that.
	if checkLive {
		statedb, err = eth.blockchain.StateAt(block.Root())
		if err == nil {
			return statedb, nil
		}
	}
	if base != nil {
		if preferDisk {
			// Create an ephemeral trie.Database for isolating the live one. Otherwise
			// the internal junks created by tracing will be persisted into the disk.
			database = state.NewDatabaseWithConfig(eth.chainDb, &trie.Config{Cache: 16})
			if statedb, err = state.New(block.Root(), database, nil); err == nil {
				log.Info("Found disk backend for state trie", "root", block.Root(), "number", block.Number())
				return statedb, nil
			}
		}
		// The optional base statedb is given, mark the start point as parent block
		statedb, database, report = base, base.Database(), false
		current = eth.blockchain.GetBlockByNumber(block.Nr() - 1)
	} else {
		// Otherwise try to reexec blocks until we find a state or reach our limit
		current = block

		// Create an ephemeral trie.Database for isolating the live one. Otherwise
		// the internal junks created by tracing will be persisted into the disk.
		database = state.NewDatabaseWithConfig(eth.chainDb, &trie.Config{Cache: 16})

		// If we didn't check the dirty database, do check the clean one, otherwise
		// we would rewind past a persisted block (specific corner case is chain
		// tracing from the genesis).
		if !checkLive {
			statedb, err = state.New(current.Root(), database, nil)
			if err == nil {
				return statedb, nil
			}
		}
		// Database does not have the state for the given block, try to regenerate
		for i := uint64(0); i < reexec; i++ {
			if current.Height() == 0 {
				return nil, errors.New("genesis state is missing")
			}
			parent := eth.blockchain.GetBlockByNumber(current.Nr() - 1)
			if parent == nil {
				return nil, fmt.Errorf("missing block %v %d", current.ParentHashes(), current.Nr()-1)
			}
			current = parent

			statedb, err = state.New(current.Root(), database, nil)
			if err == nil {
				break
			}
		}
		if err != nil {
			switch err.(type) {
			case *trie.MissingNodeError:
				return nil, fmt.Errorf("required historical state unavailable (reexec=%d)", reexec)
			default:
				return nil, err
			}
		}
	}
	// State was available at historical point, regenerate
	var (
		start  = time.Now()
		logged time.Time
		parent common.Hash
	)
	for current.Nr() < origin {
		// Print progress logs if long enough time elapsed
		if time.Since(logged) > 8*time.Second && report {
			log.Info("Regenerating historical state", "block", current.Nr()+1, "target", origin, "remaining", origin-current.Nr()-1, "elapsed", time.Since(start))
			logged = time.Now()
		}
		// Retrieve the next block to regenerate and process it
		next := current.Nr() + 1
		if current = eth.blockchain.GetBlockByNumber(next); current == nil {
			return nil, fmt.Errorf("block #%d not found", next)
		}
		_, _, _, err := eth.blockchain.Processor().Process(current, statedb, vm.Config{})
		if err != nil {
			return nil, fmt.Errorf("processing block %d failed: %v", current.Nr(), err)
		}
		// Finalize the state so any modifications are written to the trie
		root, err := statedb.Commit(true)
		if err != nil {
			return nil, fmt.Errorf("stateAtBlock commit failed, number %d root %v: %w",
				current.Nr(), current.Root().Hex(), err)
		}
		statedb, err = state.New(root, database, nil)
		if err != nil {
			return nil, fmt.Errorf("state reset after block %d failed: %v", current.Nr(), err)
		}
		database.TrieDB().Reference(root, common.Hash{})
		if parent != (common.Hash{}) {
			database.TrieDB().Dereference(parent)
		}
		parent = root
	}
	if report {
		nodes, imgs := database.TrieDB().Size()
		log.Info("Historical state regenerated", "block", current.Nr(), "elapsed", time.Since(start), "nodes", nodes, "preimages", imgs)
	}
	return statedb, nil
}

// stateAtTransaction returns the execution environment of a certain transaction.
func (eth *Ethereum) stateAtTransaction(block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, error) {
	// Short circuit if it's genesis block.
	if block.Height() == 0 {
		return nil, vm.BlockContext{}, nil, errors.New("no transaction in genesis")
	}
	// Create the parent state database
	var parent *types.Block
	ctx := context.Background()
	for _, h := range block.ParentHashes() {
		pb := eth.blockchain.GetBlock(ctx, h)
		if pb != nil && pb.Nr() == pb.Height() {
			parent = pb
			break
		}
	}
	if parent == nil {
		return nil, vm.BlockContext{}, nil, fmt.Errorf("parent %v not found", block.ParentHashes())
	}
	// Lookup the statedb of parent block from the live database,
	// otherwise regenerate it on the flight.
	statedb, err := eth.stateAtBlock(parent, reexec, nil, true, false)
	if err != nil {
		return nil, vm.BlockContext{}, nil, err
	}
	if txIndex == 0 && len(block.Transactions()) == 0 {
		return nil, vm.BlockContext{}, statedb, nil
	}
	// Recompute transactions up to the target index.
	signer := types.MakeSigner(eth.blockchain.Config())
	for idx, tx := range block.Transactions() {
		// Assemble the transaction call message and return if the requested offset
		msg, _ := tx.AsMessage(signer, block.BaseFee())
		txContext := core.NewEVMTxContext(msg)
		context := core.NewEVMBlockContext(block.Header(), eth.blockchain, nil)
		if idx == txIndex {
			return msg, context, statedb, nil
		}
		// Not yet the searched for transaction, execute on top of the current state
		vmenv := vm.NewEVM(context, txContext, statedb, eth.blockchain.Config(), vm.Config{})
		tp := token.NewProcessor(context, statedb)
		vp := validator.NewProcessor(context, statedb, eth.BlockChain())
		statedb.Prepare(tx.Hash(), idx)
		if _, err := core.ApplyMessage(vmenv, tp, vp, msg, new(core.GasPool).AddGas(tx.Gas())); err != nil {
			return nil, vm.BlockContext{}, nil, fmt.Errorf("transaction %#x failed: %v", tx.Hash(), err)
		}
		// Ensure any modifications are committed to the state
		// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
		statedb.Finalise(true)
	}
	return nil, vm.BlockContext{}, nil, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash())
}
