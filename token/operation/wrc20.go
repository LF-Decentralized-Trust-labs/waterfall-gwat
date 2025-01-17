// Copyright 2024   Blue Wave Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operation

import (
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

type allowanceOperation struct {
	operation
	addressOperation
	ownerOperation
	spenderOperation
}

// NewAllowanceOperation creates a token allowance operation.
// The operation only supports WRC-20 tokens so its Standard field sets to StdWRC20.
func NewAllowanceOperation(address common.Address, owner common.Address, spender common.Address) (Allowance, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if spender == (common.Address{}) {
		return nil, ErrNoSpender
	}
	if owner == (common.Address{}) {
		return nil, ErrNoOwner
	}

	return &allowanceOperation{
		operation: operation{
			Std: StdWRC20,
		},
		addressOperation: addressOperation{
			TokenAddress: address,
		},
		ownerOperation: ownerOperation{
			OwnerAddress: owner,
		},
		spenderOperation: spenderOperation{
			SpenderAddress: spender,
		},
	}, nil
}

// Code returns op code of an allowance operation
func (op *allowanceOperation) OpCode() Code {
	return AllowanceCode
}

// UnmarshalBinary unmarshals a token allowance operation from byte encoding
func (op *allowanceOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token allowance operation to byte encoding
func (op *allowanceOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type transferOperation struct {
	operation
	valueOperation
	toOperation
}

// NewTransferOperation creates a token transfer operation
// Only WRC-20 tokens support the operation so its Standard always sets to StdWRC20.
func NewTransferOperation(to common.Address, value *big.Int) (Transfer, error) {
	return newTransferOperation(StdWRC20, to, value)
}

func newTransferOperation(standard Std, to common.Address, value *big.Int) (*transferOperation, error) {
	if to == (common.Address{}) {
		return nil, ErrNoTo
	}
	if value == nil {
		return nil, ErrNoValue
	}

	return &transferOperation{
		operation: operation{
			Std: standard,
		},
		toOperation: toOperation{
			ToAddress: to,
		},
		valueOperation: valueOperation{
			TokenValue: value,
		},
	}, nil
}

// Code returns op code of a balance of operation
func (op *transferOperation) OpCode() Code {
	return TransferCode
}

// UnmarshalBinary unmarshals a token transfer operation from byte encoding
func (op *transferOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token transfer operation to byte encoding
func (op *transferOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}
