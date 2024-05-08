package distribution_test

import (
	"encoding/hex"
	"encoding/json"
	"github.com/Layr-Labs/eigenlayer-payment-proofs/internal/tests"
	"math/big"
	"strings"
	"testing"

	"github.com/Layr-Labs/eigenlayer-payment-proofs/pkg/distribution"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func GetTestDistribution() *distribution.Distribution {
	d := distribution.NewDistribution()

	// give some addresses many tokens
	// addr1 => token_1 => 1
	// addr1 => token_2 => 2
	// ...
	// addr1 => token_n => n
	// addr2 => token_1 => 2
	// addr2 => token_2 => 3
	// ...
	// addr2 => token_n-1 => n+1
	for i := 0; i < len(tests.TestAddresses); i++ {
		for j := 0; j < len(tests.TestTokens)-i; j++ {
			d.Set(tests.TestAddresses[i], tests.TestTokens[j], big.NewInt(int64(j+i+1)))
		}
	}

	return d
}

func GetCompleteTestDistribution() *distribution.Distribution {
	d := distribution.NewDistribution()

	for i := 0; i < len(tests.TestAddresses); i++ {
		for j := 0; j < len(tests.TestTokens); j++ {
			d.Set(tests.TestAddresses[i], tests.TestTokens[j], big.NewInt(int64(j+i+2)))
		}
	}

	return d
}

func FuzzSetAndGet(f *testing.F) {
	f.Add([]byte{69}, []byte{42, 0}, uint64(69420))

	f.Fuzz(func(t *testing.T, addressBytes, tokenBytes []byte, amounUintFuzz uint64) {
		address := common.Address{}
		address.SetBytes(addressBytes)

		token := common.Address{}
		token.SetBytes(tokenBytes)

		amount := new(big.Int).SetUint64(amounUintFuzz)

		d := distribution.NewDistribution()
		err := d.Set(address, token, amount)
		assert.NoError(t, err)

		fetched, found := d.Get(address, token)
		assert.True(t, found)
		assert.Equal(t, amount, fetched)
	})
}

func TestSetNilAmount(t *testing.T) {
	d := distribution.NewDistribution()
	err := d.Set(common.Address{}, common.Address{}, nil)
	assert.NoError(t, err)

	_, found := d.Get(common.Address{}, common.Address{})
	assert.True(t, found)
}

func TestSetAddressesInNonAlphabeticalOrder(t *testing.T) {
	d := distribution.NewDistribution()

	err := d.Set(tests.TestAddresses[1], tests.TestTokens[0], big.NewInt(1))
	assert.NoError(t, err)

	err = d.Set(tests.TestAddresses[0], tests.TestTokens[0], big.NewInt(2))
	assert.ErrorIs(t, err, distribution.ErrAddressNotInOrder)

	amount1, found := d.Get(tests.TestAddresses[1], tests.TestTokens[0])
	assert.Equal(t, big.NewInt(1), amount1)
	assert.True(t, found)

	amount2, found := d.Get(tests.TestAddresses[0], tests.TestTokens[0])
	assert.Equal(t, big.NewInt(0), amount2)
	assert.False(t, found)
}

func TestSetTokensInNonAlphabeticalOrder(t *testing.T) {
	d := distribution.NewDistribution()

	err := d.Set(tests.TestAddresses[0], tests.TestTokens[1], big.NewInt(1))
	assert.NoError(t, err)

	err = d.Set(tests.TestAddresses[0], tests.TestTokens[0], big.NewInt(2))
	assert.ErrorIs(t, err, distribution.ErrTokenNotInOrder)

	amount1, found := d.Get(tests.TestAddresses[0], tests.TestTokens[1])
	assert.Equal(t, big.NewInt(1), amount1)
	assert.True(t, found)

	amount2, found := d.Get(tests.TestAddresses[0], tests.TestTokens[0])
	assert.Equal(t, big.NewInt(0), amount2)
	assert.False(t, found)
}

func TestGetUnset(t *testing.T) {
	d := distribution.NewDistribution()

	fetched, found := d.Get(tests.TestAddresses[0], tests.TestTokens[0])
	assert.Equal(t, big.NewInt(0), fetched)
	assert.False(t, found)
}

func TestEncodeAccountLeaf(t *testing.T) {
	for i := 0; i < len(tests.TestAddresses); i++ {
		testRoot, _ := hex.DecodeString(tests.TestRootsString[i])
		leaf := distribution.EncodeAccountLeaf(tests.TestAddresses[i], testRoot)
		assert.Equal(t, distribution.EARNER_LEAF_SALT[0], leaf[0], "The first byte of the leaf should be EARNER_LEAF_SALT")
		assert.Equal(t, tests.TestAddresses[i][:], leaf[1:21])
		assert.Equal(t, testRoot, leaf[21:])
	}
}

func TestEncodeTokenLeaf(t *testing.T) {
	for i := 0; i < len(tests.TestTokens); i++ {
		testAmount, _ := new(big.Int).SetString(tests.TestAmountsString[i], 10)
		leaf := distribution.EncodeTokenLeaf(tests.TestTokens[i], testAmount)
		assert.Equal(t, distribution.TOKEN_LEAF_SALT[0], leaf[0], "The first byte of the leaf should be TOKEN_LEAF_SALT")
		assert.Equal(t, tests.TestTokens[i][:], leaf[1:21])
		assert.Equal(t, tests.TestAmountsBytes32[i], hex.EncodeToString(leaf[21:]))
	}
}

func TestGetAccountIndexBeforeMerklization(t *testing.T) {
	d := GetTestDistribution()

	accountIndex, found := d.GetAccountIndex(tests.TestAddresses[1])
	assert.False(t, found)
	assert.Equal(t, uint64(0), accountIndex)
}

func TestGetTokenIndexBeforeMerklization(t *testing.T) {
	d := GetTestDistribution()

	tokenIndex, found := d.GetTokenIndex(tests.TestAddresses[1], tests.TestTokens[1])
	assert.False(t, found)
	assert.Equal(t, uint64(0), tokenIndex)
}

func TestMerklize(t *testing.T) {
	d := GetTestDistribution()

	accountTree, tokenTrees, err := d.Merklize()
	assert.NoError(t, err)

	// check the token trees
	assert.Len(t, tokenTrees, len(tests.TestAddresses))
	for i := 0; i < len(tokenTrees); i++ {
		tokenTree, found := tokenTrees[tests.TestAddresses[i]]
		assert.True(t, found)
		assert.Len(t, tokenTree.Data, len(tests.TestTokens)-i)

		// check the data, that means the leafs are the same
		for j := 0; j < len(tests.TestTokens)-i; j++ {
			leaf := tokenTree.Data[j]
			assert.Equal(t, distribution.EncodeTokenLeaf(tests.TestTokens[j], big.NewInt(int64(j+i+1))), leaf)
		}
	}

	// check the account tree
	assert.Len(t, accountTree.Data, len(tests.TestAddresses))
	for i := 0; i < len(tests.TestAddresses); i++ {
		accountRoot := tokenTrees[tests.TestAddresses[i]].Root()
		leaf := accountTree.Data[i]
		assert.Equal(t, distribution.EncodeAccountLeaf(tests.TestAddresses[i], accountRoot), leaf)

		accountIndex, found := d.GetAccountIndex(tests.TestAddresses[i])
		assert.True(t, found)
		assert.Equal(t, uint64(i), accountIndex)

		for j := 0; j < len(tests.TestTokens)-i; j++ {
			tokenIndex, found := d.GetTokenIndex(tests.TestAddresses[i], tests.TestTokens[j])
			assert.True(t, found)
			assert.Equal(t, uint64(j), tokenIndex)
		}
	}
}

func TestNewDistributionWithData(t *testing.T) {
	distro, err := distribution.NewDistributionWithData(tests.TestJsonDistribution)
	assert.Nil(t, err)

	account, tokens, err := distro.Merklize()

	assert.Nil(t, err)
	assert.Len(t, account.Data, 1)
	addr := common.HexToAddress("0x0D6bA28b9919CfCDb6b233469Cc5Ce30b979e08E")
	assert.Len(t, tokens[addr].Data, 1)
}

func TestNewDistributionWithClaimDataLines(t *testing.T) {
	testClaimsStr := strings.Split(string(tests.TestClaims), "\n")

	earners := make([]*distribution.EarnerLine, 0)
	for _, e := range testClaimsStr {
		e = e
		earner := &distribution.EarnerLine{}
		err := json.Unmarshal([]byte(e), earner)
		assert.Nil(t, err)
		earners = append(earners, earner)
	}
	assert.Len(t, earners, 228)

	distro := distribution.NewDistribution()
	distro.LoadFromLines(earners)
}
