// Copyright 2015 The go-ethereum Authors
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

package core

import (
	"fmt"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/token"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
// Deprecated
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header()
		allLogs  []*types.Log
		gp       = new(GasPool).AddGas(block.GasLimit())
	)
	blockContext := NewEVMBlockContext(header, p.bc, nil)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, p.config, cfg)
	tokenProcessor := token.NewProcessor(blockContext, statedb)
	validatorProcessor := validator.NewProcessor(blockContext, statedb, p.bc)
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		statedb.Prepare(tx.Hash(), i)
		receipt, err := applyTransaction(tx, p.config, header, statedb, gp, usedGas, vmenv, tokenProcessor, validatorProcessor)
		if err != nil {
			log.Error(fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err).Error())
			//continue
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	header.Root = statedb.IntermediateRoot(true)

	return receipts, allLogs, *usedGas, nil
}

func applyTransaction(
	tx *types.Transaction,
	config *params.ChainConfig,
	header *types.Header,
	statedb *state.StateDB,
	gp *GasPool,
	usedGas *uint64,
	evm *vm.EVM,
	tokenProcessor *token.Processor,
	validatorProcessor *validator.Processor,
) (*types.Receipt, error) {
	txMsg, err := tx.AsMessage(types.MakeSigner(config), header.BaseFee)
	if err != nil {
		return nil, err
	}
	blockHash := header.Hash()

	// Create a new context to be used in the EVM environment.
	txContext := NewEVMTxContext(txMsg)
	evm.Reset(txContext, statedb)

	// Apply the transaction to the current state (included in the env).
	result, err := ApplyMessage(evm, tokenProcessor, validatorProcessor, txMsg, gp)
	if err != nil {
		result = &ExecutionResult{
			UsedGas:    0,
			Err:        err,
			ReturnData: nil,
		}
		addErrLog(statedb, txMsg, blockHash, err)
	} else if result.Failed() && config.IsForkSlotDelegate(header.Slot) {
		addErrLog(statedb, txMsg, blockHash, result.Err)
	}

	// Update the state with pending changes.
	var root []byte
	statedb.Finalise(true)
	*usedGas += result.UsedGas

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: *usedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = txMsg.TxHash()
	receipt.GasUsed = result.UsedGas

	// If the transaction created a contract, store the creation address in the receipt.
	if txMsg.To() == nil && !result.Failed() {
		receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, txMsg.Nonce())
	}

	// Set the receipt logs and create the bloom filter.
	receipt.Logs = statedb.GetLogs(txMsg.TxHash(), blockHash)
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = blockHash
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt, err
}

func addErrLog(statedb *state.StateDB, txMsg types.Message, blockHash common.Hash, err error) {
	if statedb.GetLogs(txMsg.TxHash(), blockHash) != nil {
		return
	}
	topics := []common.Hash{types.EvtErrorLogSignature}
	data := []byte(err.Error())
	statedb.AddLog(&types.Log{
		Address: txMsg.From(),
		Topics:  topics,
		Data:    data,
	})
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig,
	bc ChainContext,
	author *common.Address,
	gp *GasPool,
	statedb *state.StateDB,
	header *types.Header,
	tx *types.Transaction,
	usedGas *uint64,
	cfg vm.Config,
	chain *BlockChain,
) (*types.Receipt, error) {
	// Create a new context to be used in the EVM environment
	blockContext := NewEVMBlockContext(header, bc, author)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, config, cfg)

	tokenProcessor := token.NewProcessor(blockContext, statedb)
	validatorProcessor := validator.NewProcessor(blockContext, statedb, chain)
	return applyTransaction(tx, config, header, statedb, gp, usedGas, vmenv, tokenProcessor, validatorProcessor)
}
