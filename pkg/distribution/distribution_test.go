package distribution_test

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/Layr-Labs/eigenlayer-payment-proofs/pkg/distribution"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var TestRootsString = []string{
	"1cd66d8c8bc9fff584d645acad9772fcb3a2e35b668195c1f4233b4afb6c5c08",
	"a69ce4e979a47455c488ed852c0ab8e0b73334ad2435af2d74396811b4b2457f",
	"278ce3f44e462061f1258054b3376cda6078f705263b9d6fa5eefdf5b65a4b37",
	"9311747f534e56811261e372ec5a4a11fcd59019d2f5d4b84e5464883d81a011",
	"4662c30b662f292447bb7e71e83e1700cc3a9ec7e8e00390d88f09d3f28faf1e",
}

// must be in alphabetical order
var TestAddresses = []common.Address{
	common.HexToAddress("0x05f7a45e049c96769360fafef7ccfc130dc22ab6"),
	common.HexToAddress("0x55ccb6ec92959052b9f1bf35b2cef438cf626aa5"),
	common.HexToAddress("0x7aadd3816216358a86aaca56728ca82abe9378af"),
	common.HexToAddress("0xdb5117dd6769e1a3442dd19f6bf89e2b8c2e011b"),
	common.HexToAddress("0xf924f84924421031c236c6f83727cae0c8ad13f2"),
}

// must be in alphabetical order
var TestTokens = []common.Address{
	common.HexToAddress("0x257601e63cc667ba6ad3561eb197f0edad4f96f7"),
	common.HexToAddress("0x50fba4307f9e10297bcda2c4380539814f965ce1"),
	common.HexToAddress("0x6fcc04913c0cd3eca196723a780bdb4b9aa14194"),
	common.HexToAddress("0x9fc9be8f24b23f5d12a53061ed5b96030cbb375c"),
	common.HexToAddress("0xdd78fcf0c0814218f9e8863142b904d7a04b7ae5"),
}

var TestAmountsString = []string{
	"1000000000000000001",
	"2000000000000000000",
	"300000000000021352135000000",
	"4000235235000000000000000",
	"5",
}

var TestAmountsBytes32 = []string{
	"0000000000000000000000000000000000000000000000000de0b6b3a7640001",
	"0000000000000000000000000000000000000000000000001bc16d674ec80000",
	"000000000000000000000000000000000000000000f82778965839e41a6bffc0",
	"000000000000000000000000000000000000000000034f152fc5cde26f038000",
	"0000000000000000000000000000000000000000000000000000000000000005",
}

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
	for i := 0; i < len(TestAddresses); i++ {
		for j := 0; j < len(TestTokens)-i; j++ {
			d.Set(TestAddresses[i], TestTokens[j], big.NewInt(int64(j+i+1)))
		}
	}

	return d
}

func GetCompleteTestDistribution() *distribution.Distribution {
	d := distribution.NewDistribution()

	for i := 0; i < len(TestAddresses); i++ {
		for j := 0; j < len(TestTokens); j++ {
			d.Set(TestAddresses[i], TestTokens[j], big.NewInt(int64(j+i+2)))
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

	err := d.Set(TestAddresses[1], TestTokens[0], big.NewInt(1))
	assert.NoError(t, err)

	err = d.Set(TestAddresses[0], TestTokens[0], big.NewInt(2))
	assert.ErrorIs(t, err, distribution.ErrAddressNotInOrder)

	amount1, found := d.Get(TestAddresses[1], TestTokens[0])
	assert.Equal(t, big.NewInt(1), amount1)
	assert.True(t, found)

	amount2, found := d.Get(TestAddresses[0], TestTokens[0])
	assert.Equal(t, big.NewInt(0), amount2)
	assert.False(t, found)
}

func TestSetTokensInNonAlphabeticalOrder(t *testing.T) {
	d := distribution.NewDistribution()

	err := d.Set(TestAddresses[0], TestTokens[1], big.NewInt(1))
	assert.NoError(t, err)

	err = d.Set(TestAddresses[0], TestTokens[0], big.NewInt(2))
	assert.ErrorIs(t, err, distribution.ErrTokenNotInOrder)

	amount1, found := d.Get(TestAddresses[0], TestTokens[1])
	assert.Equal(t, big.NewInt(1), amount1)
	assert.True(t, found)

	amount2, found := d.Get(TestAddresses[0], TestTokens[0])
	assert.Equal(t, big.NewInt(0), amount2)
	assert.False(t, found)
}

func TestGetUnset(t *testing.T) {
	d := distribution.NewDistribution()

	fetched, found := d.Get(TestAddresses[0], TestTokens[0])
	assert.Equal(t, big.NewInt(0), fetched)
	assert.False(t, found)
}

func TestEncodeAccountLeaf(t *testing.T) {
	for i := 0; i < len(TestAddresses); i++ {
		testRoot, _ := hex.DecodeString(TestRootsString[i])
		leaf := distribution.EncodeAccountLeaf(TestAddresses[i], testRoot)
		assert.Equal(t, distribution.EARNER_LEAF_SALT[0], leaf[0], "The first byte of the leaf should be EARNER_LEAF_SALT")
		assert.Equal(t, TestAddresses[i][:], leaf[1:21])
		assert.Equal(t, testRoot, leaf[21:])
	}
}

func TestEncodeTokenLeaf(t *testing.T) {
	for i := 0; i < len(TestTokens); i++ {
		testAmount, _ := new(big.Int).SetString(TestAmountsString[i], 10)
		leaf := distribution.EncodeTokenLeaf(TestTokens[i], testAmount)
		assert.Equal(t, distribution.TOKEN_LEAF_SALT[0], leaf[0], "The first byte of the leaf should be TOKEN_LEAF_SALT")
		assert.Equal(t, TestTokens[i][:], leaf[1:21])
		assert.Equal(t, TestAmountsBytes32[i], hex.EncodeToString(leaf[21:]))
	}
}

func TestGetAccountIndexBeforeMerklization(t *testing.T) {
	d := GetTestDistribution()

	accountIndex, found := d.GetAccountIndex(TestAddresses[1])
	assert.False(t, found)
	assert.Equal(t, uint64(0), accountIndex)
}

func TestGetTokenIndexBeforeMerklization(t *testing.T) {
	d := GetTestDistribution()

	tokenIndex, found := d.GetTokenIndex(TestAddresses[1], TestTokens[1])
	assert.False(t, found)
	assert.Equal(t, uint64(0), tokenIndex)
}

func TestMerklize(t *testing.T) {
	d := GetTestDistribution()

	accountTree, tokenTrees, err := d.Merklize()
	assert.NoError(t, err)

	// check the token trees
	assert.Len(t, tokenTrees, len(TestAddresses))
	for i := 0; i < len(tokenTrees); i++ {
		tokenTree, found := tokenTrees[TestAddresses[i]]
		assert.True(t, found)
		assert.Len(t, tokenTree.Data, len(TestTokens)-i)

		// check the data, that means the leafs are the same
		for j := 0; j < len(TestTokens)-i; j++ {
			leaf := tokenTree.Data[j]
			assert.Equal(t, distribution.EncodeTokenLeaf(TestTokens[j], big.NewInt(int64(j+i+1))), leaf)
		}
	}

	// check the account tree
	assert.Len(t, accountTree.Data, len(TestAddresses))
	for i := 0; i < len(TestAddresses); i++ {
		accountRoot := tokenTrees[TestAddresses[i]].Root()
		leaf := accountTree.Data[i]
		assert.Equal(t, distribution.EncodeAccountLeaf(TestAddresses[i], accountRoot), leaf)

		accountIndex, found := d.GetAccountIndex(TestAddresses[i])
		assert.True(t, found)
		assert.Equal(t, uint64(i), accountIndex)

		for j := 0; j < len(TestTokens)-i; j++ {
			tokenIndex, found := d.GetTokenIndex(TestAddresses[i], TestTokens[j])
			assert.True(t, found)
			assert.Equal(t, uint64(j), tokenIndex)
		}
	}
}

func TestNewDistributionWithData(t *testing.T) {
	jsonBytes := []byte(`{
		"0x0D6bA28b9919CfCDb6b233469Cc5Ce30b979e08E": {
			"0x1006dd1B8C3D0eF53489beD27577C75299F71473": 2000000000000000000
		}
	}`)

	distro, err := distribution.NewDistributionWithData(jsonBytes)
	assert.Nil(t, err)

	account, tokens, err := distro.Merklize()

	assert.Nil(t, err)
	assert.Len(t, account.Data, 1)
	addr := common.HexToAddress("0x0D6bA28b9919CfCDb6b233469Cc5Ce30b979e08E")
	assert.Len(t, tokens[addr].Data, 1)
}
