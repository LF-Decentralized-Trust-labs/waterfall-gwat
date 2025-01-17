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

package token

import (
	"fmt"
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/token/operation"
	"gitlab.waterfall.network/waterfall/protocol/gwat/token/testmodels"
)

var (
	stateDb        *state.StateDB
	processor      *Processor
	wrc20Address   common.Address
	wrc721Address  common.Address
	caller         Ref
	operator       common.Address
	address        common.Address
	spender        common.Address
	owner          common.Address
	seller         common.Address
	buyer          common.Address
	to             common.Address
	approveAddress common.Address
	value          *big.Int
	id             *big.Int
	id2            *big.Int
	id3            *big.Int
	id4            *big.Int
	id5            *big.Int
	id6            *big.Int
	id7            *big.Int
	totalSupply    *big.Int
	decimals       uint8
	percentFee     uint8
	name           []byte
	symbol         []byte
	baseURI        []byte
	data           []byte
	zeroBig        = big.NewInt(0)
)

func init() {
	stateDb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	ctx := vm.BlockContext{
		CanTransfer: nil,
		Transfer:    nil,
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).SetUint64(8000000),
		Time:        new(big.Int).SetUint64(5),
		Difficulty:  big.NewInt(0x30000),
		GasLimit:    uint64(6000000),
	}

	processor = NewProcessor(ctx, stateDb)

	operator = common.BytesToAddress(testutils.RandomData(20))
	address = common.BytesToAddress(testutils.RandomData(20))
	spender = common.BytesToAddress(testutils.RandomData(20))
	owner = common.BytesToAddress(testutils.RandomData(20))
	seller = common.BytesToAddress(testutils.RandomData(20))
	buyer = common.BytesToAddress(testutils.RandomData(20))
	to = common.BytesToAddress(testutils.RandomData(20))
	approveAddress = common.BytesToAddress(testutils.RandomData(20))

	caller = vm.AccountRef(owner)

	value = big.NewInt(int64(testutils.RandomInt(10, 30)))
	id = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	id2 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	id3 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	id4 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	id5 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	id6 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	id7 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	totalSupply = big.NewInt(int64(testutils.RandomInt(100, 1000)))

	decimals = uint8(testutils.RandomInt(0, 255))
	percentFee = uint8(testutils.RandomInt(0, 100))

	name = testutils.RandomStringInBytes(testutils.RandomInt(10, 20))
	symbol = testutils.RandomStringInBytes(testutils.RandomInt(5, 8))
	baseURI = testutils.RandomStringInBytes(testutils.RandomInt(20, 40))

	data = testutils.RandomData(testutils.RandomInt(20, 50))
}

func TestProcessorCreateOperationWRC20Call(t *testing.T) {
	createOpWrc20, err := operation.NewWrc20CreateOperation(name, symbol, &decimals, totalSupply)
	if err != nil {
		t.Fatal(err)
	}

	cases := []testmodels.TestCase{
		{
			CaseName: "Correct test WRC20",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: common.Address{},
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				adr := call(t, v.Caller, v.TokenAddress, nil, createOpWrc20, c.Errs)
				*a = common.BytesToAddress(adr)

				balance := checkBalance(t, wrc20Address, owner)
				if balance.Cmp(totalSupply) != 0 {
					t.Fatal()
				}
			},
		},
		{
			CaseName: "WRC20 non empty token address",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: address,
			},
			Errs: []error{ErrNotNilTo},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				call(t, v.Caller, v.TokenAddress, nil, createOpWrc20, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &wrc20Address)
		})
	}
}

func TestProcessorCreateOperationWRC721Call(t *testing.T) {
	createOpWrc721, err := operation.NewWrc721CreateOperation(name, symbol, baseURI, &percentFee)
	if err != nil {
		t.Fatal(err)
	}
	cases := []testmodels.TestCase{
		{
			CaseName: "Correct test WRC721",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: common.Address{},
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				adr := call(t, v.Caller, v.TokenAddress, nil, createOpWrc721, c.Errs)
				*a = common.BytesToAddress(adr)
			},
		},
		{
			CaseName: "WRC721 non empty token address",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: address,
			},
			Errs: []error{ErrNotNilTo},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				call(t, v.Caller, v.TokenAddress, nil, createOpWrc721, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &wrc721Address)
		})
	}
}

func TestProcessorTransferFromOperationCall(t *testing.T) {
	opWrc20, err := operation.NewTransferFromOperation(operation.StdWRC20, owner, to, value)
	if err != nil {
		t.Fatal(err)
	}
	opWrc721, err := operation.NewTransferFromOperation(operation.StdWRC721, owner, to, id)
	if err != nil {
		t.Fatal(err)
	}

	cases := []testmodels.TestCase{
		{
			CaseName: "Correct test WRC20",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc20Address,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				callApprove(t, operation.StdWRC20, spender, v.TokenAddress, v.Caller, value, c.Errs)

				call(t, vm.AccountRef(spender), v.TokenAddress, nil, opWrc20, c.Errs)

				balance := checkBalance(t, wrc20Address, owner)

				var z, res big.Int
				if res.Sub(balance, z.Sub(totalSupply, value)).Cmp(zeroBig) != 0 {
					t.Fatal()
				}
			},
		},
		{
			CaseName: "Correct test WRC721",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				mintNewToken(t, owner, wrc721Address, id, data, caller, c.Errs)

				balance := checkBalance(t, wrc721Address, owner)

				callApprove(t, operation.StdWRC721, spender, v.TokenAddress, v.Caller, id, c.Errs)

				call(t, vm.AccountRef(spender), v.TokenAddress, nil, opWrc721, c.Errs)

				balanceAfter := checkBalance(t, wrc721Address, owner)

				var res big.Int
				if res.Sub(balance, big.NewInt(1)).Cmp(balanceAfter) != 0 {
					t.Fatal()
				}
			},
		},
		{
			CaseName: "Wrong Caller",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{ErrWrongCaller},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				callApprove(t, operation.StdWRC721, spender, v.TokenAddress, v.Caller, id, c.Errs)

				call(t, vm.AccountRef(spender), v.TokenAddress, nil, opWrc721, c.Errs)
			},
		},
		{
			CaseName: "Not minted token",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{ErrNotMinted},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				op, err := operation.NewTransferFromOperation(operation.StdWRC721, owner, to, id7)
				if err != nil {
					t.Fatal(err)
				}

				call(t, v.Caller, v.TokenAddress, nil, op, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func TestProcessorMintOperationCall(t *testing.T) {
	mintOp, err := operation.NewMintOperation(owner, id2, data)
	if err != nil {
		t.Fatal(err)
	}

	cases := []testmodels.TestCase{
		{
			CaseName: "Correct test",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				call(t, v.Caller, v.TokenAddress, nil, mintOp, c.Errs)

				balance := checkBalance(t, wrc721Address, owner)

				if balance.Cmp(big.NewInt(1)) != 0 {
					t.Fatal()
				}
			},
		},
		{
			CaseName: "Unknown minter",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(to),
				TokenAddress: wrc721Address,
			},
			Errs: []error{ErrWrongMinter},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				call(t, v.Caller, v.TokenAddress, nil, mintOp, c.Errs)
			},
		},
		{
			CaseName: "MetadataExceedsMaxSize",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{ErrMetadataExceedsMaxSize},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				bigData := make([]byte, MetadataMaxSize+1)
				id := big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
				mintOpBigData, err := operation.NewMintOperation(owner, id, bigData)
				if err != nil {
					t.Fatal(err)
				}

				balanceBefore := checkBalance(t, wrc721Address, owner)

				v := c.TestData.(testmodels.TestData)
				call(t, v.Caller, v.TokenAddress, nil, mintOpBigData, c.Errs)

				balance := checkBalance(t, wrc721Address, owner)
				if balance.Cmp(balanceBefore) != 0 {
					t.Errorf("Expected %d got %s", balanceBefore, balance)
				}
			},
		},
		{
			CaseName: "MetadataMaxSize",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				data := make([]byte, MetadataMaxSize)
				id := big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
				mintOp, err := operation.NewMintOperation(owner, id, data)
				if err != nil {
					t.Fatal(err)
				}

				balanceBefore := checkBalance(t, wrc721Address, owner)

				v := c.TestData.(testmodels.TestData)
				call(t, v.Caller, v.TokenAddress, nil, mintOp, c.Errs)

				balance := checkBalance(t, wrc721Address, owner)
				expBalance := balanceBefore.Add(balanceBefore, big.NewInt(1))
				if balance.Cmp(expBalance) != 0 {
					t.Errorf("Expected %d got %s", expBalance, balance)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func TestProcessorTransferOperationCall(t *testing.T) {
	transferOp, err := operation.NewTransferOperation(to, value)
	if err != nil {
		t.Fatal(err)
	}

	cases := []testmodels.TestCase{
		{
			CaseName: "Correct transfer test",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc20Address,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				balance := checkBalance(t, wrc20Address, owner)

				call(t, v.Caller, v.TokenAddress, nil, transferOp, c.Errs)

				balanceAfter := checkBalance(t, wrc20Address, owner)

				z := new(big.Int)
				if balanceAfter.Cmp(z.Sub(balance, value)) != 0 {
					t.Fatal()
				}
			},
		}, {
			CaseName: "No empty address",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: common.Address{},
			},
			Errs: []error{operation.ErrNoAddress},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				call(t, v.Caller, v.TokenAddress, nil, transferOp, c.Errs)
			},
		},
		{
			CaseName: "Unknown Caller",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(address),
				TokenAddress: wrc20Address,
			},
			Errs: []error{ErrNotEnoughBalance},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				call(t, v.Caller, v.TokenAddress, nil, transferOp, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func TestProcessorBurnOperationCall(t *testing.T) {
	burnOp, err := operation.NewBurnOperation(id3)
	if err != nil {
		t.Fatal(err)
	}

	cases := []testmodels.TestCase{
		{
			CaseName: "Correct test",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				mintNewToken(t, owner, wrc721Address, id3, data, caller, c.Errs)

				balance := checkBalance(t, wrc721Address, owner)

				call(t, v.Caller, v.TokenAddress, nil, burnOp, c.Errs)

				balanceAfter := checkBalance(t, wrc721Address, owner)

				z := new(big.Int)
				if balanceAfter.Cmp(z.Sub(balance, big.NewInt(1))) != 0 {
					t.Fatal()
				}
			},
		},
		{
			CaseName: "Unknown minter",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(to),
				TokenAddress: wrc721Address,
			},
			Errs: []error{ErrWrongMinter},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				call(t, v.Caller, v.TokenAddress, nil, burnOp, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func TestProcessorApprovalForAllCall(t *testing.T) {
	op, err := operation.NewSetApprovalForAllOperation(spender, true)
	if err != nil {
		t.Fatal()
	}

	unapproveOp, err := operation.NewSetApprovalForAllOperation(spender, false)
	if err != nil {
		t.Fatal()
	}

	cases := []testmodels.TestCase{
		{
			CaseName: "Use approvalForAll",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				call(t, v.Caller, v.TokenAddress, nil, op, c.Errs)

				mintNewToken(t, owner, v.TokenAddress, id4, data, caller, c.Errs)

				callTransferFrom(t, operation.StdWRC721, owner, to, v.TokenAddress, id4, vm.AccountRef(spender), c.Errs)
			},
		},
		{
			CaseName: "Cancel approvalForAll",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil, ErrWrongCaller},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				call(t, v.Caller, v.TokenAddress, nil, op, c.Errs)

				mintNewToken(t, owner, v.TokenAddress, id5, data, caller, c.Errs)
				mintNewToken(t, owner, v.TokenAddress, id6, data, caller, c.Errs)

				callTransferFrom(t, operation.StdWRC721, owner, to, v.TokenAddress, id5, vm.AccountRef(spender), c.Errs)

				call(t, v.Caller, v.TokenAddress, nil, unapproveOp, c.Errs)

				callTransferFrom(t, operation.StdWRC721, owner, to, v.TokenAddress, id6, vm.AccountRef(spender), c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func TestProcessorIsApprovedForAll(t *testing.T) {
	cases := []testmodels.TestCase{
		{
			CaseName: "IsApprovalForAll",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				approvalOp, err := operation.NewSetApprovalForAllOperation(operator, true)
				if err != nil {
					t.Fatal(err)
				}

				call(t, v.Caller, v.TokenAddress, nil, approvalOp, c.Errs)

				ok := checkApprove(t, wrc721Address, owner, operator)

				if !ok {
					t.Fatal()
				}
			},
		},
		{
			CaseName: "IsNotApprovalForAll",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				ok := checkApprove(t, wrc721Address, owner, spender)

				if ok {
					t.Fatal()
				}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func TestProcessorPropertiesWRC20(t *testing.T) {
	wrc20Op, err := operation.NewPropertiesOperation(wrc20Address, nil)
	if err != nil {
		t.Fatal(err)
	}

	i, err := processor.Properties(wrc20Op)
	if err != nil {
		t.Fatal(err)
	}

	prop := i.(*WRC20PropertiesResult)
	testutils.CompareBytes(t, prop.Name, name)

	testutils.CompareBytes(t, prop.Symbol, symbol)

	if !testutils.BigIntEquals(prop.TotalSupply, totalSupply) {
		t.Errorf("values do not match:\nwant: %+v\nhave: %+v", prop.TotalSupply, totalSupply)
	}

	if prop.Decimals != decimals {
		t.Errorf("values do not match:\nwant: %+v\nhave: %+v", decimals, prop.Decimals)
	}

	if !testutils.BigIntEquals(prop.Cost, zeroBig) {
		t.Errorf("values do not match:\nwant: %+v\nhave: %+v", prop.Cost, zeroBig)
	}
}

func TestProcessorPropertiesWRC721(t *testing.T) {
	tokenAddress := createToken(t, operation.StdWRC721, vm.AccountRef(owner))

	mintNewToken(t, owner, tokenAddress, id7, data, caller, []error{nil})
	approveOp, err := operation.NewApproveOperation(operation.StdWRC721, spender, id7)
	call(t, vm.AccountRef(owner), tokenAddress, nil, approveOp, []error{nil})

	wrc721Op, err := operation.NewPropertiesOperation(tokenAddress, id7)
	if err != nil {
		t.Fatal(err)
	}

	i, err := processor.Properties(wrc721Op)
	if err != nil {
		t.Fatal(err)
	}

	prop := i.(*WRC721PropertiesResult)
	testutils.CompareBytes(t, prop.Name, name)

	testutils.CompareBytes(t, prop.Symbol, symbol)

	testutils.CompareBytes(t, prop.BaseURI, baseURI)

	testutils.CompareBytes(t, prop.Metadata, data)

	testutils.CompareBytes(t, prop.TokenURI, concatTokenURI(baseURI, id7))

	if prop.OwnerOf != owner {
		t.Fatal()
	}

	if prop.GetApproved != spender {
		t.Fatal()
	}

	if prop.PercentFee != percentFee {
		t.Errorf("values do not match:\nwant: %+v\nhave: %+v", percentFee, prop.PercentFee)
	}
}

func TestProcessorApproveCall(t *testing.T) {
	approveOp, err := operation.NewApproveOperation(operation.StdWRC20, approveAddress, value)
	if err != nil {
		t.Fatal()
	}

	cases := []testmodels.TestCase{
		{
			CaseName: "Use approve",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc20Address,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				call(t, v.Caller, v.TokenAddress, nil, approveOp, c.Errs)

				allowanceOp, err := operation.NewAllowanceOperation(wrc20Address, owner, approveAddress)
				if err != nil {
					t.Fatal(err)
				}

				total, err := processor.Allowance(allowanceOp)
				if err != nil {
					t.Fatal(err)
				}

				if total.Cmp(value) != 0 {
					t.Fatal()
				}
			},
		},
		{
			CaseName: "Non approved address",
			TestData: testmodels.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc20Address,
			},
			Errs: []error{nil, ErrWrongCaller},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				allowanceOp, err := operation.NewAllowanceOperation(wrc20Address, owner, to)
				if err != nil {
					t.Fatal(err)
				}

				total, err := processor.Allowance(allowanceOp)
				if err != nil {
					t.Fatal(err)
				}

				if total.Cmp(zeroBig) != 0 {
					t.Fatal()
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func TestProcessorSetPriceCall(t *testing.T) {
	cases := []testmodels.TestCase{
		{
			CaseName: "WRC721_Correct",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(owner),
			},
			Errs: []error{},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				v.TokenAddress = createToken(t, operation.StdWRC721, v.Caller)

				tokenId := big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
				mintNewToken(t, owner, v.TokenAddress, tokenId, data, caller, c.Errs)

				_, err := checkCost(v.TokenAddress, tokenId)
				if err != ErrTokenIsNotForSale {
					t.Errorf("Expected: %s\nGot: %s", ErrTokenIsNotForSale, err)
					t.FailNow()
				}

				setPriceOp, err := operation.NewSetPriceOperation(tokenId, value)
				if err != nil {
					t.Error(err)
					t.FailNow()
				}
				call(t, v.Caller, v.TokenAddress, nil, setPriceOp, c.Errs)

				newCost, err := checkCost(v.TokenAddress, tokenId)
				if err != nil {
					t.Error(err)
					t.FailNow()
				}

				if !(newCost.Cmp(value) == 0) {
					t.Errorf("cost was not changed")
				}
			},
		},
		{
			CaseName: "WRC721_NoTokenId",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(owner),
			},
			Errs: []error{ErrTokenIdIsNotSet},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				v.TokenAddress = createToken(t, operation.StdWRC721, v.Caller)

				setPriceOp, err := operation.NewSetPriceOperation(nil, value)
				if err != nil {
					t.Error(err)
					t.FailNow()
				}
				call(t, v.Caller, v.TokenAddress, nil, setPriceOp, c.Errs)
			},
		},
		{
			CaseName: "WRC20_Correct",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(owner),
			},
			Errs: []error{},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				v.TokenAddress = createToken(t, operation.StdWRC20, v.Caller)

				_, err := checkCost(v.TokenAddress, nil)
				if err != ErrTokenIsNotForSale {
					t.Errorf("Expected: %s\nGot: %s", ErrTokenIsNotForSale, err)
					t.FailNow()
				}

				setPriceOp, err := operation.NewSetPriceOperation(nil, value)
				if err != nil {
					t.Error(err)
					t.FailNow()
				}
				call(t, v.Caller, v.TokenAddress, nil, setPriceOp, c.Errs)

				newCost, err := checkCost(v.TokenAddress, nil)
				if err != nil {
					t.Error(err)
					t.FailNow()
				}

				if !(newCost.Cmp(value) == 0) {
					t.Errorf("cost was not changed")
				}
			},
		},
		{
			CaseName: "WRC20_NoTokenId",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(owner),
			},
			Errs: []error{},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				v.TokenAddress = createToken(t, operation.StdWRC20, v.Caller)

				setPriceOp, err := operation.NewSetPriceOperation(nil, value)
				if err != nil {
					t.Error(err)
					t.FailNow()
				}
				call(t, v.Caller, v.TokenAddress, nil, setPriceOp, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func TestProcessorBuyCall(t *testing.T) {
	cases := []testmodels.TestCase{
		{
			CaseName: "WRC721_CorrectAndFee",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(owner),
			},
			Errs: []error{},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				v.TokenAddress = createToken(t, operation.StdWRC721, v.Caller)

				tokenId := big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
				mintNewToken(t, seller, v.TokenAddress, tokenId, data, v.Caller, nil)

				sellCaller := vm.AccountRef(seller)
				price := big.NewInt(0).Set(value)
				setPrice(t, sellCaller, v.TokenAddress, tokenId, price)

				reminder := big.NewInt(111)
				value.Add(value, reminder)

				buyCaller := vm.AccountRef(buyer)
				processor.state.AddBalance(buyCaller.Address(), value)

				newVal := big.NewInt(int64(testutils.RandomInt(10, 30)))
				buyOp, err := operation.NewBuyOperation(tokenId, newVal)
				if err != nil {
					t.Error(err)
					t.FailNow()
				}
				call(t, buyCaller, v.TokenAddress, value, buyOp, c.Errs)

				cost, err := checkCost(v.TokenAddress, tokenId)
				if err != nil {
					t.Error(err)
					t.FailNow()
				}

				if !(cost.Cmp(newVal) == 0) {
					t.Errorf("Expected cost: %s\ngot: %s", newVal, cost)
				}

				buyerWeiBalance := processor.state.GetBalance(buyCaller.Address())
				if buyerWeiBalance.Cmp(reminder) != 0 {
					t.Errorf("Expected buyerWeiBalance balance: %d\nactual: %s", reminder, buyerWeiBalance)
				}

				fee := big.NewInt(0).Set(price)
				fee.Mul(fee, big.NewInt(int64(percentFee)))
				fee.Div(fee, big.NewInt(100))
				minterWeiBalance := processor.state.GetBalance(v.Caller.Address())
				if !(minterWeiBalance.Cmp(fee) == 0) {
					t.Errorf("Expected minterWeiBalance balance: %s\nactual: %s", fee, minterWeiBalance)
				}

				expSellerWeiBalance := big.NewInt(0).Set(price)
				expSellerWeiBalance.Sub(expSellerWeiBalance, fee)
				sellerWeiBalance := processor.state.GetBalance(sellCaller.Address())
				if !(sellerWeiBalance.Cmp(expSellerWeiBalance) == 0) {
					t.Errorf("Expected sellerWeiBalance balance: %s\nactual: %s", expSellerWeiBalance, sellerWeiBalance)
				}

				sellerBalance := checkBalance(t, v.TokenAddress, seller)
				if sellerBalance.Cmp(zeroBig) != 0 {
					t.Errorf("Expected sellerBalance balance: %s\nactual: %s", zeroBig, sellerBalance)
				}

				buyerBalance := checkBalance(t, v.TokenAddress, buyer)
				if buyerBalance.Cmp(big.NewInt(1)) != 0 {
					t.Errorf("Expected buyerBalance balance: %d\nactual: %s", 1, buyerBalance)
				}
			},
		},
		{
			CaseName: "WRC721_NoTokenId",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(owner),
			},
			Errs: []error{ErrTokenIdIsNotSet},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				v.TokenAddress = createToken(t, operation.StdWRC721, v.Caller)

				newVal := big.NewInt(int64(testutils.RandomInt(10, 30)))
				buyOp, err := operation.NewBuyOperation(nil, newVal)
				if err != nil {
					t.Error(err)
					t.FailNow()
				}

				call(t, caller, v.TokenAddress, value, buyOp, c.Errs)
			},
		},
		{
			CaseName: "WRC721_NoNewValue",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(owner),
			},
			Errs: []error{ErrNewValueIsNotSet},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				v.TokenAddress = createToken(t, operation.StdWRC721, v.Caller)

				tokenId := big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
				mintNewToken(t, seller, v.TokenAddress, tokenId, data, v.Caller, nil)

				buyOp, err := operation.NewBuyOperation(tokenId, nil)
				if err != nil {
					t.Error(err)
					t.FailNow()
				}
				call(t, caller, v.TokenAddress, value, buyOp, c.Errs)
			},
		},
		{
			CaseName: "WRC721_SmallValue",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(owner),
			},
			Errs: []error{ErrTooSmallTxValue},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				v.TokenAddress = createToken(t, operation.StdWRC721, v.Caller)

				tokenId := big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
				mintNewToken(t, seller, v.TokenAddress, tokenId, data, v.Caller, nil)

				sellCaller := vm.AccountRef(seller)
				setPrice(t, sellCaller, v.TokenAddress, tokenId, value)

				newVal := big.NewInt(int64(testutils.RandomInt(10, 30)))
				buyOp, err := operation.NewBuyOperation(tokenId, newVal)
				if err != nil {
					t.Error(err)
					t.FailNow()
				}

				badVal := big.NewInt(-1)
				badVal.Add(badVal, value)
				call(t, caller, v.TokenAddress, badVal, buyOp, c.Errs)
			},
		},
		{
			CaseName: "WRC20_Correct",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(owner),
			},
			Errs: []error{},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				v.TokenAddress = createToken(t, operation.StdWRC20, v.Caller)

				pricePerToken := big.NewInt(1000)
				setPrice(t, v.Caller, v.TokenAddress, nil, pricePerToken)

				reminder := big.NewInt(111)
				expTokenCount := big.NewInt(10)
				expTotalPrice := big.NewInt(0).Mul(pricePerToken, expTokenCount)
				value.Set(expTotalPrice).Add(value, reminder)

				buyer = common.BytesToAddress(testutils.RandomData(20))
				buyCaller := vm.AccountRef(buyer)
				processor.state.AddBalance(buyCaller.Address(), value)

				creatorBalanceBefore := checkBalance(t, v.TokenAddress, owner)
				creatorWeiBalanceBefore := processor.state.GetBalance(owner)
				buyOp, err := operation.NewBuyOperation(nil, nil)
				if err != nil {
					t.Error(err)
					t.FailNow()
				}
				call(t, buyCaller, v.TokenAddress, value, buyOp, c.Errs)

				buyerWeiBalance := processor.state.GetBalance(buyCaller.Address())
				if buyerWeiBalance.Cmp(reminder) != 0 {
					t.Errorf("Expected buyerWeiBalance balance: %d\nactual: %s", reminder, buyerWeiBalance)
				}

				expCreatorWeiBalance := big.NewInt(0).Add(creatorWeiBalanceBefore, expTotalPrice)
				creatorWeiBalance := processor.state.GetBalance(v.Caller.Address())
				if !(creatorWeiBalance.Cmp(expCreatorWeiBalance) == 0) {
					t.Errorf("Expected creatorWeiBalance balance: %s\nactual: %s", expCreatorWeiBalance, creatorWeiBalance)
				}

				expCreatorBalance := creatorBalanceBefore.Sub(creatorBalanceBefore, expTokenCount)
				gotCreatorBalance := checkBalance(t, v.TokenAddress, owner)
				if gotCreatorBalance.Cmp(expCreatorBalance) != 0 {
					t.Errorf("Expected creatorBalanceBefore balance: %s\nactual: %s", zeroBig, creatorBalanceBefore)
				}

				buyerBalance := checkBalance(t, v.TokenAddress, buyer)
				if buyerBalance.Cmp(expTokenCount) != 0 {
					t.Errorf("Expected buyerBalance balance: %d\nactual: %s", 1, buyerBalance)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func checkBalance(t *testing.T, TokenAddress, owner common.Address) *big.Int {
	balanceOp, err := operation.NewBalanceOfOperation(TokenAddress, owner)
	if err != nil {
		t.Fatal(err)
	}

	balance, err := processor.BalanceOf(balanceOp)
	if err != nil {
		t.Fatal(err)
	}

	return balance
}

func mintNewToken(t *testing.T, owner, TokenAddress common.Address, id *big.Int, data []byte, Caller Ref, Errs []error) {
	mintOp, err := operation.NewMintOperation(owner, id, data)
	if err != nil {
		t.Fatal(err)
	}

	call(t, Caller, TokenAddress, nil, mintOp, Errs)
}

func call(t *testing.T, Caller Ref, TokenAddress common.Address, value *big.Int, op operation.Operation, Errs []error) []byte {
	res, err := processor.Call(Caller, TokenAddress, value, op)
	if !testutils.CheckError(err, Errs) {
		t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", Errs, err)
	}

	return res
}

func checkApprove(t *testing.T, TokenAddress, owner, operator common.Address) bool {
	op, err := operation.NewIsApprovedForAllOperation(TokenAddress, owner, operator)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := processor.IsApprovedForAll(op)
	if err != nil {
		t.Fatal(err)
	}

	return ok
}

func callApprove(t *testing.T, std operation.Std, spender, TokenAddress common.Address, Caller Ref, value *big.Int, Errs []error) {
	approveOp, err := operation.NewApproveOperation(std, spender, value)
	if err != nil {
		t.Fatal(err)
	}

	call(t, Caller, TokenAddress, nil, approveOp, Errs)
}

func callTransferFrom(
	t *testing.T,
	std operation.Std,
	owner, to, TokenAddress common.Address,
	id *big.Int,
	Caller Ref,
	Errs []error,
) {
	transferOp, err := operation.NewTransferFromOperation(std, owner, to, id)
	if err != nil {
		t.Fatal(err)
	}

	call(t, Caller, TokenAddress, nil, transferOp, Errs)
}

func checkCost(tokenAddress common.Address, tokenId *big.Int) (*big.Int, error) {
	costOp, err := operation.NewCostOperation(tokenAddress, tokenId)
	if err != nil {
		return nil, err
	}

	return processor.Cost(costOp)
}

func setPrice(t *testing.T, caller Ref, tokenAddress common.Address, tokenId, value *big.Int) {
	setPriceOp, err := operation.NewSetPriceOperation(tokenId, value)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	call(t, caller, tokenAddress, nil, setPriceOp, nil)
}

func createToken(t *testing.T, std operation.Std, caller Ref) common.Address {
	t.Helper()

	var err error
	var createOp operation.Operation
	switch std {
	case operation.StdWRC20:
		createOp, err = operation.NewWrc20CreateOperation(name, symbol, &decimals, totalSupply)
	case operation.StdWRC721:
		createOp, err = operation.NewWrc721CreateOperation(name, symbol, baseURI, &percentFee)
	default:
		err = fmt.Errorf("cannot create token")
	}

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	adr := call(t, caller, common.Address{}, nil, createOp, nil)
	return common.BytesToAddress(adr)
}
