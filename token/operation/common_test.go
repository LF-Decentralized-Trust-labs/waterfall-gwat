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
	"errors"
	"fmt"
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestApproveOperation(t *testing.T) {
	type decodedOp struct {
		op      Std
		spender common.Address
		value   *big.Int
	}

	cases := []operationTestCase{
		{
			caseName: "CorrectWRC721",
			decoded: decodedOp{
				op:      StdWRC721,
				value:   opId,
				spender: opSpender,
			},
			encoded: []byte{
				243, 13, 248, 137, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 130, 48, 57, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "CorrectWRC20",
			decoded: decodedOp{
				op:      StdWRC20,
				value:   opValue,
				spender: opSpender,
			},
			encoded: []byte{
				243, 13, 248, 136, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "EmptyValue",
			decoded: decodedOp{
				op:      0,
				value:   nil,
				spender: opSpender,
			},
			encoded: []byte{
				243, 13, 248, 133, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{ErrNoValue},
		},
		{
			caseName: "EmptySpender",
			decoded: decodedOp{
				op:      StdWRC721,
				value:   opId,
				spender: common.Address{},
			},
			encoded: []byte{
				243, 13, 248, 137, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 130, 48, 57, 128, 128, 128,
			},
			errs: []error{ErrNoSpender},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		op, err := NewApproveOperation(o.op, o.spender, o.value)
		if err != nil {
			return err
		}

		return equalOpBytes(op, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(Approve)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, o.op)
		if err != nil {
			return err
		}

		operator := opDecoded.Spender()
		if operator != o.spender {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.spender, operator)
		}

		value := opDecoded.Value()
		if !testutils.BigIntEquals(value, o.value) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", value, o.value)
		}

		if len(value.Bytes()) == 0 {
			// just stub for encoding tests
			return ErrNoValue
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestBalanceOfOperation(t *testing.T) {
	type decodedOp struct {
		op      Std
		address common.Address
		owner   common.Address
	}

	cases := []operationTestCase{
		{
			caseName: "Correct WRC20 test",
			decoded: decodedOp{
				op:      StdWRC20,
				address: opAddress,
				owner:   opTo,
			},
			encoded: []byte{
				243, 34, 248, 133, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "Correct WRC721 test",
			decoded: decodedOp{
				op:      StdWRC721,
				address: opAddress,
				owner:   opTo,
			},
			encoded: []byte{
				243, 34, 248, 133, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "No empty owner",
			decoded: decodedOp{
				op:      StdWRC20,
				address: opAddress,
				owner:   common.Address{},
			},
			encoded: []byte{
				243, 34, 248, 133, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{ErrNoOwner},
		},
		{
			caseName: "No empty token address",
			decoded: decodedOp{
				op:      0,
				address: common.Address{},
				owner:   opTo,
			},
			encoded: []byte{
				243, 34, 248, 133, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{ErrNoAddress},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewBalanceOfOperation(o.address, o.owner)
		if err != nil {
			return err
		}

		return equalOpBytes(op, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(BalanceOf)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, 0)
		if err != nil {
			return err
		}

		if o.address != opDecoded.Address() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.Address(), o.address)
		}

		if o.owner != opDecoded.Owner() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.Owner(), o.owner)
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestPropertiesOperation(t *testing.T) {
	type decodedOp struct {
		address common.Address
		id      *big.Int
	}

	cases := []operationTestCase{
		{
			caseName: "Correct test",
			decoded: decodedOp{
				address: opAddress,
				id:      opId,
			},
			encoded: []byte{
				243, 33, 248, 135, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "Empty token id",
			decoded: decodedOp{
				address: opAddress,
				id:      nil,
			},
			encoded: []byte{
				243, 33, 248, 133, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "No empty token address",
			decoded: decodedOp{
				address: common.Address{},
				id:      opId,
			},
			encoded: []byte{
				243, 33, 248, 135, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{ErrNoAddress},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewPropertiesOperation(o.address, o.id)
		if err != nil {
			return err
		}

		return equalOpBytes(op, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(Properties)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, 0)
		if err != nil {
			return err
		}

		tokenId, ok := opDecoded.TokenId()
		if !ok {
			return errors.New("invalid tokenId")
		}

		if !testutils.BigIntEquals(tokenId, o.id) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", tokenId, o.id)
		}

		if o.address != opDecoded.Address() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", o.address, opDecoded.Address())
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestTransferFromOperation(t *testing.T) {
	type decodedOp struct {
		op    Std
		to    common.Address
		value *big.Int
		from  common.Address
	}

	cases := []operationTestCase{
		{
			caseName: "Correct WRC20 test",
			decoded: decodedOp{
				op:    StdWRC20,
				to:    opTo,
				value: opValue,
				from:  opFrom,
			},
			encoded: []byte{
				243, 31, 248, 136, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "Correct WRC721 test",
			decoded: decodedOp{
				op:    StdWRC721,
				to:    opTo,
				value: opId,
				from:  opFrom,
			},
			encoded: []byte{
				243, 31, 248, 137, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 130, 48, 57, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "No empty from",
			decoded: decodedOp{
				op:    StdWRC721,
				to:    opTo,
				value: opValue,
				from:  common.Address{},
			},
			encoded: []byte{
				243, 31, 248, 138, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{ErrNoFrom},
		},
		{
			caseName: "No empty to",
			decoded: decodedOp{
				op:    StdWRC20,
				to:    common.Address{},
				value: opValue,
				from:  opFrom,
			},
			encoded: []byte{
				243, 31, 248, 136, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{ErrNoTo},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewTransferFromOperation(o.op, o.from, o.to, o.value)
		if err != nil {
			return err
		}

		return equalOpBytes(op, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(TransferFrom)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, o.op)
		if err != nil {
			return err
		}

		if !testutils.BigIntEquals(opDecoded.Value(), o.value) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.Value(), o.value)
		}

		if o.from != opDecoded.From() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.From(), o.from)
		}

		if o.to != opDecoded.To() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.To(), o.to)
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestTokenSetPrice(t *testing.T) {
	type decodedOp struct {
		op      Std
		tokenId *big.Int
		value   *big.Int
	}

	cases := []operationTestCase{
		{
			caseName: "Correct",
			decoded: decodedOp{
				op:      0,
				tokenId: opId,
				value:   opValue,
			},
			encoded: []byte{
				243, 42, 248, 138, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "EmptyValue",
			decoded: decodedOp{
				op:      0,
				tokenId: opId,
				value:   nil,
			},
			encoded: []byte{
				243, 42, 248, 135, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{ErrNoValue},
		},
		{
			caseName: "EmptyId",
			decoded: decodedOp{
				op:      0,
				tokenId: nil,
				value:   opValue,
			},
			encoded: []byte{
				243, 42, 248, 136, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "NegativeValue",
			decoded: decodedOp{
				op:      0,
				tokenId: opId,
				value:   big.NewInt(-10),
			},
			encoded: []byte{
				243, 42, 248, 135, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 10, 128, 128, 128,
			},
			errs: []error{ErrNegativeCost},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewSetPriceOperation(o.tokenId, o.value)
		if err != nil {
			return err
		}

		return equalOpBytes(op, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(SetPrice)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, 0)
		if err != nil {
			return err
		}

		tokenId, _ := opDecoded.TokenId()
		if !testutils.BigIntEquals(tokenId, o.tokenId) {
			return fmt.Errorf("id do not match:\nwant: %+v\nhave: %+v", tokenId, o.tokenId)
		}

		if o.value != nil && o.value.Sign() < 0 {
			// Encoder cannot encode negative value, just stub
			return ErrNegativeCost
		}

		value := opDecoded.Value()
		if !testutils.BigIntEquals(value, o.value) {
			return fmt.Errorf("newValues do not match:\nhave: %+v\nwant: %+v", value, o.value)
		}

		if len(value.Bytes()) == 0 {
			// just stub for encoding tests
			return ErrNoValue
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestBuy(t *testing.T) {
	type decodedOp struct {
		op      Std
		tokenId *big.Int
		newCost *big.Int
	}

	cases := []operationTestCase{
		{
			caseName: "Correct",
			decoded: decodedOp{
				op:      0,
				tokenId: opId,
				newCost: opValue,
			},
			encoded: []byte{
				243, 43, 248, 138, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 131, 1, 182, 105, 128, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "EmptyTokenId",
			decoded: decodedOp{
				op:      0,
				tokenId: nil,
				newCost: opValue,
			},
			encoded: []byte{
				243, 43, 248, 136, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 131, 1, 182, 105, 128, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "EmptyNewCost",
			decoded: decodedOp{
				op:      0,
				tokenId: opId,
				newCost: nil,
			},
			encoded: []byte{
				243, 43, 248, 135, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "NegativeNewCost",
			decoded: decodedOp{
				op:      0,
				tokenId: opId,
				newCost: big.NewInt(-10),
			},
			encoded: []byte{
				243, 43, 248, 135, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 128, 128, 128, 128,
			},
			errs: []error{ErrNegativeCost},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewBuyOperation(o.tokenId, o.newCost)
		if err != nil {
			return err
		}

		return equalOpBytes(op, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(Buy)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, 0)
		if err != nil {
			return err
		}

		tokenId, _ := opDecoded.TokenId()
		if !testutils.BigIntEquals(tokenId, o.tokenId) {
			return fmt.Errorf("id do not match:\nwant: %+v\nhave: %+v", tokenId, o.tokenId)
		}

		if o.newCost != nil && o.newCost.Sign() < 0 {
			// Encoder cannot encode negative value, just stub
			return ErrNegativeCost
		}

		newValue, _ := opDecoded.NewCost()
		if !testutils.BigIntEquals(newValue, o.newCost) {
			return fmt.Errorf("newValues do not match:\nwant: %+v\nhave: %+v", newValue, o.newCost)
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestCost(t *testing.T) {
	type decodedOp struct {
		op      Std
		address common.Address
		tokenId *big.Int
	}

	cases := []operationTestCase{
		{
			caseName: "Correct",
			decoded: decodedOp{
				op:      0,
				tokenId: opId,
				address: opAddress,
			},
			encoded: []byte{
				243, 44, 248, 135, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "EmptyId",
			decoded: decodedOp{
				op:      0,
				tokenId: nil,
				address: opAddress,
			},
			encoded: []byte{
				243, 44, 248, 133, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "EmptyAddress",
			decoded: decodedOp{
				op:      0,
				tokenId: opId,
				address: common.Address{},
			},
			encoded: []byte{
				243, 44, 248, 135, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{ErrNoAddress},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewCostOperation(o.address, o.tokenId)
		if err != nil {
			return err
		}

		return equalOpBytes(op, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(Cost)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, 0)
		if err != nil {
			return err
		}

		if o.address != opDecoded.Address() {
			t.Fatalf("addresses do not match:\nwant: %+v\nhave: %+v", opDecoded.Address(), o.address)
		}

		tokenId, _ := opDecoded.TokenId()
		if !testutils.BigIntEquals(tokenId, o.tokenId) {
			return fmt.Errorf("id do not match:\nwant: %+v\nhave: %+v", tokenId, o.tokenId)
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}
