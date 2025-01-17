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

package shuffle

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestShuffleCreators(t *testing.T) {
	indexes := make([]common.Address, testutils.RandomInt(10, 9999))
	for i := 0; i < len(indexes); i++ {
		indexes[i] = common.HexToAddress(strconv.Itoa(i))
	}

	input := make([]common.Address, len(indexes))
	copy(input, indexes)

	seed := sha256.Sum256(Bytes8(uint64(testutils.RandomInt(0, 9999))))

	shuffledList, err := ShuffleValidators(input, seed)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	if reflect.DeepEqual(indexes, shuffledList) {
		t.Fatalf("got not shuffled list")
	}
}

//func TestShuffleList(t *testing.T) {
//	testInnerShuffleList(t, shuffleList, uint64(testutils.RandomInt(0, 9999)))
//}

func TestUnshuffleList(t *testing.T) {
	testInnerShuffleList(t, unshuffleList, uint64(testutils.RandomInt(0, 9999)))
}

func testInnerShuffleList(t *testing.T, f func([]common.Address, common.Hash) ([]common.Address, error), epoch uint64) {
	validatorsCount := uint64(testutils.RandomInt(0, 9999))

	validators := make([]common.Address, validatorsCount)
	for i := 0; i < 100; i++ {
		validators[i] = common.HexToAddress(strconv.Itoa(i))
	}

	input := make([]common.Address, len(validators))
	copy(input, validators)

	seed := sha256.Sum256(Bytes8(epoch))

	shuffledList, err := f(input, seed)
	if err != nil {
		t.Fatalf("error while shuffling list: %v", err)
	}

	if reflect.DeepEqual(validators, shuffledList) {
		t.Fatalf("unexpected output: %v", shuffledList)
	}
}

func TestSwapOrNot(t *testing.T) {
	addr1 := common.HexToAddress(strconv.Itoa(1))
	addr2 := common.HexToAddress(strconv.Itoa(2))
	addr3 := common.HexToAddress(strconv.Itoa(3))
	input := []common.Address{addr1, addr2, addr3}
	buf := make([]byte, totalSize)

	tests := []struct {
		name           string
		expectedOutput []common.Address
		buf            []byte
		source         common.Hash
		byteV          byte
	}{
		{
			name:           "don`t swap elements",
			expectedOutput: []common.Address{addr1, addr2, addr3},
			buf:            make([]byte, totalSize),
			source:         common.Hash{},
			byteV:          byte(0),
		}, {
			name:           "swap elements",
			expectedOutput: []common.Address{addr1, addr3, addr2},
			buf:            make([]byte, totalSize),
			source:         common.BytesToHash([]byte{1, 2, 3}),
			byteV:          byte(4),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			swapOrNot(buf, test.byteV, 1, 2, input, test.source, CustomSHA256Hasher())
			if !reflect.DeepEqual(input, test.expectedOutput) {
				t.Errorf("expected output: %v, got: %v", test.expectedOutput, input)
			}
		})
	}
}

func BenchmarkShuffleValidators(b *testing.B) {
	var validators = make([]common.Address, 10000000)
	for i := range validators {
		validators[i] = common.HexToAddress(strconv.Itoa(i))
	}

	seed := common.HexToHash("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdef")

	sizes := []int{1000, 10000, 100000, 500000, 1000000, 10000000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			subset := validators[:size]
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := ShuffleValidators(subset, seed)
				if err != nil {
					b.Fatalf("Error shuffling validators: %v", err)
				}
			}
		})
	}
}
