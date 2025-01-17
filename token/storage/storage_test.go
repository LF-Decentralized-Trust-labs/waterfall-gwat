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

package storage

import (
	"reflect"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestCreateStorage(t *testing.T) {
	descriptors := []FieldDescriptor{
		newPanicFieldDescriptor(t, []byte("scalar"), newScalarPanic(t, Uint8Type)),
		newPanicFieldDescriptor(t, []byte("array"), NewArrayProperties(newScalarPanic(t, Uint16Type), 10)),
		newPanicFieldDescriptor(t, []byte("map"), newMapPropertiesPanic(t, newScalarPanic(t, Uint32Type), newScalarPanic(t, Uint32Type))),
	}

	t.Run("NewStorage", func(t *testing.T) {
		db, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		addr := common.BytesToAddress(testutils.RandomData(20))
		stream := NewStorageStream(addr, db)

		expectedSign, err := NewSignatureV1(descriptors)
		require.NoError(t, err, err)

		storage, err := NewStorage(stream, descriptors)
		assert.NoError(t, err, err)
		require.NotNil(t, storage)

		readSign := new(SignatureV1)
		_, err = readSign.ReadFromStream(stream)
		assert.NoError(t, err, err)
		assert.EqualValues(t, expectedSign, readSign)
	})

	t.Run("ReadStorage", func(t *testing.T) {
		db, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		addr := common.BytesToAddress(testutils.RandomData(20))
		stream := NewStorageStream(addr, db)

		sig, err := NewSignatureV1(descriptors)
		require.NoError(t, err, err)

		_, err = sig.WriteToStream(stream)
		require.NoError(t, err, err)

		storage, err := ReadStorage(stream)
		assert.NoError(t, err, err)
		assert.NotNil(t, storage)
	})
}

func TestStorage_WriteReadField(t *testing.T) {
	tests := []struct {
		descriptor FieldDescriptor
		expValue   interface{}
		readTo     func() interface{}
		isEqual    func(interface{}, interface{}) bool
	}{
		{
			descriptor: newPanicFieldDescriptor(t,
				[]byte("Scalar_Uint8"),
				newScalarPanic(t, Uint8Type),
			),
			expValue: uint8(10),
			readTo: func() interface{} {
				v := uint8(0)
				return &v
			},
			isEqual: compareScalar,
		},
		{
			descriptor: newPanicFieldDescriptor(t,
				[]byte("Uint256"),
				newScalarPanic(t, Uint256Type),
			),
			expValue: new(uint256.Int).SetBytes([]byte{
				0x01, 0x02, 0x03, 0x04,
				0x05, 0x06, 0x07, 0x08,
				0x09, 0x0a, 0x0b, 0x0c,
				0x0d, 0x0e, 0x0f, 0x00,
				0x11, 0x12, 0x13, 0x14,
				0x15, 0x16, 0x17, 0x18,
				0x19, 0x1a, 0x1b, 0x1c,
				0x1d, 0x1e, 0x1f, 0x10,
			}),
			readTo: func() interface{} {
				v := uint256.Int{}
				return &v
			},
			isEqual: compareScalar,
		},
		{
			descriptor: newPanicFieldDescriptor(
				t,
				[]byte("Array_Uint16"),
				NewArrayProperties(newScalarPanic(t, Uint16Type), 10),
			),
			expValue: []uint16{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			readTo: func() interface{} {
				var v []uint16
				return &v
			},
			isEqual: compareArray,
		},
		{
			descriptor: newPanicFieldDescriptor(
				t,
				[]byte("Map_Uint16_Uint16"),
				newMapPropertiesPanic(t, newScalarPanic(t, Uint16Type), newScalarPanic(t, Uint16Type)),
			),
			expValue: NewKeyValuePair(uint16(111), uint16(222)),
			readTo: func() interface{} {
				v := uint16(0)
				return NewKeyValuePair(uint16(111), &v)
			},
			isEqual: compareMap,
		},
		{
			descriptor: newPanicFieldDescriptor(
				t,
				[]byte("Map_ArrayUint16_Uint16"),
				newMapPropertiesPanic(t, NewArrayProperties(newScalarPanic(t, Uint16Type), 3), newScalarPanic(t, Uint16Type)),
			),
			expValue: NewKeyValuePair([]uint16{1, 2, 3}, uint16(222)),
			readTo: func() interface{} {
				v := uint16(0)
				return NewKeyValuePair([]uint16{1, 2, 3}, &v)
			},
			isEqual: compareMap,
		},
		{
			descriptor: newPanicFieldDescriptor(
				t,
				[]byte("Map_Uint16_ArrayUint16"),
				newMapPropertiesPanic(t, newScalarPanic(t, Uint16Type), NewArrayProperties(newScalarPanic(t, Uint16Type), 3)),
			),
			expValue: NewKeyValuePair(uint16(222), []uint16{1, 2, 3}),
			readTo: func() interface{} {
				var v []uint16
				return NewKeyValuePair(uint16(222), &v)
			},
			isEqual: compareMap,
		},
		{
			descriptor: newPanicFieldDescriptor(
				t,
				[]byte("Map_ArrayUint256_ArrayUint256"),
				newMapPropertiesPanic(
					t,
					NewArrayProperties(newScalarPanic(t, Uint256Type), 3),
					NewArrayProperties(newScalarPanic(t, Uint256Type), 3),
				),
			),
			expValue: NewKeyValuePair(
				[]*uint256.Int{
					new(uint256.Int).SetBytes([]byte{0x01, 0x02, 0x03, 0x04}),
					new(uint256.Int).SetBytes([]byte{0x05, 0x06, 0x07, 0x08}),
					new(uint256.Int).SetBytes([]byte{0x09, 0x0a, 0x0b, 0x0c}),
				},
				[]*uint256.Int{
					new(uint256.Int).SetBytes([]byte{0x0d, 0x0e, 0x0f, 0x00}),
					new(uint256.Int).SetBytes([]byte{0x11, 0x12, 0x13, 0x14}),
					new(uint256.Int).SetBytes([]byte{0x15, 0x16, 0x17, 0x18}),
				},
			),
			readTo: func() interface{} {
				var v []*uint256.Int
				return NewKeyValuePair([]*uint256.Int{
					new(uint256.Int).SetBytes([]byte{0x01, 0x02, 0x03, 0x04}),
					new(uint256.Int).SetBytes([]byte{0x05, 0x06, 0x07, 0x08}),
					new(uint256.Int).SetBytes([]byte{0x09, 0x0a, 0x0b, 0x0c}),
				}, &v)
			},
			isEqual: compareMap,
		},
		{
			descriptor: newPanicFieldDescriptor(
				t,
				[]byte("Map_Uint16_SliceUint16"),
				newMapPropertiesPanic(t, newScalarPanic(t, Uint16Type), NewSliceProperties(newScalarPanic(t, Uint16Type))),
			),
			expValue: NewKeyValuePair(uint16(222), []uint16{1, 2, 3, 4}),
			readTo: func() interface{} {
				var v []uint16
				return NewKeyValuePair(uint16(222), &v)
			},
			isEqual: compareMap,
		},
		{
			descriptor: newPanicFieldDescriptor(
				t,
				[]byte("Map_ArrayUint256_SliceUint8"),
				newMapPropertiesPanic(
					t,
					NewArrayProperties(newScalarPanic(t, Uint256Type), 3),
					NewSliceProperties(newScalarPanic(t, Uint8Type)),
				),
			),
			expValue: NewKeyValuePair(
				[]*uint256.Int{
					new(uint256.Int).SetBytes([]byte{0x01, 0x02, 0x03, 0x04}),
					new(uint256.Int).SetBytes([]byte{0x05, 0x06, 0x07, 0x08}),
					new(uint256.Int).SetBytes([]byte{0x09, 0x0a, 0x0b, 0x0c}),
				},
				[]uint8{1, 2, 3, 4},
			),
			readTo: func() interface{} {
				var v []uint8
				return NewKeyValuePair([]*uint256.Int{
					new(uint256.Int).SetBytes([]byte{0x01, 0x02, 0x03, 0x04}),
					new(uint256.Int).SetBytes([]byte{0x05, 0x06, 0x07, 0x08}),
					new(uint256.Int).SetBytes([]byte{0x09, 0x0a, 0x0b, 0x0c}),
				}, &v)
			},
			isEqual: compareMap,
		},
	}

	descriptors := make([]FieldDescriptor, len(tests))
	for i, test := range tests {
		descriptors[i] = test.descriptor
	}

	storage := newStorage(t, descriptors)

	for _, test := range tests {
		name := string(test.descriptor.Name())
		t.Run(name, func(t *testing.T) {
			err := storage.WriteField(name, test.expValue)
			assert.NoError(t, err, err)

			to := test.readTo()
			err = storage.ReadField(name, to)
			assert.NoError(t, err, err)
			assert.True(t, test.isEqual(test.expValue, to))
		})
	}
}

func newStorage(t *testing.T, descriptors []FieldDescriptor) Storage {
	t.Helper()

	db, err := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	require.NoError(t, err, err)

	addr := common.BytesToAddress(testutils.RandomData(20))
	stream := NewStorageStream(addr, db)

	storage, err := NewStorage(stream, descriptors)
	require.NoError(t, err, err)
	require.NotNil(t, storage)

	return storage
}

func compareScalar(exp, got interface{}) bool {
	expV := reflect.ValueOf(exp)
	if expV.Kind() == reflect.Ptr {
		expV = expV.Elem()
	}

	gotV := reflect.ValueOf(got)
	if gotV.Kind() == reflect.Ptr {
		gotV = gotV.Elem()
	}

	return reflect.DeepEqual(expV.Interface(), gotV.Interface())
}

func compareArray(expArr, arrPtr interface{}) bool {
	exp := reflect.ValueOf(expArr)
	if exp.Kind() == reflect.Ptr {
		exp = exp.Elem()
	}

	got := reflect.ValueOf(arrPtr)
	if got.Kind() == reflect.Ptr {
		got = got.Elem()
	}

	if exp.Len() != got.Len() {
		return false
	}

	for i := 0; i < exp.Len(); i++ {
		if !compareScalar(exp.Index(i).Interface(), got.Index(i).Interface()) {
			return false
		}
	}

	return true
}

func compareMap(expVal, gotPtr interface{}) bool {
	exp, ok := expVal.(*KeyValuePair)
	if !ok {
		return ok
	}

	got, ok := gotPtr.(*KeyValuePair)
	if !ok {
		return ok
	}

	keyEq := false
	kk := reflect.ValueOf(exp.key).Kind()
	if kk == reflect.Slice || kk == reflect.Array {
		keyEq = compareArray(exp.Key(), got.Key())
	} else {
		keyEq = compareScalar(exp.Key(), got.Key())
	}

	valEq := false
	vk := reflect.ValueOf(exp.value).Kind()
	if vk == reflect.Slice || vk == reflect.Array {
		valEq = compareArray(exp.Value(), got.Value())
	} else {
		valEq = compareScalar(exp.Value(), got.Value())
	}

	return keyEq && valEq
}
