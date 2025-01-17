// Copyright 2014 The go-ethereum Authors
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
	"bytes"
	"fmt"
	"math"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/consensus/misc"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/token"
	tokenOp "gitlab.waterfall.network/waterfall/protocol/gwat/token/operation"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
)

var emptyCodeHash = crypto.Keccak256Hash(nil)

/*
The State Transitioning Model

A state transition is a change made when a transaction is applied to the current world state
The state transitioning model does all the necessary work to work out a valid new state root.

1) Nonce handling
2) Pre pay gas
3) Create a new state object if the recipient is \0*32
4) Value transfer
== If contract creation ==

	4a) Attempt to run transaction data
	4b) If valid, use result as code for the new state object

== end ==
5) Run Script section
6) Derive new state root
*/
type StateTransition struct {
	gp         *GasPool
	tp         *token.Processor
	vp         *validator.Processor
	msg        Message
	gas        uint64
	gasPrice   *big.Int
	gasFeeCap  *big.Int
	gasTipCap  *big.Int
	initialGas uint64
	value      *big.Int
	data       []byte
	state      vm.StateDB
	evm        *vm.EVM
}

// Message represents a message sent to a contract.
type Message interface {
	From() common.Address
	To() *common.Address

	GasPrice() *big.Int
	GasFeeCap() *big.Int
	GasTipCap() *big.Int
	Gas() uint64
	SetGas(gas uint64) types.Message
	Value() *big.Int

	Nonce() uint64
	IsFake() bool
	SetFake(IsFake bool) types.Message
	Data() []byte
	AccessList() types.AccessList
	TxHash() common.Hash
}

// ExecutionResult includes all output after executing given evm
// message no matter the execution itself is successful or not.
type ExecutionResult struct {
	UsedGas    uint64 // Total used gas but include the refunded gas
	Err        error  // Any error encountered during the execution(listed in core/vm/errors.go)
	ReturnData []byte // Returned data from evm(function result or data supplied with revert opcode)
}

// Unwrap returns the internal evm error which allows us for further
// analysis outside.
func (result *ExecutionResult) Unwrap() error {
	return result.Err
}

// Failed returns the indicator whether the execution is successful or not
func (result *ExecutionResult) Failed() bool { return result.Err != nil }

// Return is a helper function to help caller distinguish between revert reason
// and function return. Return returns the data after execution if no error occurs.
func (result *ExecutionResult) Return() []byte {
	if result.Err != nil {
		return nil
	}
	return common.CopyBytes(result.ReturnData)
}

// Revert returns the concrete revert reason if the execution is aborted by `REVERT`
// opcode. Note the reason can be nil if no data supplied with revert opcode.
func (result *ExecutionResult) Revert() []byte {
	if result.Err != vm.ErrExecutionReverted {
		return nil
	}
	return common.CopyBytes(result.ReturnData)
}

// IntrinsicGas computes the 'intrinsic gas' for a message with the given data.
func IntrinsicGas(data []byte, accessList types.AccessList, isContractCreation, isValidatorOp bool) (uint64, error) {
	// Set the starting gas for the raw transaction
	var gas uint64
	if isContractCreation {
		gas = params.TxGasContractCreation
	} else if isValidatorOp {
		valOp, err := operation.DecodeBytes(data)
		if err != nil {
			return 0, err
		}
		switch valOp.(type) {
		case operation.ValidatorSync:
			return 0, nil
		default:
			gas = params.TxGas
		}
	} else {
		gas = params.TxGas
	}
	// Bump the required gas by the amount of transactional data
	if len(data) > 0 {
		// Zero and non-zero bytes are priced differently
		var nz uint64
		for _, byt := range data {
			if byt != 0 {
				nz++
			}
		}
		// Make sure we don't exceed uint64 for all data combinations
		nonZeroGas := params.TxDataNonZeroGasEIP2028
		if (math.MaxUint64-gas)/nonZeroGas < nz {
			return 0, ErrGasUintOverflow
		}
		gas += nz * nonZeroGas

		z := uint64(len(data)) - nz
		if (math.MaxUint64-gas)/params.TxDataZeroGas < z {
			return 0, ErrGasUintOverflow
		}
		gas += z * params.TxDataZeroGas
	}
	if accessList != nil {
		gas += uint64(len(accessList)) * params.TxAccessListAddressGas
		gas += uint64(accessList.StorageKeys()) * params.TxAccessListStorageKeyGas
	}
	return gas, nil
}

// NewStateTransition initialises and returns a new state transition object.
func NewStateTransition(evm *vm.EVM, tokenProcessor *token.Processor, validatorProcessor *validator.Processor, msg Message, gp *GasPool) *StateTransition {
	return &StateTransition{
		gp:        gp,
		evm:       evm,
		tp:        tokenProcessor,
		vp:        validatorProcessor,
		msg:       msg,
		gasPrice:  msg.GasPrice(),
		gasFeeCap: msg.GasFeeCap(),
		gasTipCap: msg.GasTipCap(),
		value:     msg.Value(),
		data:      msg.Data(),
		state:     evm.StateDB,
	}
}

// ApplyMessage computes the new state by applying the given message
// against the old state within the environment.
//
// ApplyMessage returns the bytes returned by any EVM execution (if it took place),
// the gas used (which includes gas refunds) and an error if it failed. An error always
// indicates a core error meaning that the message would always fail for that particular
// state and would never be accepted within a block.
func ApplyMessage(evm *vm.EVM, tokenProcessor *token.Processor, validatorProcessor *validator.Processor, msg Message, gp *GasPool) (*ExecutionResult, error) {
	return NewStateTransition(evm, tokenProcessor, validatorProcessor, msg, gp).TransitionDb()
}

// to returns the recipient of the message.
func (st *StateTransition) to() common.Address {
	if st.msg == nil || st.msg.To() == nil /* contract creation */ {
		return common.Address{}
	}
	return *st.msg.To()
}

func (st *StateTransition) buyGas() error {
	txFee := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), st.evm.Context.BaseFee)

	if st.msg.GasTipCap() != nil {
		tips := st.msg.GasTipCap()
		if st.msg.GasFeeCap() != nil {
			if st.msg.GasFeeCap().Cmp(new(big.Int).Add(st.evm.Context.BaseFee, tips)) < 0 {
				tips = new(big.Int).Sub(st.msg.GasFeeCap(), st.evm.Context.BaseFee)
			}
		}

		txFee = new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), new(big.Int).Add(st.evm.Context.BaseFee, tips))
	}

	balanceCheck := txFee
	if st.gasFeeCap != nil {
		balanceCheck = new(big.Int).SetUint64(st.msg.Gas())
		balanceCheck = balanceCheck.Mul(balanceCheck, st.gasFeeCap)
		balanceCheck.Add(balanceCheck, st.value)
	}
	if have, want := st.state.GetBalance(st.msg.From()), balanceCheck; have.Cmp(want) < 0 {
		return fmt.Errorf("%w: address %v have %v want %v", ErrInsufficientFunds, st.msg.From().Hex(), have, want)
	}
	if err := st.gp.SubGas(st.msg.Gas()); err != nil {
		return err
	}
	st.gas += st.msg.Gas()

	st.initialGas = st.msg.Gas()
	st.state.SubBalance(st.msg.From(), txFee)
	return nil
}

//// PreCheck make sure this transaction's data is correct.
//func (st *StateTransition) PreCheck() error {
//	return st.preCheck()
//}

func (st *StateTransition) preCheck() error {
	// Only check transactions that are not fake
	var isValOp bool
	if st.msg.To() != nil &&
		bytes.Equal(st.evm.ChainConfig().ValidatorsStateAddress.Bytes(), st.msg.To().Bytes()) &&
		st.state.IsValidatorAddress(st.msg.From()) {
		isValOp = true
	}
	if !st.msg.IsFake() {
		// Make sure this transaction's nonce is correct.
		stNonce := st.state.GetNonce(st.msg.From())
		if msgNonce := st.msg.Nonce(); stNonce < msgNonce {
			return fmt.Errorf("%w: address %v, tx: %d state: %d, msgNonce: %v", ErrNonceTooHigh,
				st.msg.From().Hex(), msgNonce, stNonce, msgNonce)
		} else if stNonce > msgNonce {
			return fmt.Errorf("%w: address %v, tx: %d state: %d", ErrNonceTooLow,
				st.msg.From().Hex(), msgNonce, stNonce)
		}

		if !isValOp {
			if codeHash := st.state.GetCodeHash(st.msg.From()); codeHash != emptyCodeHash && codeHash != (common.Hash{}) && !st.state.IsValidatorAddress(st.msg.From()) {
				return fmt.Errorf("%w: address %v, codehash: %s", ErrSenderNoEOA,
					st.msg.From().Hex(), codeHash)
			}
		}
	}

	// Make sure that transaction gasFeeCap is greater than the baseFee (post london)

	// Skip the checks if gas fields are zero and baseFee was explicitly disabled (eth_call)
	if !st.evm.Config.NoBaseFee || st.gasFeeCap.BitLen() > 0 || st.gasTipCap.BitLen() > 0 {
		if l := st.gasFeeCap.BitLen(); l > 256 {
			return fmt.Errorf("%w: address %v, maxFeePerGas bit length: %d", ErrFeeCapVeryHigh,
				st.msg.From().Hex(), l)
		}
		if l := st.gasTipCap.BitLen(); l > 256 {
			return fmt.Errorf("%w: address %v, maxPriorityFeePerGas bit length: %d", ErrTipVeryHigh,
				st.msg.From().Hex(), l)
		}
		if st.gasFeeCap.Cmp(st.gasTipCap) < 0 {
			return fmt.Errorf("%w: address %v, maxPriorityFeePerGas: %s, maxFeePerGas: %s", ErrTipAboveFeeCap,
				st.msg.From().Hex(), st.gasTipCap, st.gasFeeCap)
		}
		// This will panic if baseFee is nil, but basefee presence is verified
		// as part of header validation.
		if st.gasFeeCap.Cmp(st.evm.Context.BaseFee) < 0 && !isValOp {
			return fmt.Errorf("%w: address %v, maxFeePerGas: %s baseFee: %s", ErrFeeCapTooLow,
				st.msg.From().Hex(), st.gasFeeCap, st.evm.Context.BaseFee)
		}
	}
	return st.buyGas()
}

// TransitionDb will transition the state by applying the current message and
// returning the evm execution result with following fields.
//
//   - used gas:
//     total gas used (including gas being refunded)
//   - returndata:
//     the returned data from evm
//   - concrete execution error:
//     various **EVM** error which aborts the execution,
//     e.g. ErrOutOfGas, ErrExecutionReverted
//
// However if any consensus issue encountered, return the error directly with
// nil evm execution result.
func (st *StateTransition) TransitionDb() (*ExecutionResult, error) {
	// First check this message satisfies all consensus rules before
	// applying the message. The rules include these clauses
	//
	// 1. the nonce of the message caller is correct
	// 2. caller has enough balance to cover transaction fee(gaslimit * gasprice)
	// 3. the amount of gas required is available in the block
	// 4. the purchased gas is enough to cover intrinsic usage
	// 5. there is no overflow when calculating intrinsic gas
	// 6. caller has enough balance to cover asset transfer for **topmost** call

	// Check clauses 1-3, buy gas if everything is correct
	if err := st.preCheck(); err != nil {
		return nil, err
	}
	msg := st.msg
	sender := vm.AccountRef(msg.From())

	// Check if it's token related operations

	txType := GetTxType(msg, st.vp, st.tp)

	isTokenOp := txType == TokenCreationTxType || txType == TokenMethodTxType
	isValidatorOp := txType == ValidatorMethodTxType || txType == ValidatorSyncTxType
	isContractCreation := txType == ContractCreationTxType

	// Check clauses 4-5, subtract intrinsic gas if everything is correct
	gas, err := IntrinsicGas(st.data, st.msg.AccessList(), isContractCreation, isValidatorOp)
	if err != nil {
		return nil, err
	}

	if txType != ValidatorSyncTxType {
		if st.gas < gas {
			return nil, fmt.Errorf("%w: have %d, want %d", ErrIntrinsicGas, st.gas, gas)
		}
		st.gas -= gas
	}

	// Check clause 6
	if msg.Value().Sign() > 0 && !st.evm.Context.CanTransfer(st.state, msg.From(), msg.Value()) {
		return nil, fmt.Errorf("%w: address %v", ErrInsufficientFundsForTransfer, msg.From().Hex())
	}

	// Set up the initial access list.
	if rules := st.evm.ChainConfig().Rules(st.evm.Context.Slot); rules.IsBerlin {
		st.state.PrepareAccessList(msg.From(), msg.To(), vm.ActivePrecompiles(rules), msg.AccessList())
	}
	var (
		ret   []byte
		vmerr error // vm errors do not effect consensus and are therefore not assigned to err
	)
	if isContractCreation {
		ret, _, st.gas, vmerr = st.evm.Create(sender, st.data, st.gas, st.value)
	} else {
		// check if "to" address belongs to token, otherwise it's a contract
		if isTokenOp {
			// perform token operation if its valid op code
			op, err := tokenOp.DecodeBytes(msg.Data())
			if err != nil {
				return nil, err
			}
			ret, vmerr = st.tp.Call(sender, st.to(), st.value, op)
		} else if isValidatorOp {
			ret, vmerr = st.vp.Call(sender, st.to(), st.value, st.msg)
		} else {
			// Increment the nonce for the next transaction
			st.state.SetNonce(msg.From(), st.state.GetNonce(sender.Address())+1)

			ret, st.gas, vmerr = st.evm.Call(sender, st.to(), st.data, st.gas, st.value)
		}
	}

	// After EIP-3529: refunds are capped to gasUsed / 5
	st.refundGas(params.RefundQuotientEIP3529)

	reward := misc.CalcCreatorReward(st.msg.Gas(), st.evm.Context.BaseFee)

	if st.msg.GasTipCap() != nil {
		tips := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), st.msg.GasTipCap())
		if st.msg.GasFeeCap() != nil {
			if st.msg.GasFeeCap().Cmp(new(big.Int).Add(st.evm.Context.BaseFee, st.msg.GasTipCap())) < 0 {
				tips = new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), new(big.Int).Sub(st.msg.GasFeeCap(), st.evm.Context.BaseFee))
			}
		}

		reward = new(big.Int).Add(reward, tips)
	}

	st.state.AddBalance(st.evm.Context.Coinbase, reward)

	return &ExecutionResult{
		UsedGas:    st.gasUsed(),
		Err:        vmerr,
		ReturnData: ret,
	}, nil
}

func (st *StateTransition) refundGas(refundQuotient uint64) {
	// Apply refund counter, capped to a refund quotient
	refund := st.gasUsed() / refundQuotient
	if refund > st.state.GetRefund() {
		refund = st.state.GetRefund()
	}
	st.gas += refund

	// Return ETH for remaining gas, exchanged at the original rate.
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(st.gas), st.gasPrice)
	st.state.AddBalance(st.msg.From(), remaining)

	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	st.gp.AddGas(st.gas)
}

// gasUsed returns the amount of gas used up by the state transition.
func (st *StateTransition) gasUsed() uint64 {
	return st.initialGas - st.gas
}

const (
	DefaultTxType TxType = iota
	TokenCreationTxType
	TokenMethodTxType
	ContractCreationTxType
	ContractMethodTxType
	ValidatorMethodTxType
	ValidatorSyncTxType
	UnknownTxType
)

type TxType uint64

func GetTxType(msg Message, vp *validator.Processor, tp *token.Processor) TxType {
	if len(msg.Data()) == 0 {
		return DefaultTxType
	}

	if msg.To() == nil {
		opCode, err := tokenOp.GetOpCode(msg.Data())
		if err == nil && opCode == tokenOp.CreateCode {
			return TokenCreationTxType
		}
		return ContractCreationTxType
	}

	if tp != nil && tp.IsToken(*msg.To()) {
		return TokenMethodTxType
	}

	if vp != nil && vp.IsValidatorOp(msg.To()) {
		valOp, err := operation.DecodeBytes(msg.Data())
		if err != nil {
			return UnknownTxType
		}
		switch valOp.(type) {
		case operation.ValidatorSync:
			return ValidatorSyncTxType
		default:
			return ValidatorMethodTxType
		}
	}

	return ContractMethodTxType
}
